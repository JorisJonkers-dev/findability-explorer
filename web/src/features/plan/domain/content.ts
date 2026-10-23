/** The twelve-month roadmap, its five moves and the goals it is judged on, as the report sets them out. */

export const TITLE = 'Measure first. Then build. Then expand.'
export const SOURCE = ['Twelve months, three phases', 'A decision at the end of each'] as const

/** A phase's colour on the roadmap: orange, black or blue-grey, as on the poster. */
export type PhaseTone = 'orange' | 'black' | 'blue-grey'

export type Phase = {
  name: string
  when: string
  mode: string
  /** The month it starts in, 1 to 12, and how many months it lasts: its place on the 12-column grid. */
  start: number
  span: number
  tone: PhaseTone
  actions: readonly string[]
  owner: string
  kpi: string
  decision: string
}

export const PHASES: readonly Phase[] = [
  {
    name: 'Measure and fix',
    when: 'Months 0–3',
    mode: 'Preparing',
    start: 1,
    span: 3,
    tone: 'orange',
    actions: [
      'Verify tribelt.nl in Search Console and Bing Webmaster Tools; a sitemap per language.',
      'Track every form, phone click and e-mail click as a conversion, with its source page.',
      'Add “How did you find us?”; mark leads in the CRM.',
      'Stop the brand campaign; add negative keywords.',
      'Check crawler access and Google-Extended; Organization and Product markup.',
      'Run the 40-prompt baseline test.',
    ],
    owner: 'Marketing; sales for CRM marking.',
    kpi: 'Baseline of qualified leads per channel, indexed pages, AI share of voice.',
    decision: 'Confirm the baseline and the inputs for the business case.',
  },
  {
    name: 'Build and target',
    when: 'Months 3–6',
    mode: 'Implementing',
    start: 4,
    span: 3,
    tone: 'black',
    actions: [
      'A specification page per belt type in Dutch, German and English, with competitor-equivalent names and data sheets.',
      'German pages per belt and application; the German overview page as a router.',
      'Google Ads per market and belt family, broad match and AI Max on a qualified-lead goal, daily offline upload.',
      'Start the review programme.',
      'A small, capped ChatGPT ads test in German.',
    ],
    owner: 'Marketing, a German copywriter, an Ads specialist, engineering review.',
    kpi: 'Qualified leads per page and market; paid contact rate; ranks on own and competitor names.',
    decision: 'Scale or cut the content budget.',
  },
  {
    name: 'Expand and learn',
    when: 'Months 6–12',
    mode: 'Disseminating',
    start: 7,
    span: 6,
    tone: 'blue-grey',
    actions: [
      'Monthly “Which belt for…?” problem guides.',
      'Testimonials from reviews.',
      'Trade-fair pages and third-party listings.',
      'Shared content and links with De Spiraal and the Lesjöfors Group.',
      'Monthly prompt test; quarterly check of AI ads in the Netherlands and Germany.',
      'Evaluate an AI product-guidance tool on the terminology map.',
    ],
    owner: 'Marketing, the group companies, sales.',
    kpi: 'Goals G1 to G5.',
    decision: 'Next year’s plan; go or no-go on the guidance tool.',
  },
]

/** What runs through all three phases. */
export const THROUGHOUT =
  'Throughout: a monthly KPI report, the terminology map updated from search terms, sales notes and scans, and a scenario review every six months.'

export type Move = {
  number: string
  title: string
  /** The title as the poster's tile lists it. */
  short: string
  phase: string
  goals: string
  steps: readonly string[]
  why: string
}

export const MOVES: readonly Move[] = [
  {
    number: '1',
    title: 'Measure qualified leads, not visits',
    short: 'Measure qualified leads',
    phase: 'Phase 1',
    goals: 'G1, G2',
    steps: [
      'Count quotation requests and tracked calls and e-mails from product pages.',
      'Ask every new contact how they found Tribelt, with “AI assistant” as an option.',
      'Close the loop in the CRM and feed qualified and won leads back to Google Ads daily.',
    ],
    why: 'Today the data stops at the website. Without this chain, neither Tribelt nor Google Ads can tell a good lead from a click.',
  },
  {
    number: '2',
    title: 'Restructure paid search',
    short: 'Restructure paid search',
    phase: 'Phases 1–2',
    goals: 'G2, G3',
    steps: [
      'Stop the brand campaign; add negative keywords.',
      'One campaign per market and belt family, broad match and AI Max.',
      'Bid on a qualified-lead goal with enhanced conversions.',
    ],
    why: 'Paid search brings 78.9% of users but only 0.04% contact Tribelt. Restructured, the account is also ready for AI ad surfaces when they open in NL and DE.',
  },
  {
    number: '3',
    title: 'One specification page per belt type',
    short: 'A page per belt type',
    phase: 'Phase 2',
    goals: 'G1, G5',
    steps: [
      'HTML pages with every value as text: wire, pitch, width, material, temperature, load.',
      '“Comparable to: TwenteFlex, Omni-Grid”, marked as others’ trademarks.',
      'Open specifications; CAD drawings behind the quotation form.',
    ],
    why: 'Specifiers search exact values and the installed competitor product. Verifiable facts make Tribelt the most citable source for its own belts.',
  },
  {
    number: '4',
    title: 'Close the AI-search gap',
    short: 'Get indexed for AI search',
    phase: 'Phase 1',
    goals: 'G4',
    steps: [
      'Get indexed in every language: Search Console, Bing, sitemaps.',
      'Allow Googlebot, Google-Extended and the AI search crawlers.',
      'Organization and Product markup; llms.txt as cheap hygiene.',
    ],
    why: 'Google needs nothing extra for AI Overviews beyond being indexed. Our mirror showed Google may not come at all until a site is submitted.',
  },
  {
    number: '5',
    title: 'Restart the survey as reviews',
    short: 'Reviews after delivery',
    phase: 'Phase 2',
    goals: 'G1, G4',
    steps: ['A short review request after each delivery.', 'Short testimonials customers can approve quickly.', 'Ratings on the belt pages.'],
    why: 'One programme gives listening data, social proof, and the rating that quote-only belts need for Product rich results.',
  },
]

export type Goal = {
  id: string
  goal: string
  target: string
  measure: string
  /** Today and the target as shares of the bar, for the two goals with a figure today. */
  bar?: { today: number; target: number }
}

export const GOALS: readonly Goal[] = [
  { id: 'G1', goal: 'More qualified leads', target: '+50%', measure: 'Quotation requests plus tracked calls and e-mails, per market, against the phase-1 baseline' },
  { id: 'G2', goal: 'Paid search that converts', target: '0.04% → 0.5%', measure: 'Share of paid-search users who contact Tribelt', bar: { today: 0.04 / 0.5, target: 1 } },
  { id: 'G3', goal: 'German demand', target: '≥ 25%', measure: 'Share of qualified leads from Germany, Austria and Switzerland' },
  { id: 'G4', goal: 'Visible in AI answers', target: '0% → 25%', measure: 'Share of the 40 test prompts in which Gemini names Tribelt', bar: { today: 0, target: 0.25 } },
  { id: 'G5', goal: 'Found for every name', target: '6 × 2', measure: 'Page 1 on Google.de and Google.nl for each belt type, own and competitor name' },
]

/** The roadmap's month labels. */
export const MONTHS = Array.from({ length: 12 }, (_, i) => `M${String(i + 1)}`)
