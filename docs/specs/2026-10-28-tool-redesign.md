# Redesigning the Findability Explorer as a tool

Release 1.12.0 laid every page out as a poster: one headline, at most three figures, one chart and
one list. That reads well on a slide, but the app is being handed to Tribelt, whose staff will read
it every week and whose administrator will run it. They need to know what each page shows, where
its figures come from, how old they are, and how to start, stop, tune, and pay for the work behind
them.

**The app becomes a tool with two views, Results and Manage, both organised around the same four
processes.** This spec describes the redesign as it was reviewed in an interactive prototype (an
artifact on claude.ai, reviewed page by page between 28 and 29 October 2026, version 24). Where this
text and the prototype differ, the prototype's behaviour and copy win, and this text is corrected.

The audience for Manage is a marketer with no technical background, so every button says what it
does, what it costs, and what stopping it keeps.

Figures in the examples below are placeholders such as *n*; the repository is public, and the live
figures stay in the app.

## 1. Decisions

| # | Decision |
|---|---|
| 1 | Two views. **Results** is for everyone signed in; **Manage** is for administrators only. |
| 2 | Both views are organised by process: the Prompt test (asking AI assistants), website scans (checking websites), search terms, and recommendations. Each Results page links to its Manage page for administrators, and back. |
| 3 | The poster rule goes. Every page has the same frame (section 4) and then the sections it needs: tables, filters, and charts where they help. Things that fit side by side sit side by side; a page is not spread out. |
| 4 | Same look: the colours, fonts, and logos stay. Mist blocks with rounded corners, an orange dash before section titles, pills used sparingly. |
| 5 | Every piece of work can be stopped, by a button named for it ("Stop the Prompt run"). There is no umbrella word such as job or task. Stop first says what it keeps. |
| 6 | Every piece of work shows a dollar estimate before it starts. When the estimate is more than the budget has left, a line says so before the button is pressed, and the button asks for a second click. The budget stop stays. |
| 7 | Two monthly budgets, both in dollars: OpenRouter (the Prompt test) and Anthropic (everything a Claude model does). The Anthropic budget replaces the `LLM_MONTHLY_TOKENS` limit. |
| 8 | Results always show the newest **complete** Prompt run. A stopped or failed run is kept and listed in Manage, where a retry finishes it. |
| 9 | The Prompt set is edited as a draft, grouped by belt family, and published as a new version. Publishing says what changes and what it costs per run and per year. A new scope (a language, with or without a region) is made from the same editor. |
| 10 | The Model list is chosen from OpenRouter's whole catalogue of text models, read again daily. The cheapest model of each of the five providers whose assistants buyers use most (OpenAI, Google, Anthropic, Mistral, DeepSeek) is marked Recommended and listed first. A provider can have several models; each model is its own column in Results. |
| 11 | Trends over time break where the Prompt set or the Model list changed, with a note saying which. The charts can be filtered to one Prompt set version. A run that got fewer than half its responses is shown as a partial run, with a note. |
| 12 | Other languages follow later: the readiness check "pages in Dutch, German, and English" and the search term list stay three languages, built so a language can be added. |
| 13 | The hand-written recommendations stay in code. A **Recommendation run** writes the rest: rules find and score the gaps, a Claude model places and words each one, and may move a score by one point with a written reason. |
| 14 | Kinds of generated recommendation: search terms to add and where, pages to add, sites to get listed on, fact sheet corrections, and the specifics of AI readiness. |
| 15 | Whether a recommendation is done is decided by rules, never by a model. Only what no scan can see is ticked by hand, and a tick can be undone. |
| 16 | A Recommendation run starts by hand. Activity says when its inputs are newer than the last run. A new run replaces the open generated recommendations, keeps those done or edited, and says beforehand how many it replaces. |
| 17 | The fact check separates "wrong about Tribelt" from "left out"; only wrong claims become fact sheet corrections. |
| 18 | Scans record which page each term is on, keep tribelt.nl's page text for the Recommendation run, and keep the files they fetched (robots.txt, the sitemap, llms.txt, llms-full.txt) to read in the app. |
| 19 | The competitor definition is edited in Manage, with versions. Saving offers a recheck with its estimate; it does not start one. |
| 20 | Administrators can add and remove search terms by hand. A hand-added term is marked as such. |
| 21 | The term review stays automatic, with an on and off switch, its month's spend, and Stop. |
| 22 | Makers of plastic or fabric belts are archived. AI responses still show them under "Who the responses name", marked "Not a competitor". |
| 23 | Buyers is dropped. `/buyers` redirects to the Overview. |
| 24 | Tribelt adds its own OpenRouter and Anthropic keys. Everyone signs in with an auth-api account; the Viewer Key waits. |
| 25 | "Numbers to watch" becomes **Goals**, edited in Manage, Recommendations. |
| 26 | Every page opens with one sentence on what it shows. Help replaces How it works: short topics, searchable, each linking to where the thing is done. |
| 27 | **Every Claude step has its model and instructions in the app** (section 8), with versions. The one exception: the instructions for reading responses stay in code, so every run is read the same way; only that step's model is chosen in the app. The data each step gets and the format of its reply stay in code. |
| 28 | **An empty response counts as failed.** A model that returns no text (for example a reasoning model that spent its token limit) gets a failed response, which a retry asks again. It is never read, and never counts as naming anyone. |
| 29 | **The history of the Prompt test is imported** from the live database with its Prompt set versions (section 9). |
| 30 | The words: "response", never "answer"; "search term", never "phrase"; "best competitor", never "leader"; "buyer intent", never "intent tier"; "goal", never "target". British spelling, the Oxford comma, no colons or middle dots in headings or buttons, no em dashes. |

