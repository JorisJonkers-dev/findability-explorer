const ENTITIES: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }

function escapeHtml(text: string): string {
  return text.replace(/[&<>"']/g, (c) => String(ENTITIES[c]))
}

/** One line's text: escaped first, then links (http and https only), bold, and italics. */
function inline(text: string): string {
  return escapeHtml(text)
    .replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>')
}

type List = { tag: 'ul' | 'ol'; items: string[] }

/**
 * A model's response as safe HTML: headings (each an h4), bold, italics, lists, paragraphs, and
 * links. Everything is escaped first, so no HTML in a response is ever rendered as HTML, and a
 * link only becomes a link when it is an http or https address.
 */
export function renderMarkdown(src: string): string {
  const out: string[] = []
  let para: string[] = []
  let list = null as List | null
  const flushPara = () => {
    if (para.length > 0) out.push(`<p>${inline(para.join(' '))}</p>`)
    para = []
  }
  const flushList = () => {
    if (list) out.push(`<${list.tag}>${list.items.map((i) => `<li>${inline(i)}</li>`).join('')}</${list.tag}>`)
    list = null
  }
  for (const raw of src.split('\n')) {
    const line = raw.trim()
    const heading = /^#{1,6}\s+(.*)$/.exec(line)
    const bullet = /^[-*•]\s+(.*)$/.exec(line)
    const item = bullet ?? /^\d+[.)]\s+(.*)$/.exec(line)
    if (line === '') {
      flushPara()
      flushList()
    } else if (heading) {
      flushPara()
      flushList()
      out.push(`<h4>${inline(String(heading[1]))}</h4>`)
    } else if (item) {
      const tag = bullet ? 'ul' : 'ol'
      flushPara()
      if (list?.tag !== tag) flushList()
      list ??= { tag, items: [] }
      list.items.push(String(item[1]))
    } else {
      flushList()
      para.push(line)
    }
  }
  flushPara()
  flushList()
  return out.join('')
}
