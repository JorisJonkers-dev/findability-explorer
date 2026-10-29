package domain_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

var (
	twente  = domain.ScopeInfo{ID: domain.ScopeTwente, Language: "nl", Region: "Twente", Name: "Twente (Dutch)"}
	germany = domain.ScopeInfo{ID: domain.ScopeGermany, Language: "de", Region: "Germany", Name: "Germany (German)"}
	english = domain.ScopeInfo{ID: domain.ScopeEnglish, Language: "en", Name: "English, no region"}
)

func published(version int, at time.Time, questions ...domain.Question) domain.PublishedSet {
	return domain.PublishedSet{Version: version, PublishedAt: at, PublishedBy: "import", Note: "Note.", Questions: questions}
}

func TestAPublishedSetAsksItsQuestionsInTheirScopesLanguage(t *testing.T) {
	set := published(1, time.Time{},
		domain.Question{ID: "eyelink-supplier-twente", Family: domain.FamilyEyelink, Intent: domain.IntentProduct, Scope: twente, Text: "Wie levert oogjesbanden?"},
		domain.Question{ID: "eyelink-problem-germany", Family: domain.FamilyEyelink, Intent: domain.IntentProblem, Scope: germany, Text: "Unser Band reißt."},
		domain.Question{ID: "other-application-english", Family: domain.FamilyOther, Intent: domain.IntentApplication, Scope: english, Text: "Which belt?", English: ""},
	).PromptSet()
	want := []domain.Prompt{
		{ID: "eyelink-supplier-twente", Family: domain.FamilyEyelink, Kind: domain.KindSupplier, Scope: domain.ScopeTwente, Language: "nl", Text: "Wie levert oogjesbanden?"},
		{ID: "eyelink-problem-germany", Family: domain.FamilyEyelink, Kind: domain.KindProblem, Scope: domain.ScopeGermany, Language: "de", Text: "Unser Band reißt."},
		{ID: "other-application-english", Family: domain.FamilyOther, Kind: domain.KindApplication, Scope: domain.ScopeEnglish, Language: "en", Text: "Which belt?"},
	}
	if set.Version != 1 || len(set.Prompts) != len(want) {
		t.Fatalf("set = %+v", set)
	}
	for i, p := range set.Prompts {
		if p != want[i] {
			t.Errorf("prompt %d = %+v, want %+v", i, p, want[i])
		}
	}
}

func TestTheKindAnAnswerRecordsIsTheBuyerIntentWithProductAsSupplier(t *testing.T) {
	for intent, kind := range map[domain.Intent]domain.Kind{
		domain.IntentProduct: domain.KindSupplier, domain.IntentApplication: domain.KindApplication, domain.IntentProblem: domain.KindProblem,
		domain.IntentSpec: "spec", domain.IntentBroad: "broad",
	} {
		if got := domain.KindOf(intent); got != kind {
			t.Errorf("kind of %s = %s, want %s", intent, got, kind)
		}
	}
}

func TestTheNewestPublishedSetIsTheHighestVersion(t *testing.T) {
	if _, ok := domain.Newest(nil); ok {
		t.Error("a newest set when none is published")
	}
	sets := []domain.PublishedSet{published(1, time.Time{}), published(3, time.Time{}), published(2, time.Time{})}
	if set, ok := domain.Newest(sets); !ok || set.Version != 3 {
		t.Errorf("newest = %+v, %v", set, ok)
	}
}

func TestTheVersionsAreListedOldestFirstWithWhatTheyAsk(t *testing.T) {
	first, second := time.Date(2026, 10, 26, 15, 4, 51, 0, time.UTC), time.Date(2026, 10, 27, 6, 35, 33, 0, time.UTC)
	q := domain.Question{ID: "q"}
	sets := []domain.PublishedSet{published(2, second, q, q), published(1, first, q, q, q)}
	sets[1].Note = "Version one."
	v := domain.VersionsOf(sets)
	want := []domain.Version{{Number: 1, PublishedAt: first, Questions: 3, Note: "Version one."}, {Number: 2, PublishedAt: second, Questions: 2, Note: "Note."}}
	if len(v) != 2 || v[0] != want[0] || v[1] != want[1] {
		t.Errorf("versions = %+v", v)
	}
	if v := domain.VersionsOf(nil); v == nil || len(v) != 0 {
		t.Errorf("no sets = %#v, want an empty list", v)
	}
}
