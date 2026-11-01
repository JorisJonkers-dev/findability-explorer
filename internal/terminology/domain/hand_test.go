package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

var onTheList = []domain.Term{
	{Text: "Drahtösengliederband", Language: domain.LanguageDe, Origin: domain.OriginResearch},
	{Text: "spiral belt", Language: domain.LanguageEn, Origin: domain.OriginReview},
}

func TestNewHandTerm(t *testing.T) {
	got, err := domain.NewHandTerm(domain.Acceptance{Text: "  oven   belt ", Language: "en", Family: "eyelink", Tier: "application"}, onTheList)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "oven belt" || got.Language != domain.LanguageEn || got.Family != domain.FamilyEyelink || got.Tier != domain.TierApplication ||
		got.Origin != domain.OriginHand || got.UsedBy == nil || len(got.UsedBy) != 0 || got.Source != "" {
		t.Errorf("hand term = %+v", got)
	}
	for _, bad := range []struct {
		a    domain.Acceptance
		want error
	}{
		{domain.Acceptance{Text: "Spiral Belt", Language: "en", Family: "other", Tier: "broad"}, domain.ErrTermExists},
		{domain.Acceptance{Text: " ", Language: "en", Family: "other", Tier: "broad"}, domain.ErrInvalidTerm},
		{domain.Acceptance{Text: "belt", Language: "fr", Family: "other", Tier: "broad"}, domain.ErrUnknownValue},
	} {
		if _, err := domain.NewHandTerm(bad.a, onTheList); !errors.Is(err, bad.want) {
			t.Errorf("%+v: %v, want %v", bad.a, err, bad.want)
		}
	}
}

func TestWithoutTheRemovedTerms(t *testing.T) {
	removed := []domain.Removal{{Key: domain.TermKey{Text: "spiral belt", Language: domain.LanguageEn}, By: "joris", At: time.Now()}}
	if got := domain.WithoutRemoved(onTheList, removed); len(got) != 1 || got[0].Text != "Drahtösengliederband" {
		t.Errorf("without = %+v", got)
	}
	if got := domain.WithoutRemoved(onTheList, nil); len(got) != 2 {
		t.Errorf("nothing removed keeps %d", len(got))
	}
}

func TestFindTermIgnoresCaseAndAccents(t *testing.T) {
	got, ok := domain.FindTerm(onTheList, domain.TermKey{Text: "drahtosengliederband", Language: domain.LanguageDe})
	if !ok || got.Text != "Drahtösengliederband" {
		t.Errorf("found %+v, %v", got, ok)
	}
	if _, ok := domain.FindTerm(onTheList, domain.TermKey{Text: "spiral belt", Language: domain.LanguageDe}); ok {
		t.Error("a term in another language is not the same term")
	}
}
