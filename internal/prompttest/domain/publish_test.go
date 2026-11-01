package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

var (
	at     = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	lyon   = domain.ScopeInfo{ID: "fr-lyon", Language: "fr", Region: "Lyon", Name: "French, Lyon"}
	scopes = []domain.ScopeInfo{twente, germany, english}
)

func base() []domain.PublishedSet {
	return []domain.PublishedSet{
		published(1, at.Add(-48*time.Hour),
			domain.Question{ID: "eyelink-application-twente", Family: domain.FamilyEyelink, Intent: domain.IntentApplication, Scope: twente, Text: "Oud?"},
			domain.Question{ID: "old-question", Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: germany, Text: "Alt?"},
		),
		published(2, at.Add(-24*time.Hour),
			domain.Question{ID: "eyelink-application-twente", Family: domain.FamilyEyelink, Intent: domain.IntentApplication, Scope: twente, Text: "Welke band?", English: "Which belt?"},
			domain.Question{ID: "other-application-english", Family: domain.FamilyOther, Intent: domain.IntentApplication, Scope: english, Text: "Which belt?"},
		),
	}
}

func keep() []domain.DraftQuestion {
	return []domain.DraftQuestion{
		{ID: "eyelink-application-twente", Family: domain.FamilyEyelink, Intent: domain.IntentApplication, Scope: domain.ScopeTwente, Text: "Welke band?", English: "Which belt?"},
		{ID: "other-application-english", Family: domain.FamilyOther, Intent: domain.IntentApplication, Scope: domain.ScopeEnglish, Text: "Which belt?"},
	}
}

func TestPublishMakesTheNextVersionWithANewScopeAndFreshIDs(t *testing.T) {
	d := domain.Draft{
		Questions: append(keep()[1:],
			domain.DraftQuestion{Family: domain.FamilyEyelink, Intent: domain.IntentApplication, Scope: "fr-lyon", Text: "  Quelle bande ?  ", English: "Which belt?"},
			domain.DraftQuestion{Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: domain.ScopeGermany, Text: "Welches Band?"},
			domain.DraftQuestion{Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: domain.ScopeGermany, Text: "Noch eins?"},
		),
		Scopes:       []domain.ScopeInfo{lyon},
		RestoredFrom: 1,
	}
	d.Questions[0].Text = "Which belt, really?"
	set, added, err := domain.Publish(base(), scopes, d, "joris", at)
	if err != nil {
		t.Fatal(err)
	}
	if set.Version != 3 || set.PublishedBy != "joris" || !set.PublishedAt.Equal(at) {
		t.Fatalf("set = %+v", set)
	}
	ids := []string{}
	for _, q := range set.Questions {
		ids = append(ids, q.ID)
	}
	want := []string{"other-application-english", "eyelink-application-fr-lyon", "other-problem-germany", "other-problem-germany-2"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("ids = %v, want %v", ids, want)
	}
	if set.Questions[1].Text != "Quelle bande ?" || set.Questions[1].Scope != lyon {
		t.Errorf("question = %+v", set.Questions[1])
	}
	if len(added) != 1 || added[0] != lyon {
		t.Errorf("added scopes = %+v", added)
	}
	if want := "Restores version 1. Adds 3 questions, changes 1, and removes 1. Adds the scope French, Lyon."; set.Note != want {
		t.Errorf("note = %q, want %q", set.Note, want)
	}
}

func TestPublishNeverReusesAnIDAVersionHadAndStartsAtOne(t *testing.T) {
	d := domain.Draft{Questions: []domain.DraftQuestion{{Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: domain.ScopeGermany, Text: "Alt?"}}}
	sets := []domain.PublishedSet{published(1, at, domain.Question{ID: "other-problem-germany", Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: germany, Text: "x"})}
	set, _, err := domain.Publish(sets, scopes, d, "joris", at)
	if err != nil || set.Questions[0].ID != "other-problem-germany-2" {
		t.Fatalf("set = %+v, err = %v", set, err)
	}
	first, _, err := domain.Publish(nil, scopes, d, "joris", at)
	if err != nil || first.Version != 1 || first.Questions[0].ID != "other-problem-germany" {
		t.Fatalf("first = %+v, err = %v", first, err)
	}
	if first.Note != "Adds 1 question, changes none, and removes none." {
		t.Errorf("note = %q", first.Note)
	}
}

