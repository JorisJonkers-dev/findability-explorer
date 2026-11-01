# Language-model spending is budgeted in dollars

Two budgets limited what the app spends on language models, in two units. The Prompt test's runs had
a monthly budget in dollars (ADR-0004), counted from what OpenRouter reports per call. Everything a
Claude model does on the Anthropic key (the scans' product matching, the term review, the review of
the competitor candidates, and Claude Haiku's reading of Prompt responses, ADR-0005, -0006, -0007)
shared one monthly limit in tokens, `LLM_MONTHLY_TOKENS`, set in the deployment. A token limit says
nothing about money, counts a cheap model's tokens like a dear one's, and could only change with a
deploy.

**Each key has a monthly budget in dollars: the OpenRouter key's is the Prompt test's, and the
Anthropic key's covers every job a Claude model does** (the spec's decision 7). This amends the token
limit of ADR-0007 and the `LLM_MONTHLY_TOKENS` of ADR-0002 and ADR-0005, which the app no longer
reads.

- **The Anthropic budget** is stored in `llm_budget` (migration `00033_llm_budget.sql`): $15 until an
  administrator sets another, from one cent to $1,000, set with `updateLlmBudget` on Keys and
  spending, and recorded as an event (`budget_set`).
- **Spending is priced from the stored token counts** at each model's list price
  (`llm/domain.ListPrice`; Claude Haiku 5.5 at $0.10 and $0.50 per million tokens in and out), so
  it needs no billing data from Anthropic and stays the same whatever a subsidy pays.
- **Every Claude job stops once the month's spend reaches the budget**, and keeps what it did:
  matching ends for the scan, the reviews end, and reading falls back to the OpenRouter model as
  before. A Haiku reading counts towards the Anthropic key only, not also towards the OpenRouter
  budget.
- **The OpenRouter budget stays where it was**, the Prompt test's monthly budget in its settings,
  now changed on Schedule. Starts show their estimate against what it has left.
- Activity shows both budgets, spent against their limits, and this month's spending by kind of work
  and the key that paid.
