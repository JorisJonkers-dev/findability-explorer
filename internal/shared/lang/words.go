package lang

import "strings"

// The word list a phrase's language is told by (see Vocabulary.Possible), in one place. Each entry
// is folded (package fold), so "Fördergurt" is listed as "fordergurt"; letters only one language
// writes (ä, ö, ü, ß, ij) are told before folding and need no entry.
//
// A word goes here when the map's own terms do not settle it: a word in common use the map lacks
// ("koel", "teilung"), or one the map holds in one language that is just as much another's
// ("transportband" is Dutch and German, "oven" Dutch and English). An entry listing every focus
// language says nothing, so words like "transport" or "links" are left out.

// cueWords maps a folded word to the focus languages it is a word of, by the codes they are
// written as in the rows below.
//
//nolint:gochecknoglobals // a read-only table
var cueWords = table(map[string]string{
	"en": "and with for belt belting conveyor wire mesh chain drive eye weave woven balanced balance " +
		"compound stainless steel metal spiral flat flex rod grid honeycomb turn curve straight running " +
		"baking bakery cooling cooler freezer freezing furnace dryer drying washing cleaner cleaning " +
		"food processing industry industrial surface high heat temperature sprocket wheel edge pitch " +
		"width thickness round flattened solution product application glass tempering hygienic modular " +
		"link ladder heavy duty light load carrying transfer side guard plate cake instruction " +
		"construction terminal engineered custom",
	"de": "und mit gurt gurte gurten draht geflecht geflechte edelstahl stahl metall spirale ofen backofen " +
		"anlage glieder ose osen wabe waben kette ketten antrieb breite runddraht flachdraht stab stabe " +
		"kurve kurven kurvengangig durchlauf trocknung werkstoff werkstuck typische gefertigt gewebt " +
		"hochbelastbares transportelement teilung steigung falls zubehor lebensmittel kunststoff " +
		"gelenk forderband fordergurt forderer",
	"nl": "en met voor van banden draad draden gaas staal roestvast metalen spiraal koel koeling vries " +
		"vriezen diepvries toepassing toepassingen aandrijving ketting kettingen staaf staven vlak breed " +
		"brede gewikkelde gevlochten directe aanraking honingraat ogen oog levensmiddelen verwerking " +
		"drogen koelen waardoor bestaan wisselend",
	"de nl": "transportband spiralen industrie temperaturen kleine riemen rechte rechts",
	"en nl": "oven opening direct loop plastic diameter open",
	// Words every focus language uses alike, which the map holds in one: they tell nothing. A
	// "band" is a baking band in English too; joined into one word it is not (see compound).
	"en de nl": "band transport links radius inconel nichrome aisi hcht filter maximum minimum",
})

// cueEndings maps a folded word ending to the focus languages a word with it is in, for a word
// neither the list nor the map knows: German "teilung", Dutch "toepassingen", English "capacity".
//
//nolint:gochecknoglobals // a read-only table
var cueEndings = table(map[string]string{
	"de":    "ung ungen keit heit schaft",
	"nl":    "heid heden tje tjes baar schap",
	"de nl": "ingen",
	"en":    "ity ness ings",
	"en de": "tion",
})

// table turns rows of space-separated words, keyed by space-separated language codes, into a set
// of languages per word.
func table(rows map[string]string) map[string]set {
	out := map[string]set{}
	for codes, words := range rows {
		var s set
		for _, code := range strings.Fields(codes) {
			s |= bit(code)
		}
		for _, w := range strings.Fields(words) {
			out[w] = s
		}
	}
	return out
}
