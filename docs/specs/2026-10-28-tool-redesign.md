# Redesigning the Findability Explorer as a tool

Release 1.12.0 laid every page out as a poster: one headline, at most three figures, one chart and
one list. That reads well on a slide, but the app is being handed to Tribelt, whose staff will read
it every week and whose administrator will run it. They need to know what each page shows, where
its figures come from, how old they are, and how to start, stop, tune, and pay for the work behind
them.

**The app becomes a tool with two views, Results and Manage, both organised around the same four
processes.** This document is the spec for that redesign. The words follow the glossary
(`CONTEXT.md` in the course repo), which gained Prompt set, Model list, Recommendation run,
Competitor definition, Results, and Manage for it.

The audience for Manage is a marketer with no technical background, so every button says what it
does, what it costs, and what stopping it keeps.

## 1. Decisions

| # | Decision |
|---|---|
| 1 | Two views. **Results** is for everyone signed in; **Manage** is for administrators only. |
| 2 | Both views are organised by process: asking AI assistants, checking websites, search terms, and recommendations. Each Results page links to its Manage page for administrators, and back. |
| 3 | The poster rule goes. Every page has the same frame (section 3) and then the sections it needs: tables, filters, and charts where they help. Each page still says something different. |
| 4 | Same look: the colours, fonts, and logos stay. Type gets smaller, content is narrower, and the very large headlines and full-bleed blocks go. |
| 5 | Every piece of work can be stopped, by a button named for it ("Stop the Prompt run"). There is no umbrella word such as job or task. |
| 6 | Every piece of work shows a dollar estimate before it starts. When the estimate is more than the budget has left, the button says the work will stop when the budget runs out and asks for a second click. The budget stop stays. |
| 7 | Two monthly budgets, both in dollars, both in Keys and spending: OpenRouter (the Prompt test) and Anthropic (everything Haiku or Sonnet does). The Anthropic budget replaces the `LLM_MONTHLY_TOKENS` limit. |
| 8 | Results always show the newest **complete** Prompt run. A stopped run is kept and listed in Manage, where a retry finishes it. |
| 9 | The prompt set is edited as a draft and published as a new version. Publishing shows the change in cost per run and per year. Scopes can be added, such as French or Polish with no region. |
| 10 | The model list is chosen from OpenRouter's whole catalogue. Free models are added in one click, each provider's cheapest model is marked Recommended, and any model can be added under its provider. A provider can have several models; each model is its own column in Results. |
| 11 | Trends over time break where the prompt set or the model list changed, with a note saying which. Each model's own line runs on while it stays on the list. |
| 12 | Other languages follow later: the readiness check "pages in Dutch, German, and English" and the search term list stay three languages, built so a language can be added. |
| 13 | The hand-written recommendations stay in code. A **Recommendation run** writes the rest: rules find and score the gaps, a chosen model (Haiku 5.5 by default, Sonnet 5.5 as the option) reads the newest tribelt.nl scan to place and word each one, and may move a score by one point with a written reason. |
| 14 | Kinds of generated recommendation: search terms to add and where, pages to add, sites to get listed on, fact sheet corrections, and the specifics of AI readiness (such as the pages with no Markdown version). |
| 15 | Whether a recommendation is done is decided by rules, never by a model: a term on the page it named, a page for that family or application in that language, a listing fetched and found to name Tribelt, a readiness check passing. Only what no scan can see is ticked by hand. |
| 16 | A Recommendation run starts by hand. Activity says when its inputs are newer than the last run. A new run replaces the open generated recommendations and keeps those done or edited. |
| 17 | The fact check separates "wrong about Tribelt" from "left out", and only wrong claims become fact sheet corrections. The newest run is read again. |
| 18 | Scans record which page each term is on, and keep tribelt.nl's page text for the Recommendation run. |
| 19 | The competitor definition is edited in Manage, with versions. Saving offers a recheck with its estimate; it does not start one. |
| 20 | Administrators can add and remove search terms by hand. A hand-added term is marked as such. |
| 21 | The term review stays automatic, with an on and off switch, its month's spend, and Stop. |
| 22 | Makers of plastic or fabric belts (Habasit, Forbo Siegling, Intralox, and others) are archived. AI answers still show them under "Who the answers name", marked "Not a competitor". |
| 23 | Buyers is dropped. `/buyers` redirects to the Overview. |
| 24 | Tribelt adds its own OpenRouter and Anthropic keys. Everyone signs in with an auth-api account; the Viewer Key waits. |
| 25 | "Numbers to watch" becomes **Targets**, edited in Manage, Recommendations. The deck keeps "Five targets to hit". |
| 26 | Every page opens with one sentence on what it shows and links to its section of How this tool works. Manage explains each button next to it. |
| 27 | Before building, the whole site is built as an interactive prototype in an artifact, on the live export of 28 October with the plastic and fabric makers archived and sample data where the app has none yet. |

