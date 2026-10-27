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

- Product matching takes Haiku before the chosen model. The chosen model applies only when no
  Anthropic key works. (Amended below: turning matching off wins over Haiku.)
- The Prompt test reads its answers with Haiku in the same strict format, with the same
  instructions, and tries each answer twice, as before. Its tokens are priced at Haiku's list price
  ($0.10 and $0.50 per million input and output tokens), whatever the subsidy, so the run's cost and
  the month's budget stay conservative.
- The Prompt test's questions to each provider stay on OpenRouter: asking those providers is what
  the test measures.

Without an Anthropic key, each job behaves exactly as before.

## Consequences

- The language-model settings page shows Haiku as the model in use while an Anthropic key works,
  and what takes over without one, with the switch to turn matching off. The Prompt test settings show the model that reads the answers.
- As ADR-0004 warns, a different reader makes later runs less comparable with earlier ones. Runs
  read by Haiku and runs read on OpenRouter are not told apart in the results.
- The price estimates price reading at the model that will read: Haiku's list price while an
  Anthropic key works and the month's tokens last, else the OpenRouter model's price (amended
  below; at first they always used the OpenRouter model's price).

## Amended: turning matching off wins

At first Haiku matched products even when an administrator had turned matching off, and the
settings page hid the switch while an Anthropic key worked. The owner decided the switch must win:

- Turned off, product matching is off for every scan, Haiku or not. Choosing a model turns it on again.
- Turned on, Haiku is still preferred while an Anthropic key works, and the chosen model or the
  default takes over without one.
- The Language models page shows the switch whenever matching is on, Haiku included.

The switch is about product matching only. The term review, the competitor review and the reading
of Prompt test answers keep using Haiku while an Anthropic key works.

## Amended: Haiku stops at the month's budget

The tokens Haiku uses reading Prompt answers count towards the month's language-model tokens on the
settings page, with those of product matching, the term review and the competitor review. Once the
month's tokens are spent, OpenRouter reads the rest of the answers, as without an Anthropic key, and
product matching stops for every scan, as it always did at the budget.

## Amended: the estimates price what will really be paid

The Prompt test's estimates (the settings before a run, and "Ask … again, about $X") price reading
at the model that will read the answers, Haiku or OpenRouter's, by the tokens a reading used on
average. An answer with web search includes the search's fee: the model's own search at its price
on OpenRouter's list, else OpenRouter's search at five results for $4 per 1,000. An answer whose
reading failed can be read again on its own, by the model that reads now, without asking the
provider again.