## 2. Words

The glossary (`CONTEXT.md` in the course repo) holds these words, and the app uses them the same way:

- **Response**: what an AI assistant replies to one question of a Prompt run.
- **Search term**: what a buyer types into a search engine; the search term list holds them.
- **Buyer intent**: the kind of search a search term or question represents: specification, product,
  application, problem, or broad. Each has one plain sentence of explanation in the app.
- **Best competitor**: the competitor with the best value on a measure in the latest data. A tie goes
  to the name first alphabetically. Tribelt, its group companies, and the mirror are never it.
- **Typical competitor**: the middle value across all competitors on the list.
- **Goal**: a value Tribelt is measured against; there are five.
- **Scope**: a language, with or without a region, that a question is asked in.
- **Claude step**: one piece of work a Claude model does, such as reading responses.
- **Partial run**: a Prompt run that got fewer than half its responses.

## 3. The site map

### Results

The top bar reads Overview, AI responses, AI readiness, Search terms, Competitors, Recommendations,
and Help, with Manage on the right for administrators and the account menu after it.

| Page | Address | Reached from |
|---|---|---|
| Overview | `/` | the top bar |
| A goal | `/goals/:measure` | a row of Goals on the Overview |
| AI responses | `/ai-responses` | the top bar |
| Every response | `/ai-responses/all` | AI responses, a goal page |
| One response | `/ai-responses/:id` | a square of Every response, a cell of the family and scope table, a competitor page |
| AI readiness | `/ai-readiness` | the top bar |
| One site | `/ai-readiness/:id` | a row of Every site, a competitor page, a readiness recommendation |
| Files found on a site | `/ai-readiness/:id/files` | one site |
| Search terms | `/terms` | the top bar |
| Every term | `/terms/all` | Search terms |
| A belt family | `/terms/families/:family` | Search terms, AI responses, a competitor page |
| One search term | `/terms/:id` | Every term, a belt family, a competitor page, a term recommendation |
| Competitors | `/competitors` | the top bar |
| A competitor | `/competitors/:id` | Competitors, Who the responses name |
| Its scan history | `/competitors/:id/history` | a competitor page, a site page |
| Recommendations | `/recommendations` | the top bar, the Overview |
| One recommendation | `/recommendations/:id` | Recommendations, a goal page, a term page, a failed check |
| Help | `/help`, and a topic at `/help/:topic` | the top bar, every page's "How this is measured, in Help" |

### Manage

Manage has its own menu down the left. A section folds out to its subpages when it is open. A
section's count (running work, waiting candidates, waiting search terms) shows on the section while
it is folded, and moves to the subpage it belongs to when the section is open.

