package lang_test

import (
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

// vocabulary is a small terminology map like the research's: terms in each focus language and in
// Spanish.
func vocabulary() lang.Vocabulary {
	return lang.NewVocabulary([]lang.Text{
		{Text: "Transportband", Language: "de"},
		{Text: "Drahtgurte", Language: "de"},
		{Text: "spiral belt / spiral conveyor belt", Language: "en-US"},
		{Text: "Spiral-Förderband", Language: "de"},
		{Text: "eye-link belt / eyelink belt", Language: "en"},
		{Text: "draadogenband", Language: "nl"},
		{Text: "rondloopband (RA)", Language: "nl"},
		{Text: "1.4301 stainless steel", Language: "en"},
		{Text: "cinta transportadora metálica", Language: "es"},
		{Text: "Inconel belt", Language: "en"},
		{Text: "ab", Language: "en"},
	})
}

// The phrases are what scans of competitors' sites found, most of them on pages in several
// languages.
func TestPossible(t *testing.T) {
	v := vocabulary()
	for phrase, want := range map[string][]string{
		// The words themselves, from the map and the word list.
		"conveyor belt":               {"en"},
		"eye link belts":              {"en"},
		"tug-belts cg-belts eye-flex": {"en"},
		"modular belts":               {"en"},
		"koel vries":                  {"nl"},
		"metalen banden":              {"nl"},
		"teilung und steigung":        {"de"},
		"spiral":                      {"en"}, // the list wins over a map that has it in German too
		"ovens":                       {"nl", "en"},
		"transportband":               {"nl", "de"}, // Dutch and German alike, though the map has it in German
		// Letters.
		"transportbänder":     {"de"},
		"bakkerijen":          {"nl"},
		"geïntegreerde":       {"nl"},
		"malla metàl":         {}, // another language
		"spiraldraht ø falls": {"de"},
		// Compounds and endings.
		"draadogenbanden":     {"nl"},
		"drahtfördergurte":    {"de"},
		"typische drahtgurte": {"de"},
		"spiraalband":         {"nl"}, // a band, Dutch by its stem
		"ofenband":            {"de"},
		"honingraatband":      {"nl"},
		"cleartracband":       {"nl", "de"}, // a joined compound is no English word
		"unibelt":             {"en"},
		"festigkeit":          {"de"},
		"duurzaamheid":        {"nl"},
		"capacity":            {"en"},
		"verbindingen":        {"nl", "de"},
		"filtration":          {"de", "en"},
		"plaatjes":            {"nl"},
		// Words that disagree: the most, the last word counting twice, and on a tie the last word.
		"oven transportband":         {"nl"},
		"transportband edelstahl":    {"de"},
		"tri-flex gurt":              {"de"},
		"flat-flex stabgeflechtgurt": {"de"},
		"wire mesh gurt":             {"de"},
		"balance weave baking band":  {"en"},
		"eyelink band":               {"en"},
		"eyelink transportband":      {"nl", "de"},
		"links en rechts":            {"nl"},
		"cinta belt":                 {"en"},
		"cinta transportadora":       {},
		// Nothing tells.
		"1.4301":       {"nl", "de", "en"},
		"t304 inconel": {"nl", "de", "en"},
		"transport":    {"nl", "de", "en"},
		"ra ab":        {"nl", "de", "en"},
		"":             {"nl", "de", "en"},
	} {
		if got := v.Possible(phrase); !slices.Equal(got, want) {
			t.Errorf("Possible(%q) = %q, want %q", phrase, got, want)
		}
	}
}

func TestPhrase(t *testing.T) {
	v := vocabulary()
	for phrase, want := range map[string]string{
		"conveyor belt":        "en",
		"Drahtgurte":           "de",
		"draadogenbanden":      "nl",
		"transportband":        lang.Unknown,
		"cinta transportadora": lang.Unknown,
		"1.4301":               lang.Unknown,
	} {
		if got := v.Phrase(phrase); got != want {
			t.Errorf("Phrase(%q) = %q, want %q", phrase, got, want)
		}
	}
	// With no map, the word list alone still tells.
	if got := (lang.Vocabulary{}).Phrase("drahtgurt"); got != "de" {
		t.Errorf("Phrase without a map = %q, want de", got)
	}
}

func TestTold(t *testing.T) {
	for tag, want := range map[string]bool{"en": true, "de-DE": true, "fr": true, "": false, "und": false, "UND": false} {
		if got := lang.Told(tag); got != want {
			t.Errorf("Told(%q) = %v, want %v", tag, got, want)
		}
	}
}
