package domain

// stopwordLanguages are the languages DetectLanguage can tell apart, in the order it prefers on a tie.
//
//nolint:gochecknoglobals // a read-only table
var stopwordLanguages = []string{"en", "de", "nl", "fr", "es", "it", "pl"}

// stopwords are the most frequent function words per language: they start or end no keyword, sit
// inside none (except "and"), and their frequency tells a page's language. A word missing here
// only costs a slightly noisier keyword list.
//
//nolint:gochecknoglobals // a read-only table
var stopwords = map[string]map[string]struct{}{
	"en": set("a", "about", "after", "all", "also", "am", "an", "and", "any", "are", "as", "at", "be",
		"because", "been", "before", "being", "between", "both", "but", "by", "can", "could", "did", "do",
		"does", "doing", "down", "during", "each", "few", "for", "from", "further", "get", "had", "has",
		"have", "having", "he", "her", "here", "his", "how", "if", "in", "into", "is", "it", "its",
		"just", "may", "me", "more", "most", "much", "must", "my", "no", "nor", "not", "now", "of",
		"off", "on", "once", "only", "or", "other", "our", "ours", "out", "over", "own", "same", "see",
		"she", "should", "so", "some", "such", "than", "that", "the", "their", "them", "then", "there",
		"these", "they", "this", "those", "through", "to", "too", "under", "until", "up", "us", "use",
		"used", "very", "was", "we", "were", "what", "when", "where", "which", "while", "who", "why",
		"will", "with", "would", "you", "your", "yours", "read", "learn", "click", "here", "via",
		"offer", "offers", "provide", "provides", "find", "make", "makes", "made", "designed", "contact"),
	"de": set("aber", "alle", "allem", "allen", "aller", "als", "also", "am", "an", "auch", "auf", "aus",
		"bei", "beim", "bis", "da", "damit", "dann", "das", "dass", "dem", "den", "denen", "der", "des",
		"die", "dies", "diese", "diesem", "diesen", "dieser", "doch", "dort", "durch", "ein", "eine",
		"einem", "einen", "einer", "eines", "er", "es", "etwa", "für", "fur", "gegen", "hat", "hier",
		"ihr", "ihre", "ihrem", "ihren", "im", "in", "ist", "jede", "jeder", "kann", "können", "konnen",
		"mehr", "mit", "nach", "nicht", "noch", "nur", "ob", "oder", "ohne", "sehr", "sein", "seine",
		"sich", "sie", "sind", "so", "sowie", "über", "uber", "um", "und", "uns", "unser", "unsere",
		"unter", "vom", "von", "vor", "war", "was", "weitere", "wenn", "werden", "wie", "wir", "wird",
		"wo", "zu", "zum", "zur", "zwischen", "mehr", "erfahren", "hier", "bietet", "bieten", "finden",
		"sorgt", "anfrage", "kontakt"),
	"nl": set("aan", "al", "alle", "als", "altijd", "ben", "bij", "daar", "dan", "dat", "de", "deze",
		"die", "dit", "door", "dus", "een", "eens", "en", "er", "ge", "geen", "gaan", "gaat", "had",
		"heb", "hebben", "heeft", "hem", "het", "hier", "hij", "hoe", "hun", "ik", "in", "is", "ja",
		"je", "kan", "kun", "kunnen", "kunt", "maar", "me", "meer", "men", "met", "mij", "mijn", "na",
		"naar", "niet", "niets", "nog", "nu", "of", "om", "omdat", "onder", "ons", "onze", "ook", "op",
		"over", "te", "toch", "tot", "u", "uit", "uw", "van", "veel", "voor", "waar", "wanneer", "wat",
		"we", "wel", "werd", "wie", "wij", "wil", "wordt", "worden", "zal", "ze", "zelf", "zich", "zij",
		"zijn", "zit", "zo", "zoals", "zonder", "zou", "lees", "meer", "staat", "vormen", "vinden",
		"maken", "zorgen", "biedt", "bieden", "leveren", "brengen"),
	"fr": set("à", "a", "au", "aux", "avec", "ce", "ces", "cet", "cette", "d", "dans", "de", "des",
		"du", "elle", "en", "entre", "est", "et", "être", "etre", "eux", "il", "ils", "je", "l", "la",
		"le", "les", "leur", "leurs", "lui", "mais", "me", "même", "meme", "mes", "moi", "mon", "ne",
		"nos", "notre", "nous", "on", "ou", "où", "par", "pas", "plus", "pour", "qu", "que", "qui", "sa",
		"se", "ses", "son", "sont", "sur", "ta", "te", "tes", "toi", "ton", "tous", "tout", "toute",
		"très", "tres", "tu", "un", "une", "vos", "votre", "vous", "y"),
	"es": set("a", "al", "algo", "como", "con", "cual", "de", "del", "desde", "donde", "el", "ella",
		"en", "entre", "es", "esta", "este", "esto", "hasta", "la", "las", "le", "les", "lo", "los",
		"más", "mas", "muy", "ni", "no", "nos", "nuestra", "nuestras", "nuestro", "nuestros", "o", "para",
		"pero", "por", "que", "se", "ser", "si", "sin", "sobre", "su", "sus", "también", "tambien", "un",
		"una", "uno", "y", "ya"),
	"it": set("a", "ad", "al", "alla", "alle", "anche", "che", "ci", "come", "con", "da", "dal", "dalla",
		"degli", "dei", "del", "della", "delle", "di", "e", "ed", "gli", "i", "il", "in", "la", "le",
		"lo", "ma", "nei", "nel", "nella", "non", "nostri", "nostro", "o", "per", "più", "piu", "quale",
		"questo", "se", "si", "sono", "su", "sul", "sulla", "tra", "un", "una", "uno"),
	"pl": set("a", "aby", "ale", "bez", "być", "byc", "co", "czy", "dla", "do", "i", "ich", "jak", "jest",
		"jego", "jej", "już", "juz", "lub", "na", "nad", "nie", "o", "od", "oraz", "po", "pod", "przez",
		"przy", "się", "sie", "są", "sa", "ta", "tak", "te", "też", "tez", "to", "w", "we", "z", "za",
		"ze", "że", "ze"),
}

// fillerWords are the marketing words a phrase about belts often starts with ("every", "custom",
// "high-quality"), and the words of links and buttons ("view", "details"): they start or end no
// keyword, but do not tell a page's language.
//
//nolint:gochecknoglobals // a read-only table
var fillerWords = set(
	"every", "custom", "high-quality", "quality", "various", "different", "many", "best",
	"perfect", "ideal", "unique", "reliable", "durable", "innovative", "leading", "complete", "suitable",
	"range", "new", "great", "huidige", "eenmaal", "toegepaste", "langer",
	"viele", "verschiedene", "hochwertige", "hochwertigen", "individuelle", "neue",
	"verschillende", "hoogwaardige", "nieuwe",
	// Link and button text: "Conveyor belts details", "View flat wire".
	"details", "detail", "view", "more", "read", "learn", "download", "overview",
	"mehr", "weiterlesen", "meer", "bekijk", "lees", "voir",
)

func set(words ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}
