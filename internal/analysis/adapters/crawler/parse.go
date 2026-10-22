package crawler

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// skipped are the elements whose text is not page content: code, and the menus and footers every
// page repeats, which would otherwise outweigh what a page is about. Their links and headings
// are still read.
func skipped(a atom.Atom) bool {
	switch a { //nolint:exhaustive // every other element keeps its text
	case atom.Script, atom.Style, atom.Noscript, atom.Template, atom.Svg, atom.Head, atom.Iframe,
		atom.Nav, atom.Header, atom.Footer:
		return true
	default:
		return false
	}
}

// parse reads an HTML page: its title, description, language, headings and visible text, and the
// links it holds (anchors, and the alternate-language links a multilingual site declares).
func parse(r io.Reader, base *url.URL) (domain.Page, []*url.URL, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return domain.Page{}, nil, err
	}
	pr := pageReader{base: base, page: domain.Page{URL: base.String()}}
	pr.walk(doc, false)
	pr.flush()
	// One line per block, so no keyword phrase runs from one block into the next.
	pr.page.Text = strings.Join(pr.lines, "\n")
	return pr.page, pr.links, nil
}

type pageReader struct {
	base  *url.URL
	page  domain.Page
	line  []string // the text of the current block, in pieces
	lines []string
	links []*url.URL
}

// flush ends the current block of text.
func (pr *pageReader) flush() {
	if s := collapse(strings.Join(pr.line, " ")); s != "" {
		pr.lines = append(pr.lines, s)
	}
	pr.line = pr.line[:0]
}

// block reports whether a starts a new block of text; inline elements such as <b> or <a> do not.
func block(a atom.Atom) bool {
	switch a { //nolint:exhaustive // every other element is inline
	case atom.P, atom.Div, atom.Li, atom.Ul, atom.Ol, atom.Dd, atom.Dt, atom.Td, atom.Th, atom.Tr,
		atom.Table, atom.Section, atom.Article, atom.Aside, atom.Main, atom.Blockquote, atom.Br,
		atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Figcaption, atom.Button, atom.Option:
		return true
	default:
		return false
	}
}

func (pr *pageReader) walk(n *html.Node, hidden bool) {
	isBlock := n.Type == html.ElementNode && block(n.DataAtom)
	switch n.Type { //nolint:exhaustive // comments, doctypes and raw nodes carry nothing to read
	case html.ElementNode:
		pr.element(n)
		hidden = hidden || skipped(n.DataAtom)
	case html.TextNode:
		if !hidden {
			pr.line = append(pr.line, n.Data)
		}
	}
	if isBlock {
		pr.flush()
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		pr.walk(c, hidden)
	}
	if isBlock {
		pr.flush()
	}
}

func (pr *pageReader) element(n *html.Node) {
	switch n.DataAtom { //nolint:exhaustive // only these elements carry what the page summary needs
	case atom.Html:
		pr.page.Language = attr(n, "lang")
	case atom.Title:
		if pr.page.Title == "" {
			pr.page.Title = collapse(textOf(n))
		}
	case atom.Meta:
		if strings.EqualFold(attr(n, "name"), "description") {
			pr.page.Description = collapse(attr(n, "content"))
		}
	case atom.H1, atom.H2, atom.H3:
		if h := collapse(textOf(n)); h != "" {
			pr.page.Headings = append(pr.page.Headings, h)
		}
	case atom.A:
		pr.link(attr(n, "href"))
	case atom.Link:
		if strings.EqualFold(attr(n, "rel"), "alternate") && attr(n, "hreflang") != "" {
			pr.link(attr(n, "href"))
		}
	}
}

func (pr *pageReader) link(href string) {
	if href = strings.TrimSpace(href); href == "" {
		return
	}
	if u, err := pr.base.Parse(href); err == nil {
		pr.links = append(pr.links, u)
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data + " ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