## 2. The site map

### Results

The top bar reads Overview, AI answers, AI readiness, Search terms, Competitors, Recommendations and
How it works, with Manage on the right for administrators and the account menu after it.

| Page | Address | Reached from |
|---|---|---|
| Overview | `/` | the top bar |
| A target | `/targets/:measure` | a row of Targets on the Overview |
| AI answers | `/ai-answers` | the top bar |
| Every answer | `/ai-answers/all` | AI answers, a target page |
| One answer | `/ai-answers/:id` | a square of Every answer, a cell of the family and scope table, a competitor page |
| AI readiness | `/ai-readiness` | the top bar |
| One site | `/ai-readiness/:id` | a row of Every site, a competitor page, a readiness recommendation |
| Search terms | `/terms` | the top bar |
| Every term | `/terms/all` | Search terms |
| A belt family | `/terms/families/:family` | Search terms, AI answers, a competitor page |
| One term | `/terms/:id` | Every term, a belt family, a competitor page, a term recommendation |
| Competitors | `/competitors` | the top bar |
| A competitor | `/competitors/:id` | Competitors, Who the answers name |
| Its scan history | `/competitors/:id/history` | a competitor page, a site page |
| Recommendations | `/recommendations` | the top bar, Do these next on the Overview |
| One recommendation | `/recommendations/:id` | Recommendations, a target page, a term page, a failed check |
| How this tool works | `/how-it-works` | the top bar, every page's "How this is measured" |

### Manage

Manage has its own menu down the left: Activity, Prompt test, Website scans, Competitor list,
Search term list, Recommendations, and Keys and spending.

| Page | Address |
|---|---|
| Activity | `/manage` |
| Prompt test | `/manage/prompt-test`, with `/questions`, `/models`, `/schedule`, `/fact-sheet` and `/runs` |
| Website scans | `/manage/scans`, and a scan at `/manage/scans/:id` |
| Competitor list | `/manage/competitors` |
| Search term list | `/manage/terms` |
| Recommendations | `/manage/recommendations` |
| Keys and spending | `/manage/keys` |

### Old addresses

Every address of 1.12.0 and earlier redirects: `/numbers/:measure` to `/targets/:measure`,
`/ai-search` to `/ai-answers`, `/ai-search/answers` to `/ai-answers/all`, `/ai-search/readiness` to
`/ai-readiness`, `/buyers` to `/`, `/scan/:id` to `/manage/scans/:id`, `/settings` to `/manage/keys`,
`/settings/prompt-test` to `/manage/prompt-test`, `/settings/terms` to `/manage/terms`, and
`/settings/competitors` to `/manage/competitors`. The redirects 1.12.0 already has stay.

## 3. The page frame

Every page, in both views, has, in this order:

1. **The title.**
2. **One sentence on what the page shows.**
3. **The data line**: where its figures come from and when, such as "From the Prompt run of 1
   October 2026: 20 questions asked of 5 models, 200 answers."
4. **For administrators, a link to its Manage page** on the right, and on a Manage page a link to
   its Results page.
5. **Its sections.**
6. **"How this is measured"**, a link to its process's section of How this tool works.

A detail page has a back link to the page it came from above the title. The team footer stays.
Empty states are sentences that say what is missing and, for administrators, where to start it,
such as "No Prompt run has finished yet. Start one in Manage, Prompt test."

## 4. Results, page by page

The figures here are samples from the export; every figure on a real page comes from the API.

### Overview

> **Overview**
> How easily buyers find Tribelt in search and AI assistants, measured against five targets.
> *From the Prompt run of 1 October 2026 and the scans up to 27 October 2026.*

