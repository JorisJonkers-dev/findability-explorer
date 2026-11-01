import { describe, expect, it } from 'vitest'
import { bots, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { builtWith, crawlerRows, evidence, filesAsFound, scanDataLine, siteHost, sitePurpose, siteTitle, structuredData } from './site'

const tribelt = readinessRow('Tribelt B.V.', {
  competitorId: 'tribelt',
  role: 'tribelt',
  scan: summary({ competitorId: 'tribelt', url: 'https://www.tribelt.nl', number: 3 }),
  readiness: readiness({ markdown: false }),
})
const noLlms = { llmsTxt: { present: false, bytes: 0, links: 0 } }

describe('a site’s page', () => {
  it('is titled and introduced by the site', () => {
    expect(siteTitle(tribelt)).toBe('Tribelt’s AI readiness')
    expect(siteTitle(readinessRow('ESFO'))).toBe('ESFO’s AI readiness')
    expect(siteHost(tribelt)).toBe('tribelt.nl')
    expect(sitePurpose(tribelt)).toBe('tribelt.nl passes 7 of the 8 checks.')
    expect(sitePurpose(readinessRow('ESFO', { scan: undefined, readiness: readiness({ bots: bots({ GPTBot: 'blocked' }) }) }))).toBe('ESFO passes 7½ of the 8 checks.')
    expect(sitePurpose(readinessRow('ESFO', { readiness: undefined }))).toBe('No scan has recorded the AI readiness of twentebelt.com yet.')
  })

  it('says which scan the figures come from', () => {
    expect(scanDataLine(tribelt)).toBe('From scan 3 of 22 October 2026.')
    expect(scanDataLine(readinessRow('ESFO', { readiness: readiness({ checkedAt: '2026-10-28T08:00:00Z' }) }))).toBe('From the scan of 22 October 2026, checked again on 28 October 2026.')
    expect(scanDataLine(readinessRow('ESFO', { scan: summary({ finishedAt: undefined }) }))).toBe('From the scan of 22 October 2026.')
    expect(scanDataLine(readinessRow('ESFO', { scan: undefined }))).toBe('')
  })

  it('says what the site is built with', () => {
    expect(builtWith(readiness({ stack: { cms: 'WordPress', seoPlugin: 'Yoast SEO', botProtection: 'Cloudflare' } }))).toBe('Built with WordPress, Yoast SEO, and Cloudflare.')
    expect(builtWith(readiness({ stack: {} }))).toBe('')
  })
})

describe('the evidence of each check', () => {
  it('counts the crawlers let in and kept out', () => {
    expect(evidence('crawlers', readiness())).toBe('10 of the 10 crawlers are allowed in, and 0 are blocked.')
    expect(evidence('crawlers', readiness({ bots: bots({ GPTBot: 'blocked', CCBot: 'partly', ClaudeBot: 'partly' }) }))).toBe(
      '9 of the 10 crawlers are allowed in, 2 of them with some paths kept out, and 1 is blocked.',
    )
    expect(evidence('crawlers', readiness({ robots: 'missing' }))).toBe('The site has no robots.txt, so nothing keeps a crawler out.')
    expect(evidence('crawlers', readiness({ robots: 'unreachable', bots: [] }))).toBe('robots.txt could not be reached, so the crawler policy is not known.')
  })

  it('describes the sitemap, and how many of the pages read it lists', () => {
    expect(evidence('sitemap', readiness())).toBe('A sitemap in 2 files lists 124 addresses, which covers 85% of the pages the scan read.')
    expect(evidence('sitemap', readiness({ sitemap: { declared: true, files: 1, addresses: 1200, languages: [], coverage: { pages: 10, listed: 10, share: 1 } } }))).toBe(
      'A sitemap lists 1,200 addresses, which covers every page the scan read.',
    )
    expect(evidence('sitemap', readiness({ sitemap: { declared: true, files: 1, addresses: 3, languages: [] } }))).toBe('A sitemap lists 3 addresses.')
    expect(evidence('sitemap', readiness({ sitemap: { declared: true, files: 0, addresses: 0, languages: [] } }))).toBe('robots.txt points to a sitemap, but the scan could not read it.')
    expect(evidence('sitemap', readiness({ sitemap: { declared: false, files: 0, addresses: 0, languages: [] } }))).toBe('robots.txt does not point to a sitemap.')
  })

  it('names the languages of the sitemap and of the site', () => {
    expect(evidence('sitemapLanguages', readiness())).toBe('The sitemap lists pages in Dutch, German, and English.')
    expect(evidence('sitemapLanguages', readiness({ sitemap: { declared: false, files: 0, addresses: 0, languages: [] } }))).toBe('The sitemap lists no language for its pages.')
    expect(evidence('languages', readiness({ languages: ['nl', 'en'] }))).toBe('The site has pages in Dutch and English.')
    expect(evidence('languages', readiness({ languages: [] }))).toBe('The home page names no language versions.')
    expect(evidence('languages', readiness({ homeRead: false }))).toBe('The home page could not be read, so its languages are not known.')
  })

  it('says what the structured data says of the company', () => {
    expect(evidence('organization', readiness())).toBe('The home page says who the company is in structured data (Organization and Product).')
    const notOk = readiness().advice.map((a) => (a.check === 'organization' ? { ...a, ok: false } : a))
    expect(evidence('organization', readiness({ advice: notOk, structuredData: ['WebSite'] }))).toBe('The home page carries structured data (WebSite), but none says who the company is.')
    expect(evidence('organization', readiness({ advice: notOk, structuredData: [] }))).toBe('The home page carries no structured data.')
    expect(evidence('organization', readiness({ homeRead: false }))).toBe('The home page could not be read, so its structured data is not known.')
  })

  it('describes the llms files and the Markdown pages', () => {
    expect(evidence('llmsTxt', readiness())).toBe('llms.txt is 5,120 bytes and links to 124 pages.')
    expect(evidence('llmsTxt', readiness(noLlms))).toBe('The site has no llms.txt.')
    expect(evidence('llmsFullTxt', readiness())).toBe('The site has an llms-full.txt.')
    expect(evidence('llmsFullTxt', readiness({ llmsFullTxt: false }))).toBe('The site has no llms-full.txt.')
    expect(evidence('markdown', readiness())).toBe('Its pages are also offered as Markdown.')
    expect(evidence('markdown', readiness({ markdown: false }))).toBe('Its pages are not offered as Markdown.')
  })
})

describe('what the scan found', () => {
  it('lists the structured data, and Product beside the score', () => {
    expect(structuredData(readiness())).toEqual({ found: 'Organization and Product.', product: 'Product structured data: found. It is shown here, but it does not count toward the score.' })
    const noProduct = readiness().advice.map((a) => (a.check === 'product' ? { ...a, ok: false } : a))
    expect(structuredData(readiness({ structuredData: [], advice: noProduct }))).toEqual({
      found: 'No structured data was found on the home page.',
      product: 'Product structured data: not found. It is shown here, but it does not count toward the score.',
    })
    expect(structuredData(readiness({ homeRead: false })).found).toBe('The home page could not be read, so its structured data is not known.')
  })

  it('lists the files as found', () => {
    expect(filesAsFound(readiness())).toEqual([
      { name: 'robots.txt', text: 'Found.' },
      { name: 'llms.txt', text: 'Found, 5,120 bytes, 124 links.' },
      { name: 'llms-full.txt', text: 'Found.' },
      { name: 'Sitemap', text: 'Found, 2 files with 124 addresses.' },
    ])
    const none = readiness({ robots: 'missing', ...noLlms, llmsFullTxt: false, sitemap: { declared: false, files: 0, addresses: 0, languages: [] } })
    expect(filesAsFound(none).map((f) => f.text)).toEqual(['Not found.', 'Not found.', 'Not found.', 'Not found.'])
    expect(filesAsFound(readiness({ robots: 'unreachable', sitemap: { declared: true, files: 1, addresses: 1, languages: [] } })).map((f) => f.text)).toEqual([
      'The scan could not reach it.',
      'Found, 5,120 bytes, 124 links.',
      'Found.',
      'Found, 1 file with 1 address.',
    ])
  })
})

describe('the crawler policy', () => {
  const described = [{ bot: 'GPTBot', lab: 'OpenAI', gets: 'What future ChatGPT models know', feeds: 'future' as const }]

  it('says, per crawler, who runs it, what it does, and what robots.txt says to it', () => {
    const rows = crawlerRows(readiness({ bots: [...bots({ GPTBot: 'blocked' }).slice(5, 6), ...bots({ CCBot: 'partly' }, false).slice(9)] }), described)
    expect(rows).toEqual([
      { bot: 'GPTBot', lab: 'OpenAI', gets: 'What future ChatGPT models know', access: 'Blocked', note: '', blocked: true },
      { bot: 'CCBot', lab: 'Common Crawl', gets: '', access: 'Allowed, with some paths kept out', note: 'Not named, so the general rules apply.', blocked: false },
    ])
    expect(crawlerRows(readiness({ bots: bots().slice(0, 1) }), [])[0]).toMatchObject({ bot: 'Googlebot', lab: 'Google Search', access: 'Allowed', note: '' })
    expect(crawlerRows(readiness({ bots: [{ bot: 'Other', access: 'allowed', named: true, followsAdvice: true }] }), [])[0]).toMatchObject({ lab: '', access: 'Allowed' })
  })

  it('says when robots.txt does not mention them, or could not be read', () => {
    expect(crawlerRows(readiness({ robots: 'missing', bots: bots().slice(0, 1) }), [])[0]).toMatchObject({ access: 'Not mentioned', note: 'The site has no robots.txt.' })
    expect(crawlerRows(readiness({ robots: 'unreachable', bots: [] }), described)).toEqual([])
    expect(crawlerRows(readiness({ robots: 'unreachable', bots: bots().slice(0, 1) }), [])[0]).toMatchObject({ access: 'Not known', note: 'robots.txt could not be reached.' })
  })
})
