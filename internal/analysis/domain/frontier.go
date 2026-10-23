package domain

import (
	"container/heap"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Link is an address the crawl found: in an anchor, as a language version a page declares, or in
// a sitemap.
type Link struct {
	URL *url.URL
	// Text is the anchor's text; empty for a language version or a sitemap entry.
	Text string
	// Alternate is true for a language version of the page (<link rel="alternate" hreflang>).
	Alternate bool
}

// Score bonuses and penalties a link earns from the words of its address and anchor text.
const (
	// sectionBonus is for a word naming a products, applications or specifications section.
	sectionBonus = 2.0
	// offTopicPenalty is for a word naming news, contact, legal, careers and the like: pages that
	// say nothing about belts and are fetched last.
	offTopicPenalty = 8.0
	// alternateBonus lifts a language version just above the page that declares it, so every
	// language of a product page is reached.
	alternateBonus = 0.5
	// depthPenalty is per path segment, so on a tie the shallower page comes first.
	depthPenalty = 0.1
)

// sectionWords name the parts of a site where products, their applications and their
// specifications are, in the languages of the map, folded.
func sectionWords() []string {
	return []string{
		"product", "products", "produkt", "produkte", "produkten", "producten", "produit", "produits",
		"prodotti", "prodotto", "productos", "producto", "produkty", "produkter",
		"application", "applications", "anwendung", "anwendungen", "toepassing", "toepassingen",
		"industries", "industry", "branchen", "industrie", "sectors", "markets",
		"specification", "specifications", "technical", "technische", "technisch", "datasheet",
		"belts", "belt", "band", "bander", "banden", "gurte", "foerderband", "transportband",
		"conveyor", "conveyors", "mesh", "wire", "draht",
	}
}

// offTopicWords name the parts of a site that say nothing about belts, folded.
func offTopicWords() []string {
	return []string{
		"news", "nieuws", "aktuelles", "neuigkeiten", "actualites", "notizie", "noticias", "blog",
		"press", "presse", "pers", "events", "event", "messe", "messen", "beurs", "beurzen",
		"contact", "kontakt", "contatti", "contacto", "kontakty", "impressum", "imprint", "colofon",
		"privacy", "datenschutz", "legal", "disclaimer", "agb", "terms", "conditions", "voorwaarden",
		"cookie", "cookies", "career", "careers", "karriere", "jobs", "job", "vacatures", "vacature",
		"werken", "stellenangebote", "login", "cart", "account", "search", "suche", "zoeken",
		"sitemap", "team", "history", "geschichte", "geschiedenis", "downloads", "download",
	}
}

// LinkScorer ranks the links of a crawl: words of the belt vocabulary in a link's address and
// anchor text raise it, words naming a products or applications section raise it a little, and
// words naming news, contact, legal or careers pages sink it.
type LinkScorer struct {
	lex     Lexicon
	section map[string]bool
	off     map[string]bool
}

// NewLinkScorer returns a LinkScorer judging by lex.
func NewLinkScorer(lex Lexicon) LinkScorer {
	return LinkScorer{lex: lex, section: wordSet(sectionWords()), off: wordSet(offTopicWords())}
}

func wordSet(words []string) map[string]bool {
	out := make(map[string]bool, len(words))
	for _, w := range words {
		out[w] = true
	}
	return out
}

// Score is how early a link should be fetched: higher first. Each distinct word counts once.
func (s LinkScorer) Score(u *url.URL, anchor string) float64 {
	words := linkWords(u, anchor)
	score, section, off := 0.0, false, false
	for _, w := range words {
		score += s.lex.wordWeight(w)
		section = section || s.section[w]
		off = off || s.off[w]
	}
	if section {
		score += sectionBonus
	}
	if off {
		score -= offTopicPenalty
	}
	return score - depthPenalty*float64(depth(u))
}

// linkWords are the distinct folded words of a link's path and anchor text. A path splits at
// slashes, hyphens, underscores and dots, so /de/produkte/drahtoesen-gliederband reads as words.
func linkWords(u *url.URL, anchor string) []string {
	p := u.Path
	if unescaped, err := url.PathUnescape(p); err == nil {
		p = unescaped
	}
	p = strings.TrimSuffix(p, path.Ext(p))
	raw := strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '-' || r == '_' || r == '.' })
	var out []string
	for _, w := range append(strings.Fields(fold.Fold(strings.Join(raw, " "))), foldedWords(anchor)...) {
		for _, part := range strings.Split(fold.Fold(w), "-") {
			if part != "" && !slices.Contains(out, part) {
				out = append(out, part)
			}
		}
	}
	return out
}

