package domain

// stopwordLanguages are the languages DetectLanguage can tell apart, in the order it prefers on a tie.
//
//nolint:gochecknoglobals // a read-only table
var stopwordLanguages = []string{"en", "de", "nl", "fr"}

// stopwords are the most frequent function words per language: they start or end no keyword, and
// their frequency tells a page's language. Short on purpose; a word missing here only costs a
// slightly noisier keyword list.
//
//nolint:gochecknoglobals // a read-only table
var stopwords = map[string]map[string]struct{}{
	"en": set("a", "about", "all", "also", "an", "and", "any", "are", "as", "at", "be", "been", "but",
		"by", "can", "do", "for", "from", "has", "have", "how", "if", "in", "into", "is", "it", "its",
		"more", "most", "no", "not", "of", "on", "or", "other", "our", "out", "over", "so", "such",
		"than", "that", "the", "their", "them", "then", "there", "these", "they", "this", "to", "up",
		"us", "was", "we", "were", "what", "when", "which", "who", "will", "with", "you", "your"),
	"de": set("aber", "als", "am", "an", "auch", "auf", "aus", "bei", "bis", "das", "dass", "dem",
		"den", "der", "des", "die", "durch", "ein", "eine", "einem", "einen", "einer", "es", "für",
		"hat", "ihr", "ihre", "im", "in", "ist", "mit", "nach", "nicht", "noch", "nur", "oder", "sich",
		"sie", "sind", "so", "über", "um", "und", "uns", "unsere", "unser", "von", "vor", "wie", "wir",
		"wird", "zu", "zum", "zur"),
	"nl": set("aan", "al", "als", "bij", "dan", "dat", "de", "die", "dit", "door", "een", "en", "er",
		"het", "hun", "in", "is", "je", "kan", "met", "na", "naar", "niet", "of", "om", "onze", "ook",
		"op", "over", "te", "tot", "u", "uit", "uw", "van", "voor", "wat", "wij", "wordt", "zijn", "zo"),
	"fr": set("au", "aux", "avec", "ce", "ces", "dans", "de", "des", "du", "elle", "en", "est", "et",
		"il", "la", "le", "les", "leur", "mais", "nos", "notre", "nous", "ou", "par", "pas", "pour",
		"qui", "sa", "se", "son", "sont", "sur", "un", "une", "vos", "votre", "vous"),
}

func set(words ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}
