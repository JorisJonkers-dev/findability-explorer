/** An element's text with its parts apart, as a reader sees it: "25% AI responses that name Tribelt". */
export function words(w: { element: Element }): string {
  const walker = document.createTreeWalker(w.element, NodeFilter.SHOW_TEXT)
  const parts: string[] = []
  for (let n = walker.nextNode(); n; n = walker.nextNode()) parts.push(n.textContent ?? '')
  return parts
    .map((p) => p.trim())
    .filter(Boolean)
    .join(' ')
}
