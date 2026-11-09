# Claude Haiku matches products and reads Prompt test answers on an Anthropic key

Two jobs read text in the background and do not care which model does it, as long as it follows the
format: product matching in scans, which used the model chosen on the language-model settings page
(often an OpenRouter one), and the reading of Prompt test answers, which ADR-0004 fixed to
`google/gemini-2.5-flash-lite` on OpenRouter. The owner's usage of the Claude API is subsidised, so
the same work costs less on an Anthropic key than on OpenRouter.

**While an Anthropic key works, both jobs run on Claude Haiku through Anthropic's own API.** The key
is the one ADR-0005 uses for the term review: the Vault key, else a stored Anthropic key whose
latest test did not fail, never OpenRouter. Each job asks for it when it starts, a scan's matching
or a Prompt run, so a key added or removed takes effect from the next one.

- Product matching takes Haiku before the chosen model, and before the choice to turn matching off.
  The chosen model, or no matching, applies only when no Anthropic key works.
- The Prompt test reads its answers with Haiku in the same strict format, with the same
  instructions, and tries each answer twice, as before. Its tokens are priced at Haiku's list price
  ($0.10 and $0.50 per million input and output tokens), whatever the subsidy, so the run's cost and
  the month's budget stay conservative.
- The Prompt test's questions to each provider stay on OpenRouter: asking those providers is what
  the test measures.

Without an Anthropic key, each job behaves exactly as before.

## Consequences

- The language-model settings page shows Haiku as the model in use while an Anthropic key works,
  and what takes over without one. The Prompt test settings show the model that reads the answers.
- As ADR-0004 warns, a different reader makes later runs less comparable with earlier ones. Runs
  read by Haiku and runs read on OpenRouter are not told apart in the results.
- The price estimate before any run still prices reading at the OpenRouter model's price, which is
  close to Haiku's.