| Section | Address | Subpages |
|---|---|---|
| Activity | `/manage` | |
| Prompt test | `/manage/prompt-test` | Questions, Models, Schedule, Fact sheet, Runs, Reading responses |
| Website scans | `/manage/scans` | Scan queue, Recent scans, Scan settings, Matching products; a scan at `/manage/scans/:id` |
| Competitor list | `/manage/competitors` | Companies (one at `/manage/competitors/companies/:id`), Candidates, Check the list, Competitor definition, Tidying names, Checking companies |
| Search term list | `/manage/terms` | Term review, Waiting terms, Add or remove terms, Reviewing terms |
| Recommendations | `/manage/recommendations` | Recommendation runs, Tick off yourself, Edit a recommendation, Goals, Writing recommendations |
| Keys and spending | `/manage/keys` | OpenRouter key, Anthropic key, Budgets, Events, Models and instructions |

Each subpage has its own address under its section, such as `/manage/prompt-test/questions`. The
Claude steps (section 8) are at `/manage/claude-steps/:step`, listed in the menu under the section
each belongs to, and all six at `/manage/claude-steps`.

### Old addresses

Every address of 1.12.0 and earlier redirects: `/numbers/:measure` to `/goals/:measure`,
`/ai-search` to `/ai-responses`, `/ai-search/answers` to `/ai-responses/all`, `/ai-search/readiness`
to `/ai-readiness`, `/buyers` to `/`, `/scan/:id` to `/manage/scans/:id`, `/settings` to
`/manage/keys`, `/settings/prompt-test` to `/manage/prompt-test`, `/settings/terms` to
`/manage/terms`, and `/settings/competitors` to `/manage/competitors`. The redirects 1.12.0 already
has stay.

## 4. The page frame

Every page, in both views, has, in this order:

1. **A back link** to the page it came from, on a detail page. In Manage it goes to the parent page.
2. **The title.**
3. **One sentence on what the page shows.**
4. **The data line**: where its figures come from and when, such as "From the Prompt run of *date*:
   20 questions to 5 models, each from the model's own knowledge and with web search, 200 responses
   in all. Prompt set version 2."
5. **For administrators, a link to its Manage page** on the right; on a Manage page, a link to its
   Results page.
6. **Its sections.**
7. **"How this is measured, in Help"**, a link to its process's topic.

Empty states are sentences that say what is missing and, for administrators, where to start it.
Sample data the app does not have yet is never shown in the app; the prototype marks its samples.

Manage pages are built from widgets on a twelve-column grid: a mist block with a title, an optional
action on the right, a body, and a footer for its buttons. A widget alone on a row stretches to fill
it. A long editor can keep its companion widget (such as Publish) in view while scrolling.

Tables sort by any column, search when they have more than twelve rows, and show a limited number of
rows with "Show all" where the list is long. Headings wrap when a table is short of room; row buttons
wrap rather than run past the table's edge. Folds stay open or closed when the page redraws.

## 5. Results, page by page

### Overview

> **Overview**
> How easily buyers find Tribelt in search and AI assistants, measured against five goals.
> *From the Prompt run of date and the scans up to date.*

1. **Summary.** One sentence written by rules from the data.
2. **Goals.** A row per goal: its name, Tribelt's value, the goal, the best competitor and its value,
   and a progress bar marking Tribelt now, the best competitor, and the goal. Each row opens its goal
   page. The five: AI responses that name Tribelt; AI responses about the Twente region name
   Tribelt; web responses that cite tribelt.nl; AI readiness checks passed; search terms tribelt.nl
   uses.
3. **Numbered tiles**, one per part of the app (AI responses, AI readiness, search terms,
   competitors, recommendations), each with its key figure, opening that part.
4. **Latest runs and scans.** A row per process: when it last ran, what comes next, and, for
   administrators, Manage.

### A goal

> **AI responses that name Tribelt**
> The share of a Prompt run's responses that name Tribelt.

