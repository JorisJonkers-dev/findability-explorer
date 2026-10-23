package domain

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// Sitemap is what one sitemap file lists: pages, with the language versions each declares, or,
// for a sitemap index, other sitemaps.
type Sitemap struct {
	Pages    []string
	Sitemaps []string
}

// MaxSitemapEntries is the most addresses one sitemap file yields.
const MaxSitemapEntries = 5000

// ParseSitemap reads a sitemap (<urlset>) or a sitemap index (<sitemapindex>), as sitemaps.org
// defines them, with the <xhtml:link rel="alternate" hreflang> language versions a page entry may
// carry. Namespaces are ignored, so a sitemap without its xmlns still reads. It fails only on XML
// that does not parse before any entry was read.
func ParseSitemap(r io.Reader) (Sitemap, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	p := sitemapParser{}
	for p.entries() < MaxSitemapEntries {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return p.out, nil
		}
		if err != nil {
			if p.entries() > 0 {
				return p.out, nil // a cut-off file still lists what came before the cut
			}
			return Sitemap{}, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			p.start(t)
		case xml.CharData:
			if p.inLoc {
				p.loc.Write(t)
			}
		case xml.EndElement:
			p.end(t.Name.Local)
		}
	}
	return p.out, nil
}

// sitemapParser follows where in a sitemap the decoder is.
type sitemapParser struct {
	out                     Sitemap
	inURL, inSitemap, inLoc bool
	loc                     strings.Builder
}

func (p *sitemapParser) entries() int { return len(p.out.Pages) + len(p.out.Sitemaps) }

func (p *sitemapParser) start(t xml.StartElement) {
	switch t.Name.Local {
	case "url":
		p.inURL = true
	case "sitemap":
		p.inSitemap = true
	case "loc":
		p.inLoc = p.inURL || p.inSitemap
		p.loc.Reset()
	case "link":
		if href := xmlAttr(t, "href"); p.inURL && xmlAttr(t, "rel") == "alternate" && href != "" {
			p.out.Pages = append(p.out.Pages, strings.TrimSpace(href))
		}
	}
}

func (p *sitemapParser) end(name string) {
	switch name {
	case "loc":
		p.out.add(strings.TrimSpace(p.loc.String()), p.inSitemap)
		p.inLoc = false
	case "url":
		p.inURL = false
	case "sitemap":
		p.inSitemap = false
	}
}

func (s *Sitemap) add(loc string, index bool) {
	switch {
	case loc == "":
	case index:
		s.Sitemaps = append(s.Sitemaps, loc)
	default:
		s.Pages = append(s.Pages, loc)
	}
}

func xmlAttr(e xml.StartElement, name string) string {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