func TestPublishSaysWhenNothingChanged(t *testing.T) {
	if _, _, err := domain.Publish(base(), scopes, domain.Draft{Questions: keep()}, "joris", at); !errors.Is(err, domain.ErrNothingChanged) {
		t.Fatalf("err = %v", err)
	}
	reordered := []domain.DraftQuestion{keep()[1], keep()[0]}
	if _, _, err := domain.Publish(base(), scopes, domain.Draft{Questions: reordered}, "joris", at); err != nil {
		t.Errorf("a new order is a change, err = %v", err)
	}
	two := domain.Draft{Questions: keep()[:1]}
	set, _, err := domain.Publish(base(), scopes, two, "joris", at)
	if err != nil || set.Note != "Adds none, changes none, and removes 1." {
		t.Errorf("set = %+v, err = %v", set, err)
	}
}

func TestPublishNamesSeveralNewScopesAndKeepsTheNoteShort(t *testing.T) {
	paris := domain.ScopeInfo{ID: "fr", Language: "fr", Name: "French, no region"}
	d := domain.Draft{Questions: keep(), Scopes: []domain.ScopeInfo{lyon, paris}}
	set, _, err := domain.Publish(base(), scopes, d, "joris", at)
	if err != nil || !strings.HasSuffix(set.Note, "Adds the scopes French, Lyon; French, no region.") {
		t.Fatalf("set = %+v, err = %v", set, err)
	}
	many := []domain.ScopeInfo{}
	for i := range 6 {
		name := strings.Repeat(string(rune('a'+i)), 60)
		many = append(many, domain.ScopeInfo{ID: domain.Scope("s" + string(rune('a'+i))), Language: "fr", Name: name})
	}
	long, _, err := domain.Publish(base(), scopes, domain.Draft{Questions: keep(), Scopes: many}, "joris", at)
	if err != nil || len([]rune(long.Note)) != 300 || !strings.HasSuffix(long.Note, "…") {
		t.Fatalf("note = %q (%d), err = %v", long.Note, len([]rune(long.Note)), err)
	}
}

func TestPublishCutsALongBaseIDDown(t *testing.T) {
	long := domain.ScopeInfo{ID: domain.Scope("x" + strings.Repeat("y", 39)), Language: "fr", Name: "Long"}
	d := domain.Draft{Questions: []domain.DraftQuestion{{Family: domain.FamilyTriFlex, Intent: domain.IntentApplication, Scope: long.ID, Text: "Q?"}}, Scopes: []domain.ScopeInfo{long}}
	set, _, err := domain.Publish(nil, nil, d, "joris", at)
	if err != nil || len(set.Questions[0].ID) > 70 {
		t.Fatalf("set = %+v, err = %v", set, err)
	}
}

func TestPublishRefusesADraftItCannotAsk(t *testing.T) {
	q := func(change func(*domain.DraftQuestion)) domain.Draft {
		d := domain.Draft{Questions: keep()}
		change(&d.Questions[0])
		return d
	}
	withScope := func(s domain.ScopeInfo) domain.Draft {
		return domain.Draft{Questions: keep(), Scopes: []domain.ScopeInfo{s}}
	}
	tooMany := make([]domain.DraftQuestion, domain.MaxQuestions+1)
	cases := map[string]domain.Draft{
		"no questions":       {},
		"too many questions": {Questions: tooMany},
		"bad scope id":       withScope(domain.ScopeInfo{ID: "Fr Lyon", Language: "fr", Name: "x"}),
		"bad language":       withScope(domain.ScopeInfo{ID: "fr", Language: "fra", Name: "x"}),
		"long region":        withScope(domain.ScopeInfo{ID: "fr", Language: "fr", Region: strings.Repeat("r", 101), Name: "x"}),
		"no name":            withScope(domain.ScopeInfo{ID: "fr", Language: "fr", Name: " "}),
		"scope exists":       withScope(domain.ScopeInfo{ID: domain.ScopeTwente, Language: "nl", Name: "Again"}),
		"name exists":        withScope(domain.ScopeInfo{ID: "nl2", Language: "nl", Name: twente.Name}),
		"bad id":             q(func(d *domain.DraftQuestion) { d.ID = "Not An Id" }),
		"twice":              q(func(d *domain.DraftQuestion) { d.ID = "other-application-english" }),
		"bad family":         q(func(d *domain.DraftQuestion) { d.Family = "plastic" }),
		"bad intent":         q(func(d *domain.DraftQuestion) { d.Intent = domain.IntentBroad }),
		"unknown scope":      q(func(d *domain.DraftQuestion) { d.Scope = "mars" }),
		"empty text":         q(func(d *domain.DraftQuestion) { d.Text = "   " }),
		"long text":          q(func(d *domain.DraftQuestion) { d.Text = strings.Repeat("a", 1001) }),
		"long english":       q(func(d *domain.DraftQuestion) { d.English = strings.Repeat("a", 1001) }),
	}
	for name, d := range cases {
		_, _, err := domain.Publish(base(), scopes, d, "joris", at)
		var de domain.DraftError
		if !errors.As(err, &de) || de.Error() == "" {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
