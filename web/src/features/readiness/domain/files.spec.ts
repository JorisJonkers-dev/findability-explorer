import { describe, expect, it } from 'vitest'
import { readiness } from '../../../testing/fixtures'
import { escapeHtml, FILE_KEYS, fileKeyOf, filesFoundLine, fileSize, fileTabs, missingSentence, searchLines } from './files'

const files = { robotsTxt: 'User-agent: *\nDisallow: /admin/', robotsTruncated: false, llmsTxt: '# Tribelt', llmsTruncated: true, llmsFullTxt: '', llmsFullTruncated: false }

describe('the search in a file', () => {
  it('marks the lines with a match, and counts the matches', () => {
    expect(searchLines('User-agent: *\nDisallow: /admin/', 'admin')).toEqual({
      lines: [
        { n: 1, html: 'User-agent: *', hit: false },
        { n: 2, html: 'Disallow: /<mark>admin</mark>/', hit: true },
      ],
      hits: 1,
    })
  })

  it('matches without regard to case, every time a line has it, keeping the text as written', () => {
    const { lines, hits } = searchLines('GPTBot gptbot\nnone', 'GptBot')
    expect(lines[0]?.html).toBe('<mark>GPTBot</mark> <mark>gptbot</mark>')
    expect(hits).toBe(2)
  })

  it('escapes the text first, so a query with < finds it and the file cannot add markup', () => {
    const { lines, hits } = searchLines('<url>&<loc>"x"\'y\'</loc>', '<loc')
    expect(lines[0]?.html).toBe('&lt;url&gt;&amp;<mark>&lt;loc</mark>&gt;&quot;x&quot;&#39;y&#39;&lt;/loc&gt;')
    expect(hits).toBe(1)
    expect(searchLines('a.b', '.').lines[0]?.html).toBe('a<mark>.</mark>b')
    expect(searchLines('a+b', 'a+').hits).toBe(1)
    // A query never matches inside an escape.
    expect(searchLines('a&b', 'amp')).toEqual({ lines: [{ n: 1, html: 'a&amp;b', hit: false }], hits: 0 })
  })

  it('marks nothing for an empty or blank query', () => {
    expect(searchLines('Disallow: /', '')).toEqual({ lines: [{ n: 1, html: 'Disallow: /', hit: false }], hits: 0 })
    expect(searchLines('Disallow: /', '  ').hits).toBe(0)
  })

  it('escapes the HTML characters', () => {
    expect(escapeHtml('<a href="x">&\'</a>')).toBe('&lt;a href=&quot;x&quot;&gt;&amp;&#39;&lt;/a&gt;')
  })
})

describe('the files of a site', () => {
  it('measures a file in bytes and lines', () => {
    expect(fileSize('ab\ncd')).toEqual({ lines: 2, bytes: 5 })
    expect(fileSize('é')).toEqual({ lines: 1, bytes: 2 })
  })

  it('reads a tab from the address, robots.txt by default', () => {
    expect(FILE_KEYS).toEqual(['robots', 'sitemap', 'llms', 'llmsFull'])
    expect(fileKeyOf('llms')).toBe('llms')
    expect(fileKeyOf('other')).toBe('robots')
    expect(fileKeyOf(undefined)).toBe('robots')
  })

  it('has a tab per file: the text kept, not found, not reached, or the sitemap whose text is not kept', () => {
    const tabs = fileTabs(files, readiness({ llmsFullTxt: false }))
    expect(tabs.map((t) => [t.key, t.name, t.state, t.text, t.truncated])).toEqual([
      ['robots', 'robots.txt', 'found', 'User-agent: *\nDisallow: /admin/', false],
      ['sitemap', 'sitemap.xml', 'notKept', '', false],
      ['llms', 'llms.txt', 'found', '# Tribelt', true],
      ['llmsFull', 'llms-full.txt', 'missing', '', false],
    ])
    const unreached = fileTabs({ ...files, robotsTxt: '' }, readiness({ robots: 'unreachable', sitemap: { declared: false, files: 0, addresses: 0, languages: [] } }))
    expect(unreached.map((t) => t.state)).toEqual(['unreachable', 'missing', 'found', 'missing'])
    expect(fileTabs(undefined, undefined).map((t) => t.state)).toEqual(['missing', 'missing', 'missing', 'missing'])
  })

  it('says which files were found and which were not', () => {
    expect(filesFoundLine(fileTabs(files, readiness({ llmsFullTxt: false })))).toBe('Found: robots.txt, sitemap.xml, and llms.txt. Not found: llms-full.txt.')
    expect(filesFoundLine(fileTabs(undefined, undefined))).toBe('No file was found.')
    expect(filesFoundLine(fileTabs({ ...files, llmsFullTxt: 'x' }, readiness()))).toBe('Found: robots.txt, sitemap.xml, llms.txt, and llms-full.txt.')
  })

  it('says what is missing, and why it matters for the llms files', () => {
    expect(missingSentence({ key: 'llmsFull', name: 'llms-full.txt', state: 'missing' }, 'example.com')).toBe(
      'The scan found no llms-full.txt on example.com. Adding one is one of the eight AI readiness checks.',
    )
    expect(missingSentence({ key: 'robots', name: 'robots.txt', state: 'missing' }, 'example.com')).toBe('The scan found no robots.txt on example.com.')
    expect(missingSentence({ key: 'robots', name: 'robots.txt', state: 'unreachable' }, 'example.com')).toBe('The scan could not reach robots.txt on example.com.')
    expect(missingSentence({ key: 'sitemap', name: 'sitemap.xml', state: 'notKept' }, 'example.com')).toBe("This release does not keep the sitemap's text yet.")
    expect(missingSentence({ key: 'robots', name: 'robots.txt', state: 'found' }, 'example.com')).toBe('')
  })
})
