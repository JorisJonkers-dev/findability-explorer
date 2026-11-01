# The Recommendation run writes recommendations

The recommendations were a catalogue written by hand (ADR-0014): general advice for the whole site,
the same for every scan and run. The data the app gathers says more specific things: which search
terms competitors use that tribelt.nl does not, which sites the AI responses cite without naming
Tribelt, what the responses get wrong about it. Nobody turned that into recommendations.

**A Recommendation run writes the rest: rules find and score the gaps, and a Claude model places and
words each one** (`internal/insight`, `Gaps` and `Place` in its domain). An administrator starts it in
Manage, Recommendations; nothing starts it on its own.

- **Rules find the gaps**, from the newest Prompt run, each company's newest scan, and the search term
  list. There are five kinds: a search term competitors use and tribelt.nl does not; a belt family
  with no tribelt.nl page in a language where competitors have one; a site the web responses cite
  that names competitors but not Tribelt; a claim the fact check marked wrong (its "Wrong about
  Tribelt", never "Left out"); and an AI readiness check tribelt.nl fails that no written
  recommendation covers (crawlers kept out, no Organization markup, no language versions). A run
  writes at most the 40 best by value.
- **Rules score each gap** from 1 to 5. Benefit grows with how many competitors use a term or a
  family, a buyer close to buying (the term's buyer intent), the family's gap in AI responses, and how
  often a listing is cited or a wrong claim made. Effort is small for a term on an existing page or a
  correction, larger for a listing, larger still for a new page.
- **The model places and words each gap**, as the Claude step "Writing recommendations" (ADR-0020)
  says, ten gaps to a request, from tribelt.nl's page text as its newest scan kept it. It may move
  each score by one point, with a reason; the code enforces the bound. A page it names that the scan
  did not read, or none, leaves the page blank with "Where to put this is for the web agency to
  decide". A gap it writes nothing for keeps the rules' draft.
- **A run replaces only the open recommendations a run wrote.** One that ends done deletes those that
  are neither done nor edited and adds its own, in one transaction; a gap a kept one covers gets no
  new one. The hand-written catalogue, done ones and edited ones stay. A run stopped by hand, cut
  short by the Anthropic budget, or failed changes nothing.
- **Before it starts, the page says** how many open recommendations it replaces, what it reads, its
  estimate, and, when tribelt.nl was last scanned before the newest Prompt run, that the page text is
  older than the responses.
- **Each run is recorded** with how it ended, the model, what it added, replaced and kept, and its
  tokens (`recommendation_runs`, migration 00044), and in the work journal for Activity. Its tokens
  count towards the Anthropic budget.
- **The plan lists both**, best value first. Each recommendation says its kind and where it goes, and
  one the newest run wrote is marked New.

**Rules decide when a written recommendation is done, never a model** (`DoneByRule` in
`internal/insight/domain`), from the newest scan of tribelt.nl and the newest Prompt run, each time
the plan is read:

- **A search term** is done when that scan finds it on the page the recommendation names, or on any
  page when it names none (folded as every search compares text).
- **A page** is done when tribelt.nl has a page about the belt family in the language.
- **A listing** is done when a web response that cites the site names Tribelt or links to tribelt.nl.
- **An AI readiness check** is done when it passes: every AI crawler let in, Organization markup,
  pages in Dutch, German and English.
- **A fact sheet correction** no scan can see, so it alone is ticked off by hand, with Undo; a tick
  on one a rule decides is refused.

A run keeps what the rules find done, as it keeps what was ticked off. **An administrator can edit a
written recommendation's wording, page and scores** (`editRecommendation`); the edit records when it
was made, and an edited one is kept when the next run replaces the others. A hand-written one stays
in code and cannot be edited.
