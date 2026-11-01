import { describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  it('escapes HTML before it formats anything', () => {
    expect(renderMarkdown('<b>x</b> **bold**')).toBe('<p>&lt;b&gt;x&lt;/b&gt; <strong>bold</strong></p>')
    expect(renderMarkdown('<script>alert("x")</script>')).toBe('<p>&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;</p>')
    expect(renderMarkdown('<img src=x onerror=alert(1)>')).toBe('<p>&lt;img src=x onerror=alert(1)&gt;</p>')
    expect(renderMarkdown("Tribelt's & Co")).toBe('<p>Tribelt&#39;s &amp; Co</p>')
  })

  it('makes links of http and https addresses only', () => {
    expect(renderMarkdown('See [Tribelt](https://www.tribelt.nl/).')).toBe(
      '<p>See <a href="https://www.tribelt.nl/" target="_blank" rel="noopener noreferrer">Tribelt</a>.</p>',
    )
    expect(renderMarkdown('[plain](http://example.com)')).toBe('<p><a href="http://example.com" target="_blank" rel="noopener noreferrer">plain</a></p>')
    expect(renderMarkdown('[x](javascript:alert(1))')).toBe('<p>[x](javascript:alert(1))</p>')
    expect(renderMarkdown('[x](data:text/html,hi)')).toBe('<p>[x](data:text/html,hi)</p>')
  })

  it('keeps a quote in an address from leaving the attribute', () => {
    expect(renderMarkdown('[x](https://a.test/"onmouseover="alert(1))')).toBe(
      '<p><a href="https://a.test/&quot;onmouseover=&quot;alert(1" target="_blank" rel="noopener noreferrer">x</a>)</p>',
    )
  })

  it('writes bold and italics', () => {
    expect(renderMarkdown('**Tribelt** makes *eyelink* belts')).toBe('<p><strong>Tribelt</strong> makes <em>eyelink</em> belts</p>')
  })

  it('makes every heading an h4', () => {
    expect(renderMarkdown('## Belts')).toBe('<h4>Belts</h4>')
    expect(renderMarkdown('# One\n###### Six')).toBe('<h4>One</h4><h4>Six</h4>')
  })

  it('makes lists, numbered or not', () => {
    expect(renderMarkdown('- a\n- b')).toBe('<ul><li>a</li><li>b</li></ul>')
    expect(renderMarkdown('* a\n• b')).toBe('<ul><li>a</li><li>b</li></ul>')
    expect(renderMarkdown('1. a\n2) b')).toBe('<ol><li>a</li><li>b</li></ol>')
  })

  it('ends a list where another kind of list or a paragraph starts', () => {
    expect(renderMarkdown('- a\n1. b')).toBe('<ul><li>a</li></ul><ol><li>b</li></ol>')
    expect(renderMarkdown('- a\nafter')).toBe('<ul><li>a</li></ul><p>after</p>')
    expect(renderMarkdown('before\n- a')).toBe('<p>before</p><ul><li>a</li></ul>')
  })

  it('joins lines into paragraphs and splits them at blank lines', () => {
    expect(renderMarkdown('one\ntwo\n\nthree')).toBe('<p>one two</p><p>three</p>')
    expect(renderMarkdown('  \n\n')).toBe('')
    expect(renderMarkdown('')).toBe('')
  })

  it('closes a paragraph before a heading', () => {
    expect(renderMarkdown('intro\n## Next\n- a')).toBe('<p>intro</p><h4>Next</h4><ul><li>a</li></ul>')
  })
})