The summary says whether Tribelt or the best competitor is ahead, and the goal. Then, side by side,
**Tribelt against the competitors** (bars for Tribelt, the typical competitor, the best competitor,
and the goal) and **Over time** (one point per run, with the Prompt set filter of section 5's AI
responses, or per tribelt.nl scan for readiness and terms). Then **Broken down** (by model, scope, and
belt family; by check; by language) and **What improves it**, the recommendations aimed at it.

### AI responses

> **AI responses**
> How often AI assistants name and cite Tribelt when buyers ask which metal conveyor belt to use and
> who makes it.

1. **Three figures**: responses that name Tribelt, responses about the Twente region that name
   Tribelt, and web responses that cite tribelt.nl, each against its goal.
2. **See every response**, a button.
3. **Over time.** A line per language and one for all, with the goal, and the end labels joined to
   their last points. Above it, the **Prompt set filter**: All versions, Version 2 (current), Version
   1. All versions shows a shaded band labelled with the version where the questions changed; one
   version shows only its runs, with a line saying when it was published, how many questions it
   asks, and what it asks. A partial run has a dashed ring and a note: "a partial run of version 1.
   Only *n* of its *n* requests got a response (the rest failed because the OpenRouter key was
   refused), and *n* of those could be read, so its figure rests on few responses." The figures as a
   table, folded away.
4. **Who the responses name.** Companies ranked by the share of responses that name them. Tribelt
   highlighted, group companies marked "Group company", archived makers "Not a competitor".
5. **By model**, side by side with **the questions asked**: a row per model ("OpenAI, gpt-oss-20b")
   with named from its own knowledge, named with web search, and cited; each question in its own
   language with an English translation, per belt family, folded.
6. **By belt family and scope.** Rows "Eyelink, Twente, application" with the buyer intent explained
   on hover; a cell opens Every response filtered to it.

### Every response

> **Every response**
> Each square is one model's response to one question. Its colour shows whether the response names
> Tribelt, and where.

A grid with models as column groups and the two modes under each. Filters: search mode, scope, belt
family, model, and names Tribelt; a run picker by date and Prompt set version, with a stopped run
marked. The key sits inside the grid's panel, with "*n* responses shown". Download the shown
responses as CSV. A square opens One response.

### One response

> **OpenAI, gpt-oss-20b, with web search**
> The response to "question" in the Prompt run of date.