func depth(u *url.URL) int {
	return len(strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' }))
}

// maxFrontier is the most addresses a crawl keeps waiting.
const maxFrontier = 2000

// Frontier is a crawl's queue of same-site pages not yet visited, highest score first and, among
// equal scores, in the order they were found.
type Frontier struct {
	site    string
	scorer  LinkScorer
	seen    map[string]bool
	fetched map[string]bool
	queue   frontierHeap
	next    int
}

// NewFrontier returns an empty frontier for the site of host ("www." aside).
func NewFrontier(host string, scorer LinkScorer) *Frontier {
	return &Frontier{site: SiteOf(host), scorer: scorer, seen: map[string]bool{}, fetched: map[string]bool{}}
}

// Push queues each link that is an http(s) page of the site not seen before. A language version
// scores at least as high as the page that declares it (parentScore), plus a little.
func (f *Frontier) Push(links []Link, parentScore float64) {
	for _, l := range links {
		u := Clean(l.URL)
		if u == nil || SiteOf(u.Hostname()) != f.site || f.seen[Key(u)] || f.queue.Len() >= maxFrontier {
			continue
		}
		f.seen[Key(u)] = true
		score := f.scorer.Score(u, l.Text)
		if l.Alternate {
			score = max(score, parentScore) + alternateBonus
		}
		heap.Push(&f.queue, frontierItem{url: u, score: score, order: f.next})
		f.next++
	}
}

// Seen marks u visited without queueing it, as the start page is.
func (f *Frontier) Seen(u *url.URL) { f.seen[Key(u)] = true }

// Fetched records that the page at address (after redirects) was fetched, and reports whether it
// is the first fetch of that page: two links may redirect to one page.
func (f *Frontier) Fetched(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return true
	}
	k := Key(u)
	if f.fetched[k] {
		return false
	}
	f.fetched[k], f.seen[k] = true, true
	return true
}

// Pop returns the next address and its score, or nil when the frontier is empty.
func (f *Frontier) Pop() (*url.URL, float64) {
	if f.queue.Len() == 0 {
		return nil, 0
	}
	it := heap.Pop(&f.queue).(frontierItem)
	return it.url, it.score
}

// Len is how many addresses wait.
func (f *Frontier) Len() int { return f.queue.Len() }

type frontierItem struct {
	url   *url.URL
	score float64
	order int
}

type frontierHeap []frontierItem

func (h frontierHeap) Len() int { return len(h) }
func (h frontierHeap) Less(i, j int) bool {
	if h[i].score != h[j].score {
		return h[i].score > h[j].score
	}
	return h[i].order < h[j].order
}
func (h frontierHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *frontierHeap) Push(x any)   { *h = append(*h, x.(frontierItem)) }

func (h *frontierHeap) Pop() any {
	old := *h
	it := old[len(old)-1]
	*h = old[:len(old)-1]
	return it
}

// skippedExtension reports whether p names a file type that is never an HTML page.
func skippedExtension(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".css", ".js", ".json",
		".xml", ".gz", ".zip", ".mp4", ".mp3", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".dwg", ".step", ".stp":
		return true
	default:
		return false
	}
}

// Clean returns l without fragment and query, or nil when it is not an http(s) page link. Query
// strings are dropped because they mostly spell filters and sessions that repeat the same page.
func Clean(l *url.URL) *url.URL {
	if l == nil || (l.Scheme != "http" && l.Scheme != "https") || l.User != nil || skippedExtension(l.Path) {
		return nil
	}
	c := *l
	c.Fragment, c.RawQuery, c.RawFragment, c.ForceQuery = "", "", "", false
	if c.Path == "" {
		c.Path = "/"
	}
	return &c
}

// SiteOf is the host without a leading "www.", so www.example.com and example.com are one site.
func SiteOf(host string) string { return strings.TrimPrefix(strings.ToLower(host), "www.") }

// Key identifies a page of a site regardless of "www." and a final slash.
func Key(u *url.URL) string { return SiteOf(u.Hostname()) + strings.TrimSuffix(u.EscapedPath(), "/") }
