package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

const robotsTxt = `# A site's rules
User-agent: GPTBot
Disallow: /

User-agent: *
Disallow: /private
Allow: /private/brochures # brochures are public
Disallow: /search*
Disallow: /cart$
Disallow:

User-agent: FindabilityExplorer
User-agent: SomeOtherBot
Disallow: /drafts
not a rule
`

func TestParseRobotsUsesTheNamedGroup(t *testing.T) {
	r := domain.ParseRobots(robotsTxt, "findabilityexplorer")
	if r.Allowed("/drafts/a") || !r.Allowed("/private") {
		t.Fatal("the FindabilityExplorer group does not apply")
	}
}

func TestParseRobotsFallsBackToTheStarGroup(t *testing.T) {
	r := domain.ParseRobots(robotsTxt, "Unknown")
	cases := map[string]bool{
		"/":                       true,
		"/private":                false,
		"/private/x":              false,
		"/private/brochures/a.pd": true,
		"/search?q=belt":          false,
		"/cart":                   false,
		"/drafts":                 true,
	}
	for path, want := range cases {
		if got := r.Allowed(path); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestEmptyRobotsAllowsEverything(t *testing.T) {
	if !(domain.Robots{}).Allowed("/anything") || !domain.ParseRobots("", "x").Allowed("/") {
		t.Fatal("no rules must allow everything")
	}
	tie := domain.ParseRobots("User-agent: *\nDisallow: /a\nAllow: /a\n", "x")
	if !tie.Allowed("/a") {
		t.Fatal("on a tie the allow rule wins")
	}
}

func TestRobotsWildcards(t *testing.T) {
	r := domain.ParseRobots("User-agent: *\nDisallow: /*cHash\nDisallow: /*.pdf$\nDisallow: /a*b*c$\nDisallow: /exact$\n", "x")
	cases := map[string]bool{
		"/":                    true,
		"/de/produkte":         true,
		"/page?cHash=12":       false,
		"/files/brochure.pdf":  false,
		"/files/brochure.pdfx": true,
		"/a-b-c":               false,
		"/a-b-cd":              true,
		"/a-b":                 true,
		"/exact":               false,
		"/exactly":             true,
	}
	for path, want := range cases {
		if got := r.Allowed(path); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", path, got, want)
		}
	}
}