1. **Summary.** One sentence written by rules from the data, such as "AI answers name six
   competitors more often than Tribelt, and tribelt.nl uses 45 of the 387 search terms buyers use."
2. **Targets.** A row per target: its name, Tribelt's value, the target, the leader and its value,
   and a progress bar. Each row opens its target page. The five are: AI answers that name Tribelt;
   AI answers about Twente that name Tribelt; web answers that cite tribelt.nl; AI readiness checks
   passed; search terms tribelt.nl uses.
3. **What changed.** Against the previous complete Prompt run and tribelt.nl scan, such as "Since
   the Prompt run of 1 September, AI answers that name Tribelt went from 4% to 5%." With nothing to
   compare: "This is the first complete Prompt run, so there is nothing to compare yet."
4. **Do these next.** The three open recommendations with the best benefit against effort, each
   with its kind, and "See all recommendations".
5. **How fresh the data is.** A row per process: the last complete Prompt run and the next
   scheduled one; the newest tribelt.nl scan and how many competitors were scanned in the last 30
   days; the last term review; the last Recommendation run.

### A target

> **AI answers that name Tribelt**
> The share of a Prompt run's answers that name Tribelt. The target is 40%.

1. **Tribelt against the competitors.** Bars for Tribelt, the typical competitor, the leader and
   the target.
2. **Over time.** A line over the complete Prompt runs (or tribelt.nl scans, for readiness and
   terms). Where the prompt set or the model list changed, the line breaks, with the note "The
   questions changed here, so the figures before and after are not the same measure" or "The model
   list changed here".
3. **Broken down.** For the answer targets, by model, by scope, and by belt family; for readiness,
   by check; for search terms, by language.
4. **What improves it.** The recommendations aimed at this target.

### AI answers

> **AI answers**
> How often AI assistants name and cite Tribelt when a buyer describes a job and asks which metal
> conveyor belt to use and who makes it.
> *From the Prompt run of 1 October 2026: 20 questions asked of 5 models, each from the model's own
> knowledge and with web search, 200 answers in all. Prompt set version 2.*

1. **Three figures:** answers that name Tribelt (5%, target 40%); answers about Twente that name
   Tribelt (16%); web answers that cite tribelt.nl (28%).
2. **Who the answers name.** Companies ranked by the share of answers that name them. Tribelt is
   highlighted, group companies are marked "Group company", archived makers "Not a competitor".
   Competitors link to their pages.
