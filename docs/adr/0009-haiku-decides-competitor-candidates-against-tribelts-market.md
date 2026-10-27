# Claude Haiku decides competitor candidates against Tribelt's target market

ADR-0006 has Claude Haiku check each company the Prompt runs' answers name for whether it is a
competitor of Tribelt, as a suggestion a person acts on. Haiku judged from the names and websites
alone, against one sentence about Tribelt. Many of the companies it called competitors make
conveyor belts, but not for the same purposes as Tribelt at all: rubber belting for quarries,
plastic belts for packaging lines, conveyor systems that buy their belts. A person still had to go
through hundreds of them.

**The check is made against Tribelt's target market, written down once.** The market definition
is `TargetMarket` in `internal/prompttest/domain`, in plain sentences, researched from tribelt.nl's
sector and product pages and the client interviews. Tribelt makes metal conveyor belts (eyelink,
open and closed spiral, Tri-Flex curve-going, chain, ladder, plate link, honeycomb, interlaced wire
and woven baking belts) and their sprockets and drums, for process conveying: heat up to about
1,150 °C, cold in spiral freezers and coolers, wet and hygienic steps, and heavy or abrasive loads
in recycling, waste and water filtration. It sells to industrial food producers, to the builders of
ovens, freezers and lines, and to industry, in Europe, above all the Netherlands, Germany and
English-speaking buyers.

- A competitor makes metal conveyor belts for those process uses, to those customers, or has them
  made and sells them under its own name. A maker anywhere that sells into Europe counts.
- Not competitors: makers of rubber, textile or steel-cord belting for mining and bulk handling;
  makers of fabric, PVC or polyurethane belts for packaging and logistics; builders of conveyor
  systems, ovens or freezers that buy their belts (customers and partners); distributors and
  traders; makers of wire mesh for fencing, screening, sieving or architecture; directories,
  marketplaces and publishers.
- A maker of plastic modular belts for the same food lines makes a substitute, not the same
  product: not a competitor unless it also makes metal process belts, and its reason says so.

The editable fact sheet of the Prompt test (`DefaultFactSheet`) is not used for the check. It holds
facts about Tribelt for judging whether an answer is right about it, and an administrator may edit
it at any time; the check needs a definition of who competes, and a change to it must re-check
every waiting company, which only a code change with a new check version does reliably.

**Haiku judges from evidence, not from the name.** For each company it reads the names the answers
gave, its website or "No website.", up to three snippets of the answers around its name, and its
home page: the title, the meta description, the first headings and the start of its text, clipped
short. The home page is fetched once per check, by a small adapter behind the `HomePages` port,
with the crawler's User-Agent, a ten-second timeout, a 1 MB limit, and no connection to an address
that is not public. It does not read robots.txt: one page per company is a link preview, not a
crawl. A page that cannot be read is said to be unreadable, with why.

**A sure verdict decides.** A sure competitor with a website joins the competitor list as an added
company, and a sure non-competitor is dismissed, both recorded as decided by `language model`, as
the term review does (ADR-0005). An unsure verdict, a competitor without a website, and a company
the list refuses wait for a person, with Haiku's verdict as the suggestion. A person can undo any of
Haiku's decisions; the company then waits for a person and Haiku does not decide it again, since
it is checked once per check version. A company a person decided is never checked or decided by
Haiku, and one a person decides while Haiku is checking is left as the person decided it.

**The check has a version.** Each stored check records the version that made it
(`prompt_candidate_checks.check_version`). The checks made under ADR-0006 are version 1; this is
version 2. A waiting company whose check is older is shown as not checked and is checked again on
the next review. Raising `CheckVersion` with a change to the market definition or the evidence
re-checks every waiting company, never a decided one.

## Consequences

- The review still runs only when an administrator asks for it. Reviewing a full queue of about
  500 companies fetches about 500 home pages, six at a time, and may take several minutes; what it
  checked before its 30-minute limit is kept, and the next review goes on from there.
- What each request to Haiku used, merges included, is kept in `prompt_review_usage` and counts
  towards the month's tokens on the language-model settings page, which ADR-0006 left for later.
- A wrong decision by Haiku adds a company to the competitor list, or dismisses one, until someone
  undoes it. The Competitor review page names Claude Haiku and its reason on each of its decisions,
  and filters on them, so they can be checked.
- The companies already on the competitor list are not checked yet. Checking them the same way,
  and letting an administrator archive the ones Haiku says do not compete, is a follow-up.