1. **The response**, in full, with its Markdown rendered.
2. **Companies named**, in order, with Tribelt highlighted and its place ("Tribelt is named, number
   2").
3. **Sources cited**, with tribelt.nl's addresses marked.
4. **Checked against the fact sheet**: "Wrong about Tribelt" and "Left out".
5. **Checked by** the model that read it, or "No model has checked this response yet."

### AI readiness

> **AI readiness**
> How well search engines and AI assistants read, understand, and cite each website, scored out of
> eight checks.

1. **Tribelt's checks.** The eight, each passed or not, with a sentence on what it is and why it
   matters. A failed check links to the recommendation that fixes it.
2. **How tribelt.nl compares with the competitors.** Per check: tribelt.nl passes or fails, and how
   many competitors pass, as a bar.
3. **Every site.** Name, score, a tick or cross per check, and the platform; 15 rows with Show all;
   a filter for competitors or all sites. A row opens the site page.

### One site, and the files found

One site shows, in two columns, the eight checks with their evidence, the structured data found,
and the files as found (left), and the crawler policy (right): each search and AI crawler, what it
does for the company, and whether robots.txt allows it, blocks it, or does not mention it.

**Files found on a site** shows the files a scan fetched, one tab each (robots.txt, sitemap.xml,
llms.txt, llms-full.txt), in a viewer with line numbers and a search that highlights its matches.
The file's lines, size, and matches are listed in its header, with Copy and Download. A scan picker
shows the files of an earlier scan. Files a scan did not find are listed as not found.

### Search terms

> **Search terms**
> The words buyers use to look for metal conveyor belts, and how many of them tribelt.nl uses.

1. **Three figures**: terms in the list, used on tribelt.nl (against its goal), used only by
   competitors.
2. **Search term adoption over time**: the share of the list tribelt.nl uses, at each scan, for all
   terms and per language.
3. **By language** and **by belt family** (each family as "*n* of 5" with its own names), side by
   side where they fit.
4. **What buyer intent means**, folded: one sentence per kind.
5. **Terms competitors use and tribelt.nl does not**, the top ones, each opening its term page.

**Every term** is a table of term, language, belt family, buyer intent, the competitors that use it,
and whether tribelt.nl does, with filters and a search; hand-added terms are marked. **A belt family**
has one purpose sentence, its terms by language, Tribelt's own names, who sells it, and the AI
responses about it. **One search term** says its language, family, and buyer intent in a sentence, and
shows, in two columns, where competitors use it and on tribelt.nl, related terms, and its
recommendation.

### Competitors

One table: name, country, the belt families it sells (as "*n* of 5", with the names on hover), the
share of AI responses that name it, its readiness score, the search terms it uses, and when it was
last scanned. Tribelt is pinned at the top. A summary line counts the competitors and the archived
companies, which can be shown. A row opens the competitor.

**A competitor** shows, side by side where they fit: against Tribelt (paired bars), where it is ahead
of Tribelt, what it sells and what it calls it, the search terms it uses and tribelt.nl does not, and
how often each model names it. **Its scan history** lists each scan and compares two.

### Recommendations

> **Recommendations**
> Changes to make on tribelt.nl, ranked by benefit against effort.

1. **Benefit against effort**, a five-by-five grid of bubbles; a bubble narrows the list to its cell.
2. **The ranked list**, scrollable, with a count and a scroll cue; a filter by kind.
3. **Done**, folded away.

**One recommendation**: where and how; why it ranks here; the goal it improves (written correctly
whether Tribelt is ahead of or behind the best competitor); steps; when it counts as done; and, for
one no scan can check, Mark as done.

### Help

Topics in nine groups (Getting started, Reading the results, the Prompt test, Websites and AI
readiness, Search terms, Competitors, Recommendations, Costs and budgets, Words used in this tool),
one group shown at a time with a search over all. Topics sit two across. Each topic is a few plain
sentences and links to where the thing is done (Manage links only for administrators). The words are
a two-column glossary. Figures in topics, such as what a run costs, come from the app's own
estimates.

## 6. Manage, page by page

Every section opens on an index page: its key figures, its start buttons first, and a card per
subpage with that subpage's figure and a link.

### Activity

> **Activity**
> See what is running, what comes next, and what is waiting for you.

1. **Running now**: a widget per piece of work with its meter, cost so far and to go, and its Stop.
2. **Coming up**: the next scheduled Prompt run and its estimate; whether the term review is on; the
   scan queue.
3. **Waiting for you**: search terms and competitor candidates waiting; new data since the last
   Recommendation run; a missing key.
4. **The two budgets**, each spent against its limit.
5. **Spending by kind of work**: Prompt runs, scans, reading responses, the term review, candidate
   checks, and Recommendation runs, with the key that paid.
6. **Recent work**: the last 20 pieces of work, how each ended (done, done with failed responses,
   stopped by hand, stopped by the budget, failed), what it did, and its cost.

### Prompt test

The index: last run, next run, cost per run, budget left; **Start a Prompt run** with its estimate
first; then a card per subpage, including Reading responses.

- **Questions.** One editor grouped by belt family: each question with its scope and Remove on one
  line and its text below, at full width. Under each family, **+ Add a question** opens a row: a
  scope picker that starts on a scope that family has no question in yet, ending with "A new scope…",
  which shows a language and a region; then the question; then Add the question and Cancel. After a
  question in a new scope, a note offers to add one in that scope for every other belt family. The
  published set is a fold above the editor. Beside it, **Publish as version *n***, kept in view, says
  what changes ("Adds *n* questions (*n* in French) … Adds 1 scope (French, Lyon). A run will cost
  about $*x* more, or $*y* more a year on a monthly schedule.") with Discard the draft. The version
  history lists every version with its questions, and restores one into the draft.
- **Models.** The model list (provider, model, cost per run, free or not, Remove) beside **Save the
  model list**, which says the new cost and that the charts will mark the date. **Add a model**:
  every text model on OpenRouter, read again daily, with the date it was listed; a search, a provider
  filter with counts, and Free only; the recommended five first, the rest cheapest first; a column
  for when each model arrived on OpenRouter. Models already asked show what their runs cost; the
  rest are estimated from the list price for a typical response (300 tokens in, 900 out), plus the
  model's own web search fee, or OpenRouter's search fee.
- **Schedule.** Off, monthly, or weekly; the next run; the cost over a year. When a schedule would
  cost more than the budget, the warning offers "Set to monthly" and "Change the budget".
- **Fact sheet.** The text, how it is used, and its versions.
- **Runs.** Figures (runs, those needing a retry, the run status); a panel per run with failed
  responses, offering **Retry the *n* failed responses** and **Read them again with Claude Haiku**,
  each with its estimate; all runs with prompt set version, models, responses, how it ended, and
  cost.
- **Reading responses**, a Claude step (section 8).

### Website scans

The index: last scan, budget, readiness checked; then four ways to start, each with its estimate:
**Scan tribelt.nl**, **Scan one site** (a company or an address, and the pages, 1 to 100),
**Rescan competitors** (all, not scanned in 30 days, or never scanned), and **Check AI readiness
again** (free). Then the running work and the cards.

- **Scan queue**: scans running, each with Stop; scans waiting, each with Remove, and Remove all.
- **Recent scans**: each company's newest scan, opening its scan page.
- **Scan settings**: the default number of pages; product matching on or off; the model and
  instructions (a summary and link to the Claude step); the fallback model when no Anthropic key
  works.
- **Matching products**, a Claude step.
- **A scan**: its company, number, date, pages read, matching model, readiness score, and Scan
  again.

### Competitor list

The index: competitors, waiting candidates, the definition's version, the checks running; a card
per subpage.

- **Companies**: the list at full width (archive, restore, open), and below it Add a company and
  paste several. **One company**: details (name, website, country) with Save, archive with an
  optional reason, and where it shows.
- **Candidates**: figures (waiting, added, dismissed, decided by an administrator); **Classify
  automatically** (only the waiting ones, or also recheck what Claude Haiku decided) and **Check the
  candidates** (advice only), each with its estimate; all candidates with a status filter, and Add,
  Dismiss, Split, Back to waiting, and Undo the split on every row. Decisions an administrator made
  are never changed by a model.
- **Check the list**: Claude archives the companies it is sure do not compete, each with its reason
  and Restore.
- **Competitor definition**: the text with its versions; saving offers Check again with its
  estimate.
- **Tidying names** and **Checking companies**, Claude steps.

### Search term list

- **Term review**: Review now with its estimate, the automatic switch, this month's spend.
- **Waiting terms**: a table at full width: term, languages, companies, the suggested belt family
  ("No belt family" when there is none) and buyer intent, the closest term on the list, and Accept,
  Reject, and Undo.
- **Add or remove terms**: add a term (term, language, family, buyer intent, with what buyer intent
  means) and remove one; hand-added terms are listed.
- **Reviewing terms**, a Claude step.

### Recommendations

- **Recommendation runs**: a warning when tribelt.nl was last scanned before the newest Prompt run;
  **Start a Recommendation run** with its inputs, "This replaces *n* open recommendations. Those you
  marked as done or edited stay.", and its estimate; the model and instructions (a summary and link);
  past runs (added, replaced, kept, model, cost).
- **Tick off yourself**: open recommendations no scan can check, each with Mark as done; a Done list
  with Undo.
- **Edit a recommendation**: the generated ones at full width; one opens its wording and scores.
  An edited one is kept when the next run replaces the others.
- **Goals**: the five values, with Save and a toast.
- **Writing recommendations**, a Claude step.

### Keys and spending

- **OpenRouter key** and **Anthropic key**: the key (its name and last four characters), Test the
  key, and Replace the key (a fold with Label and Key, then Save the new key); what it pays for;
  spent this month; the monthly budget. The Anthropic key page adds the optional Admin key for usage
  reports.
- **Budgets**: per key, spent and left, and the budget in dollars with Save.
- **Events**: the log of key changes, tests, and budget changes.
- **Models and instructions**: the six Claude steps (section 8).

## 7. The Prompt set filter and the charts

The line charts draw runs at even spacing, with the date under each point and the time when two runs
share a day. A Prompt set change is a shaded band from the change to the right edge, labelled with
the version. End labels sit in the right margin at their value, pushed at least 17 pixels apart, each
joined to its last point by a short line in the series colour. A run that is alone in the chosen
version sits at the right edge, beside its labels, and the note says "It has one run so far, so there
is no line yet." Charts redraw at their container's width.

## 8. Claude steps

Six steps ask a Claude model, all paid by the Anthropic key:

| Step | Section | Gets (from) | Replies |
|---|---|---|---|
| Reading responses | Prompt test | the fact sheet in use when the run started; the buyer's question; the response; its cited sources | companies, addresses, namesTribelt, citesTribelt, correctness, wrongClaims |
| Matching products | Website scans | the five belt families (fixed); the text of the scanned pages, cut at a length limit | product, family, reason, searchTerms |
| Tidying names | Competitor list | the competitor list; the candidate names with spellings, websites, and response counts | clusters |
| Checking companies | Competitor list | the competitor definition in use; each company's names, websites, response count, and home page text | competitor, sure, reason |
| Reviewing terms | Search term list | the belt families; examples from the list; the search terms to review with who uses them and the closest term | accept, term, language, family, buyer intent, sure, reason |
| Writing recommendations | Recommendations | the gaps with their figures and scores; the tribelt.nl pages; the search term list | title, where, why, steps, doneWhen, benefit, effort, scoreReason |

Each step's page has:

1. **Instructions**: the system message, in an editable box, saved as versions. The data goes in a
   separate message after it, and the reply format is a strict JSON schema in code, so editing the
   text cannot break the step. **Back to the original** restores the text the tool came with. For
   **reading responses**, the instructions are read-only and marked "Set in the code": they change
   only with a release, so runs stay comparable.
2. **Model**: every Anthropic model on the catalogue, cheapest first, Claude Haiku 5.5 the default,
   with the cost of one job of that step with the chosen model and the difference from now, and what
   a change does. Reading responses has its own Save the model and Undo.
3. **What the model gets** (each part with where it comes from) and **What the model replies** (each
   field with what it means), side by side.
4. **The version history** (for reading responses, the model history); restoring copies a version
   into the editor, and saving makes a new version.

A step's cost is the tool's own estimate for Claude Haiku 5.5, scaled by the chosen model's price for
the step's typical mix of tokens read and written. Every other estimate that includes the step
(scans, the Prompt run's reading, the term review, candidate checks, the Recommendation run) scales
the same way. Changing the model for reading responses marks a break on the AI responses charts.
The starting instructions are the live app's prompts, reworded to the app's words; the
Recommendation run's are new.

## 9. The Prompt test history and its import

The live database holds runs of two Prompt set versions. Version 1 asked 90 questions (a supplier,
an application, and a problem question for each belt family, in six scopes: Twente, the Netherlands,
Germany, Dutch, German, English); version 2 asks 20 (one application question per family, in four
scopes). The questions live in code; each stored response keeps its own question text, kind, scope,
and version.

1. **The export** (`handoff/prompt-export` in the course repo) reads every run, response, reading,
   candidate decision, and the company list, read only. Both versions' questions are rebuilt from git
   (commits de6f134 and a5a484c).
2. **The Prompt set becomes data**: version 1 and version 2 are stored as published versions with
   their questions, version 1 published on 26 October 2026 and version 2 on 27 October 2026.
3. **Mapping version 1**: the kind "supplier" becomes the buyer intent "product"; the scopes
   Netherlands and Germany become Dutch with the region Netherlands and German with the region
   Germany, shown only in version 1's history.
4. **Partial runs**: version 1's only run is kept in the history and marked partial.
5. **Empty responses**: existing responses with no text are marked failed during the migration, so
   they are retried, not counted (decision 28).

## 10. The Recommendation run

1. **Rules find the gaps**, from the newest complete Prompt run, each company's newest scan, and the
   search term list: search terms competitors use and tribelt.nl does not; a belt family or
   application with no tribelt.nl page in a language where competitors have one; sites the web
   responses cite that name competitors but not Tribelt; claims the fact check marked wrong; each
   failed readiness check with its specifics.
2. **Rules score each gap** from 1 to 5 for benefit (how many competitors use a term, its buyer
   intent, the family's gap in AI responses, how often a listing is cited) and effort (a term on an
   existing page is small, a new page larger).
3. **The model places and words each gap** from the newest tribelt.nl page text (section 8), and may
   move a score by one point with a reason. A gap it cannot place keeps its page blank, with "Where
   to put this is for the web agency to decide".
4. **The run replaces** the open generated recommendations, keeps the done and edited ones, and marks
   the new ones "New".
5. **Rules decide done**: the next tribelt.nl scan finds the term on the named page; a page for that
   family or application in that language; the listing names Tribelt or links to tribelt.nl; the
   readiness check passes. Anything else is ticked by hand.

## 11. Changes to the backend

1. **Stopping**: a stop for Prompt runs (reusing the budget halt), scans (and removing a queued
   scan), rescans, readiness checks, term reviews, candidate reviews, competitor checks, and
   Recommendation runs, each recorded as "stopped by hand".
2. **Estimates and budgets**: a dollar estimate for every start operation; Anthropic usage priced in
   dollars from the stored token counts, against a budget per key.
3. **The Prompt set** in the database: drafts, published versions with notes, scopes; imported as in
   section 9.
4. **The Model list**: stored models replacing the cheapest per provider; a catalogue endpoint from
   OpenRouter's model list (text in, text out; no aliases or routers), refreshed daily, with prices,
   web search fees, the date each model was listed, and the recommended five.
5. **Empty responses** are failures (decision 28), in new runs and in the migration.
6. **History**: figures per run (all, per language, Twente, citation) with the version and whether it
   was partial; per tribelt.nl scan for readiness and terms.
7. **Claude steps**: stored versions of model and instructions per step, the read-only reading
   instructions, and each call site taking its step's current version; the cost scaling.
8. **Pages per term, page text, and fetched files** in scans.
9. **The Recommendation run**: rules, the model call, storage of runs and generated
   recommendations, edits, and the done checks.
10. **The fact check** separates wrong from left out.
11. **The competitor definition** as stored versions.
12. **Hand-added and removed terms.**
13. **Activity**: one endpoint for what runs, what is scheduled, what waits, and the month's spending
    by kind.
14. **Data**: the plastic and fabric makers archived.

## 12. ADRs to write

- **The Prompt set is edited and published in the app**, superseding the fixed set of 0004.
- **The Prompt test asks the Model list**, superseding the cheapest model per provider of 0004.
- **The model that reads responses is chosen in the app; its instructions stay in code**, amending
  0004 and 0007.
- **Claude steps keep their model and instructions in the app; data and reply format stay in code.**
- **An empty response is a failed response.**
- **Language model spending is budgeted in dollars**, amending 0007's token limit.
- **The Recommendation run writes recommendations, and rules decide when they are done**, amending
  0014.
- **The competitor definition is edited in the app**, amending 0009.
- **Search terms can be added by hand**, amending 0003.

## 13. How it gets built

The prototype is reviewed. The implementation plan (kept outside the repository, as planning
artifacts are) splits the work so agents build in parallel in worktrees, with the coverage gate,
`task check`, and the e2e tests. The whole spec ships by Monday 2 November 2026, 09:00. The work
is tracked as tickets under the course repo's issue #5, each a vertical slice with its blockers: first
the frame, the Manage layout, and the addresses; then, in parallel, the Results pages, Manage, empty
responses failing, stopping a Prompt run by hand, and the history per run; then the Prompt set in the
database with its import (section 9) and its editor, the Model list, the Claude steps (section 8),
dollar budgets and stopping all work, scans that keep more, hand-added terms, the competitor
definition's versions, the fact check's split, and the Recommendation run (section 10). The release
ticket waits for every other ticket.

## 14. Not in this redesign

- More languages beyond the Prompt set (decision 12).
- The Viewer Key.
- Recommendations from Buyers' analytics.
- Automatic translation of new questions into English.