3. **By model.** A row per model on the model list, named by provider and model ("Google, Gemini
   2.5 Flash"), with three columns: named from its own knowledge, named with web search, and cited.
4. **By belt family and scope.** Five families against the scopes, with the share that names
   Tribelt in each cell. A cell opens Every answer filtered to it.
5. **The questions asked.** Every question in its own language, with an English translation under
   it.
6. **See every answer.**

### Every answer

> **Every answer**
> Each square is one model's answer to one question. Its colour shows whether the answer names
> Tribelt, and where.

- Filters: search mode, scope, belt family, model, and "Names Tribelt".
- A run picker: earlier runs by date and prompt set version. A stopped run is listed and marked,
  such as "Stopped after 140 of 200 answers".
- The legend in sentences: "Tribelt is named first", "Tribelt is named, but not first", "Tribelt is
  not named", "The answer failed".
- Download the shown answers as CSV, for everyone signed in.
- A square opens One answer.

### One answer

> **Gemini 2.5 Flash, with web search**
> The answer to "We are building a tunnel oven for an industrial bakery. What kind of conveyor belt
> should we use, and which manufacturers can supply it?" in the Prompt run of 1 October 2026.

1. **The answer**, in full.
2. **Companies named**, in order, with Tribelt highlighted.
3. **Sources cited**, with tribelt.nl's addresses marked.
4. **Checked against the fact sheet**, in two lists: "Wrong about Tribelt" (claims that contradict
   the fact sheet) and "Left out" (facts the answer did not mention).
5. **Read by** Haiku 5.5, or Gemini 2.5 Flash Lite when Haiku could not.

### AI readiness

> **AI readiness**
> What each website tells search engines and AI assistants about itself, scored out of eight
> checks. A site that passes more of them is easier for an assistant to read, understand, and cite.
> *From each site's newest scan, checked again on 27 October 2026. 65 sites.*

1. **Tribelt's checks.** The eight, each passed or not, each with a sentence on what it is and why
   it matters ("llms-full.txt is the whole site as one plain text file, so an assistant can read it
   in one go"). A failed check links to the recommendation that fixes it. The crawler check links
   to tribelt.nl's site page.
2. **How the sites compare.** A bar per check: how many of the 65 sites pass it, with Tribelt's
   result beside it.
3. **Every site.** A sortable table: name, score, a tick or cross per check, and the platform, with
   a filter for competitors, group companies, or all. A row opens the site page.

### One site

> **Ammeraal Beltech's AI readiness**
> ammeraalbeltech.com passes 5 of the 8 checks.
> *From scan 3 of 12 October 2026.*

1. **The eight checks, with the evidence**, such as "A sitemap at /sitemap.xml lists 412 pages, in
   Dutch, German, and English."
2. **Crawler policy.** Each search and AI crawler: what it does for the company, and whether
   robots.txt allows it, blocks it, or does not mention it.
3. **The files as found**: robots.txt, llms.txt, llms-full.txt, and the sitemap.
4. **The structured data found**, with product structured data shown but not scored.
5. Links to its competitor page (if it has one) and its scan history.

### Search terms

> **Search terms**
> The words buyers use when they look for metal conveyor belts, and how many of them tribelt.nl
> uses. A page that uses a buyer's words is more likely to be found for them.
> *From the search term list (387 terms, last reviewed on 27 October 2026) and each site's newest
> scan.*

1. **Three figures:** 387 terms in the search term list; 45 used on tribelt.nl (target 90); 342
   used only by competitors.
2. **By language.** Dutch, German, and English: terms in the list, used by tribelt.nl, used only by
   competitors.
3. **By belt family.** Each family: Tribelt's own names, its terms and how many tribelt.nl uses,
   and a link to its page.
4. **Terms competitors use and tribelt.nl does not.** The top 15 by the number of competitors that
   use them. Each opens its term page and shows its recommendation if there is one.
5. **See every term.**

### Every term

A table of term, language, belt family, intent tier, the competitors that use it, and whether
tribelt.nl does (with the page), with a filter per column and a search. A row opens its term page.
Hand-added terms are marked.

### A belt family

> **Eyelink belts**
> Tribelt sells these as [its names]. The terms buyers use for them, and the names competitors sell
> them under.

1. **Terms, by language**, each marked used or not on tribelt.nl.
2. **Who sells them**: each competitor and its name for the belt.
3. **AI answers about eyelink belts**: the share that names Tribelt per scope, and a link to them.

### One term

> **Ösenband**
> A German term for eyelink belts, of the product intent tier. 12 of the 69 competitors use it, and
> tribelt.nl does not.

1. **Where competitors use it**: each company, the pages it is on, and how often. A scan from
   before pages were recorded says "Scanned before pages were recorded".
2. **On tribelt.nl**: the pages that use it, or "No tribelt.nl page uses this term."
3. **Related terms** of the same family and language.
4. **The recommendation** to add it, if a Recommendation run wrote one, with the page it names.

### Competitors

> **Competitors**
> The companies that make metal conveyor belts Tribelt's prospects could buy instead, and how each
> compares with Tribelt in AI answers, AI readiness, and search terms.
> *69 competitors on the competitor list. Newest scans up to 27 October 2026.*

One table: name, country, the belt families it sells, the share of AI answers that name it, its
readiness score, the search terms it uses, and when it was last scanned. Tribelt is pinned at the
top as the row to compare with. It sorts by any column, filters by belt family and country, and
searches by name. A row opens the competitor.

### A competitor

> **Ammeraal Beltech**
> ammeraalbeltech.com, on the competitor list since 3 October 2026. It sells eyelink and open
> spiral belts.

1. **Against Tribelt.** Paired bars: AI answers that name it, web answers that cite it, readiness
   checks passed, and search terms used.
2. **Where it is ahead of Tribelt.** A sentence per measure it leads on, written by rules from the
   gaps ("Named in 16% of AI answers, against Tribelt's 5%"; "Has llms-full.txt, which tribelt.nl
   does not").
3. **What it sells, and what it calls it**: each belt family with its own product names.
4. **Search terms it uses and tribelt.nl does not**, each linking to its term page.
5. **In AI answers**: how often each model names it, with a link to those answers.
6. Links to its AI readiness page, its scan history, and its website.

### Its scan history

Each scan, with its date, the pages read, and the readiness score. Picking two compares them: the
terms gained and lost, and the checks that changed.

### Recommendations

> **Recommendations**
> Changes to make on tribelt.nl, ranked by benefit against effort. Some apply in general; the rest
> come from the latest Recommendation run and name a specific page, search term, or site.
> *From the Recommendation run of 28 October 2026, on the Prompt run of 1 October and the tribelt.nl
> scan of 27 October.*

1. **The benefit against effort chart**, as in 1.12.0.
2. **The list**, one ranking (ADR 0014). Each row: a short title, where on tribelt.nl, its kind,
   benefit and effort, the target it improves, and "New" when it is new since the last run. A
   filter narrows it to one kind: Search terms to add, Pages to add, Sites to get listed on, Fact
   sheet corrections, AI readiness, General.
3. **Done**, folded away: each with when and how it was marked done.

### One recommendation

> **Use "Ösenband" on the German eyelink page**
> Add the term to tribelt.nl/de/oesenbaender. 12 competitors use it, and buyers search for it when
> they look for a product.

1. **Where, and how**: the page, and a suggested sentence or place on it, worded by the model.
2. **Why it ranks here**: the rules' evidence, and how benefit and effort were scored. When the
   model moved a score, its reason ("Effort raised from 1 to 2: the German page has no section
   about applications yet").
3. **The target it improves**, as a progress bar.
4. **Steps**, a checklist, such as the 14 pages that still need a Markdown version.
5. **When it counts as done**, such as "The next tribelt.nl scan finds Ösenband on this page", and
   where it stands now.
6. **The leader to look at.**

### How this tool works

> **How this tool works**
> What the Findability Explorer measures, where each figure comes from, and how often it is
> refreshed.

1. **A section per process**, Asking AI assistants, Checking websites, Search terms, and
   Recommendations, each with: what it does, what it reads, how often it runs, roughly what it
   costs, where its results are, and where administrators run it.
2. **The five targets**, and why each matters.
3. **Words used in this tool**: the glossary terms a reader meets (Prompt run, scope, AI share of
   voice, citation share, leader, gap, scan, and so on).

## 5. Manage, page by page

### Activity

> **Activity**
> What the Findability Explorer is doing now, what it will do next, and what is waiting for you.

1. **Running now.** A row per piece of work ("Prompt run, 140 of 200 answers, $0.81 so far, about
   $0.34 to go"), with its Stop button. Stop says first what it keeps: "Stopping keeps the 140
   answers so far. You can finish the run later with a retry." Empty: "Nothing is running."
2. **Coming up.** The next scheduled Prompt run and its estimate; whether the term review is on;
   how many scans wait in the queue.
3. **Waiting for you.** Term candidates and competitor candidates to review; "New data since the
   last Recommendation run"; a missing key, and what cannot start without it.
4. **This month's spending.** A bar each for OpenRouter and Anthropic: dollars spent against the
   budget, split by kind of work.
5. **Recent work.** The last 20 pieces of work: how each ended (done, stopped by hand, stopped by
   the budget, failed), its cost, and a link.

### Prompt test

> **Prompt test**
> Ask AI assistants the buyer questions, and see what a run costs before you start it.

The top card: "A run asks 20 questions of 6 models, each from its own knowledge and with web
search: 240 answers. It will cost about $1.32, from the last run. $8.68 of this month's $10 is
left." with **Start a Prompt run**. Its subpages:

- **Questions.** The published prompt set, grouped by belt family and scope, with an English
  translation beside each question. The draft edits, adds, and removes questions and adds a scope (a
  language, with or without a region). The publish panel says what changes and what it costs:
  "Adds 5 questions in French. A run will cost about $0.33 more, or $3.96 more a year on a monthly
  schedule." The versions are listed.
- **Models.** The model list: provider, model, price per run, free or not, and Remove. **Add a
  model** opens OpenRouter's catalogue with a search, a provider filter, and a switch for free
  models, sorted by price; each provider's cheapest is marked Recommended. The new estimate shows
  before saving.
- **Schedule.** Off, monthly, or weekly, the next run's date, and the cost over a year.
- **Fact sheet.** The text and its versions.
- **Runs.** Each run: date, prompt set version, models, answers, how it ended, and cost; **Retry
  the failed answers** and **Read the failed answers again**, each with its estimate; Stop.

### Website scans

> **Website scans**
> Read the websites of tribelt.nl and its competitors: their pages, the search terms they use, the
> belts they sell, and their AI readiness.

1. **Four ways to start**, each with its estimate beside the button:
   - **Scan tribelt.nl**: "Reads up to 60 pages. Product matching costs about $0.02; reading pages
     is free. It takes about five minutes."
   - **Scan one site**: pick a company or type an address, and the number of pages (1 to 100).
   - **Rescan competitors**: all, those not scanned in 30 days, or those never scanned ("41 scans,
     about $0.85, roughly 25 minutes with three at a time").
   - **Check AI readiness again**: "Fetches robots.txt, the llms files, and the sitemaps again for
     every site's newest scan. Free."
2. **The queue.** Scans running, each with Stop; scans waiting, each with Remove, and **Remove all
   waiting scans**.
3. **Recent scans**, each opening its scan page.
4. **Settings**: the default number of pages; product matching on or off; the matching model
   (Haiku by default) and the fallback when no Anthropic key works. Each with a sentence on what it
   changes.

### Competitor list

> **Competitor list**
> Which companies count as competitors. AI answers keep naming new companies; this is where they are
> added or dismissed.

1. **The list.** Add a company, paste several, edit the name, website, or country, archive and
   restore.
2. **Candidates from AI answers.** How many wait; **Check the candidates** (Haiku), with its
   estimate and Stop; add, dismiss, or split each one, and undo.
3. **Check the list.** Haiku checks the companies on the list against the competitor definition
   and archives those it is sure do not compete, each with its reason and Restore.
4. **The competitor definition.** The text Haiku is given, edited here; each save is a version.
   Saving says "69 competitors and 383 candidates were checked against an older version" and
   offers **Check again** with its estimate.

### Search term list

> **Search term list**
> The list of search terms buyers use for metal conveyor belts. Scans find new phrases, and the term
> review decides which join the list.

1. **Term review.** The automatic switch, **Review now** with its estimate, Stop, this month's
   spend, and how many wait.
2. **Waiting phrases.** Accept, reject, and undo.
3. **The list.** Add a term (term, language, family, intent tier) and remove one. Hand-added terms
   are marked.

### Recommendations

> **Recommendations**
> Write recommendations from the latest data, mark the ones a scan cannot check, and set the
> targets.

1. **Start a Recommendation run.** Its inputs ("the Prompt run of 1 October, the tribelt.nl scan of
   27 October and the search term list of 27 October"), a warning when one is out of date
   ("tribelt.nl was last scanned before the newest Prompt run"), the model, its estimate, and Stop.
2. **Past Recommendation runs**: what each added, replaced, and kept.
3. **To mark by hand**: open recommendations no scan can check, such as "Submit tribelt.nl to Google
   and Bing", or a listing on a site that blocks fetching.
4. **Edit a recommendation**: a generated one's wording and scores. An edited one is kept when the
   next run replaces the others.
5. **Targets**: the five values.

### Keys and spending

> **Keys and spending**
> The API keys the Findability Explorer uses to ask language models, and how much it may spend each
> month.

1. **Getting started**, while a key is missing: add an OpenRouter key (it pays for the Prompt test);
   add an Anthropic key (it pays for product matching, reading answers, the reviews, and the
   Recommendation runs); set both budgets.
2. **OpenRouter**: the key, Test, OpenRouter's own record of the month's spend, and the monthly
   budget in dollars.
3. **Anthropic**: the key, the optional Admin key for usage reports, Test, and the monthly budget in
   dollars.
4. **Events**: the log.

## 6. The Recommendation run

1. **Rules find the gaps**, from the newest complete Prompt run, each company's newest scan, and the
   search term list:
   - **Search terms**: terms in the list that competitors use and tribelt.nl does not.
   - **Pages**: a belt family or application with no tribelt.nl page in a language, where
     competitors have one.
   - **Listings**: sites the web answers cite that name competitors but not Tribelt.
   - **Fact sheet corrections**: claims marked "wrong about Tribelt" by the fact check.
   - **AI readiness**: each failed check, with its specifics (the pages with no Markdown version,
     the languages missing from the sitemap).
2. **Rules score each gap** from 1 to 5. Benefit grows with the evidence: how many competitors use
   a term, its intent tier, the belt family's gap in AI answers, how often a listing is cited.
   Effort follows the kind: a term on an existing page is small, a new page larger.
3. **The model places and words each gap** from the newest tribelt.nl scan's page text: the page it
   belongs on, and a suggested sentence or place. It may move a score by one point, and must then
   give its reason. A gap it cannot place stays, with its page left blank and "Where to put this is
   for the web agency to decide".
4. **The run replaces** the open generated recommendations, keeps the done and edited ones, and
   marks the new ones "New".
5. **Rules decide done**: the next tribelt.nl scan finds the term on the named page; a page for
   that family or application in that language; the listing's page names Tribelt or links to
   tribelt.nl (fetched in the readiness check); the readiness check passes. Anything else is ticked
   by hand.

The readiness recommendations in code stay the general ones; their specifics become their steps.

## 7. Changes to the backend

1. **Stopping.** A stop operation for Prompt runs (reusing the budget halt), scans (a cancel per
   scan, and removing a queued scan), rescans (every scan of the batch), readiness checks, term
   reviews, candidate reviews, competitor checks, and Recommendation runs. Each records "stopped
   by hand".
2. **Estimates and budgets.** A dollar estimate for every start operation. The Anthropic usage is
   priced in dollars from the token counts already stored, against a budget set in Manage. Start
   operations return the estimate and what is left, and the start of a Prompt run checks it.
3. **The prompt set** moves from code to the database: drafts, published versions, and scopes as
   data. The current set is version 2's 20 questions.
4. **The model list** replaces `ChooseModels`: stored models, a catalogue endpoint with prices and
   the cheapest per provider, and an estimate per change.
5. **Pages per term.** Scans record the page of each known term, and keep tribelt.nl's page text.
6. **The Recommendation run**: rules, the model call, storage of runs and generated
   recommendations, edits, and the done checks, including fetching listings.
7. **The fact check** separates wrong from left out; the newest run is read again.
8. **The competitor definition** moves from `TargetMarket` to stored versions, and the check
   version follows it.
9. **Hand-added terms** in the search term list.
10. **History and change**: figures per complete Prompt run and tribelt.nl scan for "Over time" and
    "What changed", with the version breaks.
11. **Activity**: one endpoint listing what runs, what is scheduled, what waits, and the month's
    spending by kind.
12. **Data**: the plastic and fabric makers archived; the oven and freezer patch from the handoff
    landed first, as it changes the competitor list.

## 8. ADRs to write

- **The prompt set is edited and published in the app**, superseding the fixed set of 0004.
- **The Prompt test asks the model list**, superseding the cheapest model per provider of 0004.
- **Language model spending is budgeted in dollars**, amending 0007's token limit.
- **The Recommendation run writes recommendations, and rules decide when they are done**, amending
  0014.
- **The competitor definition is edited in the app**, amending 0009.
- **Search terms can be added by hand**, amending 0003.

## 9. How it gets built

1. **The prototype.** The whole site, both views, as one interactive artifact on claude.ai: every
   page reachable, the filters and pickers working, Start and Stop changing what Activity shows, the
   estimates and budget warnings working, and a switch between a reader and an administrator. It
   runs on the export of 28 October with the plastic and fabric makers archived, and sample data
   for what the app does not have yet (history, Recommendation runs, running work). The owner goes
   through it page by page, and its copy and behaviour become final there.
2. **The plan**, from this spec and the reviewed prototype, splitting the work so agents can build
   in parallel: the frame and navigation; the Results pages; the Manage pages; and each backend
   change above.
3. **The build**, in worktrees, with the coverage gate, `task check`, and the e2e tests, through the
   lock script.

## 10. Not in this redesign

- More languages beyond the prompt set (decision 12).
- The Viewer Key.
- Recommendations from Buyers' analytics.
- A design canvas of artboards; the prototype replaces it.
