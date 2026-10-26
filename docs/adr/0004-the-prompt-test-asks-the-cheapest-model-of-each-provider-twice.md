# The Prompt test asks the cheapest model of each provider twice, and one fixed model reads the answers

The report's Prompt test asks whether an AI assistant names Tribelt when a buyer asks, and it ran
once by hand as a five-prompt pilot. The Findability Explorer now repeats it on a schedule, so its
results can show trends, the competitors assistants name instead, and what they get wrong about
Tribelt. Four choices shape every result it stores, and are hard to undo once runs have a history.

**The questions are fixed and versioned.** The prompt set is 5 belt families × a supplier, an
application and a problem question × 6 scopes (Twente, the Netherlands and Germany in their own
language; Dutch, German and English with no region) = 90 prompts, in general terms, never
Tribelt's product names, built in code (`internal/prompttest/domain/prompts.go`). Every answer
stores the prompt-set version and the prompt's own text; a change to any question is a new
version, so a change in the questions never looks like a change in visibility. Settings shows the
set read-only; an editor is a later step.

**The cheapest model of each large provider, free ones included.** A run takes, for each provider
on the list (OpenAI, Google, Anthropic, Mistral, Meta, xAI and DeepSeek by default, editable in
Settings), its cheapest text model on OpenRouter's price list by prompt plus completion price,
leaving out image, audio and embedding models, batch-only variants, variable-priced routers and
moderation models. A free model is paced within OpenRouter's limits and retried when rate limited;
when it still cannot finish, the provider's cheapest paid model answers the rest. Every answer
stores the exact model that gave it. The test so follows the market without upkeep and stays
cheap, at the price of comparing small models rather than the ones in the consumer apps: the
results measure what the providers' models know and find, not what the Gemini app or ChatGPT shows.

**Two passes.** Every prompt is asked from the model's own knowledge and again with OpenRouter's
web search, so a result tells "the model knows Tribelt" (training data) from "search finds
Tribelt" (grounded answers), the report's two routes. Web-search answers carry the addresses they
cite, which feed citation share. A run is about 90 × 7 × 2 ≈ 1,300 answers.

**Natural answers, read by one fixed extractor.** The models are asked as a buyer would ask, with
no format imposed, so the answer is what a person would see. One extractor model, fixed in code
(`domain.ExtractorModel`, `google/gemini-2.5-flash-lite`) and changed only by a release, reads
every answer with a strict JSON schema: the companies it names in order and the address given for
each, the addresses it cites, whether it names and cites Tribelt, and, against the administrator's
fact sheet, whether what it says of Tribelt is correct, partly correct or wrong, quoting any wrong
claim. Names are matched to the company list by name, alias and website host; Tribelt counts as
named or cited when a matched name or a cited host shows it, even if the extractor missed it. An
answer the extractor cannot read is retried once, then marked, never dropped. Names not on the list
become competitor candidates, reviewed like the terminology map's (ADR-0003).

## Consequences

- AI share of voice is the share of read answers that name Tribelt, citation share the share that
  cite a Tribelt address; both are stored as counts per provider, scope, belt family and mode, so
  any grouping is a sum.
- Changing the extractor model, the matching or the prompt set makes later runs incomparable with
  earlier ones; the prompt-set version is recorded for that reason, and the extractor's change
  belongs in a release note.
- Runs have their own monthly budget in dollars ($10 by default), counted from the cost OpenRouter
  reports per call, apart from scan matching's token budget. A run that reaches it stops, keeping
  its answers.
- One run runs at a time, across app instances: a partial unique index refuses a second running
  run, a due scheduled run is claimed in one transaction, and the instance running it keeps a
  heartbeat, so another instance resumes a run whose holder went quiet (a restart) with the
  prompts not yet answered.
- Nothing calls OpenRouter in CI: the run, the fallback, the budget, the lock and the resume are
  tested at the service seam with fakes, the OpenRouter adapter against a local server.
