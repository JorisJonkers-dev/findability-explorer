package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// checkVersion is the competitor definition in use in these tests: the first stored version.
const checkVersion = 2

func TestTheLanguageModelDecidesAsTheLanguageModel(t *testing.T) {
	if domain.ModelDecider != "language model" {
		t.Errorf("decider %q", domain.ModelDecider)
	}
}

func TestASureVerdictDecides(t *testing.T) {
	withSite := domain.Candidate{Website: "https://www.ashworth.com/"}
	cases := []struct {
		name string
		v    domain.CompanyVerdict
		c    domain.Candidate
		want domain.CandidateStatus
		ok   bool
	}{
		{"a sure competitor with a website", domain.CompanyVerdict{Competitor: true, Sure: true}, withSite, domain.CandidateAdded, true},
		{"a sure competitor without one", domain.CompanyVerdict{Competitor: true, Sure: true}, domain.Candidate{}, domain.CandidatePending, false},
		{"a sure non-competitor", domain.CompanyVerdict{Sure: true}, domain.Candidate{}, domain.CandidateDismissed, true},
		{"an unsure verdict", domain.CompanyVerdict{Competitor: true}, withSite, domain.CandidatePending, false},
	}
	for _, tc := range cases {
		if got, ok := tc.v.Decides(tc.c); got != tc.want || ok != tc.ok {
			t.Errorf("%s: %s, %v", tc.name, got, ok)
		}
	}
}

func TestACandidateIsQuotedFromAFewAnswers(t *testing.T) {
	var sightings []domain.Sighting
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	for _, id := range ids {
		sightings = append(sightings, domain.Sighting{AnswerID: id, Name: "Ashworth", Provider: "openai"})
	}
	sightings = append(sightings,
		domain.Sighting{AnswerID: ids[0], Name: "Ashworth", Provider: "openai"},
		domain.Sighting{AnswerID: ids[3], Name: "Ashworth Bros", URL: "https://www.ashworth.com/", Provider: "openai"},
		domain.Sighting{AnswerID: ids[3], Name: "Ashworth", URL: "https://www.ashworth.com/", Provider: "openai"},
	)
	got := domain.Candidates(sightings, nil, nil, nil)
	if len(got) != 1 || !slices.Equal(got[0].Quoted, ids[:domain.MaxQuoted]) {
		t.Errorf("quoted = %+v", got)
	}
}

func TestSnippetsQuoteTheAnswersAroundTheName(t *testing.T) {
	long := strings.Repeat("word ", 50) + "the belts of ASHWORTH Bros are   stainless\n" + strings.Repeat("more ", 50)
	got := domain.Snippets([]string{
		"Try Ashworth for oven belts.",
		long,
		"Nothing about it here.",
		"Try Ashworth for oven belts.", // quoted already
		"Ashworth Bros and Ashworth again.",
	}, []string{"  ", "Ashworth Bros", "Ashworth"})
	if len(got) != 3 || got[0] != "Try Ashworth for oven belts." || got[2] != "Ashworth Bros and Ashworth again." {
		t.Fatalf("snippets = %q", got)
	}
	s := got[1]
	if !strings.HasPrefix(s, "…word w") || !strings.HasSuffix(s, "more more…") || !strings.Contains(s, "ASHWORTH Bros are stainless more") || len(s) > 400 {
		t.Errorf("long snippet = %q", s)
	}
	// Cut inside a word and a character: whole words and whole characters only.
	odd := strings.Repeat("é", 120) + " Ashworth " + strings.Repeat("ü", 120)
	if s := domain.Snippets([]string{odd}, []string{"Ashworth"}); len(s) != 1 || !strings.Contains(s[0], "Ashworth") ||
		!strings.HasPrefix(s[0], "…") || !strings.HasSuffix(s[0], "…") || strings.ContainsRune(s[0], '�') {
		t.Errorf("odd snippet = %q", s)
	}
}
