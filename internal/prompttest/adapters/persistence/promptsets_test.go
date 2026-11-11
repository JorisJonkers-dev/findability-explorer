package persistence_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/db"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// version1 is Prompt set version 1 as rebuilt from commit de6f134, with the mapping the import
// applies: the supplier kind is the product intent, and the scopes keep their ids.
func version1(t *testing.T) domain.PublishedSet {
	t.Helper()
	raw, err := os.ReadFile("testdata/prompt-set-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Questions []struct{ ID, Family, Kind, Scope, Language, Text string }
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	set := domain.PublishedSet{
		Version: 1, PublishedAt: time.Date(2026, 10, 26, 15, 4, 51, 0, time.UTC), PublishedBy: "import",
		Note: "A supplier, an application, and a problem question for each belt family, in six scopes.",
	}
	intent := map[string]domain.Intent{"supplier": domain.IntentProduct, "application": domain.IntentApplication, "problem": domain.IntentProblem}
	for _, q := range file.Questions {
		scope := domaintest.Scope(domain.Scope(q.Scope))
		if scope.Language != q.Language {
			t.Fatalf("%s is in %s, its scope in %s", q.ID, q.Language, scope.Language)
		}
		set.Questions = append(set.Questions, domain.Question{
			ID: q.ID, Family: domain.Family(q.Family), Intent: intent[q.Kind], Scope: scope, Text: q.Text,
		})
	}
	return set
}

func TestTheImportStoresVersionsOneAndTwoWithTheMapping(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	sets, err := repo.PromptSets(t.Context())
	if err != nil || len(sets) != 2 {
		t.Fatalf("sets = %d, %v", len(sets), err)
	}
	for i := range sets {
		sets[i].PublishedAt = sets[i].PublishedAt.UTC()
	}
	// Version 2 is what the code asked before the Prompt set moved to the database.
	if want := domaintest.Version2(); !reflect.DeepEqual(sets[0], want) {
		t.Errorf("version 2 =\n%+v\nwant\n%+v", sets[0], want)
	}
	v1 := version1(t)
	if len(v1.Questions) != 90 || len(sets[1].Questions) != 90 {
		t.Fatalf("version 1 has %d questions, the source %d", len(sets[1].Questions), len(v1.Questions))
	}
	if !reflect.DeepEqual(sets[1], v1) {
		for i, q := range sets[1].Questions {
			if q != v1.Questions[i] {
				t.Errorf("version 1 question %d = %+v, want %+v", i, q, v1.Questions[i])
			}
		}
		t.Errorf("version 1 = %+v %s %s", sets[1].PublishedAt, sets[1].PublishedBy, sets[1].Note)
	}
	products, regions := 0, map[string]int{}
	for _, q := range sets[1].Questions {
		if q.Intent == domain.IntentProduct {
			products++
		}
		regions[q.Scope.Region]++
	}
	if products != 30 || regions["Netherlands"] != 15 || regions["Germany"] != 15 || regions["Twente"] != 15 || regions[""] != 45 {
		t.Errorf("version 1 has %d product questions, by region %v", products, regions)
	}
}

func TestTheScopesAreALanguageWithAnOptionalRegion(t *testing.T) {
	t.Parallel()
	scopes, err := repository(t).PromptScopes(t.Context())
	if err != nil || len(scopes) != 6 {
		t.Fatalf("scopes = %+v, %v", scopes, err)
	}
	// By name.
	names := []string{"Dutch, no region", "English, no region", "German, no region", "Germany (German)", "Netherlands (Dutch)", "Twente (Dutch)"}
	for i, s := range scopes {
		if s.Name != names[i] || s != domaintest.Scope(s.ID) {
			t.Errorf("scope %d = %+v", i, s)
		}
	}
}

// Every imported question can be asked: its answers fit the answers' kind and scope.
func TestAnAnswerToEveryImportedQuestionIsStored(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	sets, err := repo.PromptSets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, set := range sets {
		r := run(domain.TriggerManual)
		r.PromptSetVersion = set.Version
		if err := repo.CreateRun(ctx, r, "a", at); err != nil {
			t.Fatal(err)
		}
		for _, p := range set.PromptSet().Prompts {
			a := domain.Answer{ID: uuid.New(), RunID: r.ID, Prompt: p, PromptSetVersion: set.Version, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeWeb, At: at}
			if err := repo.AddAnswer(ctx, a); err != nil {
				t.Fatalf("version %d, %s: %v", set.Version, p.ID, err)
			}
		}
		if err := repo.FinishRun(ctx, r.ID, domain.RunDone, "", at); err != nil {
			t.Fatal(err)
		}
		answers, err := repo.Answers(ctx, r.ID, true)
		if err != nil || len(answers) != len(set.Questions) {
			t.Fatalf("version %d: %d answers, %v", set.Version, len(answers), err)
		}
		asked := map[string]domain.Prompt{}
		for _, p := range set.PromptSet().Prompts {
			asked[p.ID] = p
		}
		for _, a := range answers {
			if a.Prompt != asked[a.Prompt.ID] {
				t.Errorf("version %d: answered %+v, asked %+v", set.Version, a.Prompt, asked[a.Prompt.ID])
			}
		}
	}
}

// The migrations run on a database migrated past them, so the test holds when later ones are added.
func TestTheImportRunsAgainWithoutHarm(t *testing.T) {
	t.Parallel()
	url := pgtest.URL(t)
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	for _, name := range []string{"00031_prompt_sets.sql", "00032_import_prompt_sets.sql"} {
		migration, err := fs.ReadFile(db.Migrations, "migrations/"+name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("%s again: %v", name, err)
		}
	}
	store, err := pg.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	sets, err := persistence.New(store).PromptSets(ctx)
	if err != nil || len(sets) != 2 || len(sets[0].Questions) != 20 || len(sets[1].Questions) != 90 {
		t.Errorf("after the import ran again: %d sets, %v", len(sets), err)
	}
}
