# The Prompt test asks the Model list

ADR-0004 had every Prompt run ask the cheapest text model of each provider in the settings, chosen
again from OpenRouter's price list when the run started. An administrator chose providers, not
models: the model behind a provider's column could change between runs when a cheaper one appeared,
two models of one provider could not both be asked, and a model a buyer actually uses could not be
picked over a cheaper one.

**Prompt runs ask a stored Model list: OpenRouter models an administrator picks from OpenRouter's
catalogue** (the spec's decision 10). This supersedes the cheapest model per provider of ADR-0004;
the reading of the responses stays as ADR-0004 and ADR-0020 have it, and the Claude steps keep their
own models (ADR-0020), which are not on this list.

- **Storage.** `prompt_settings.models` holds the list's model ids, in order (migration
  `00042_prompt_model_list.sql`); the service keeps it to 1 to 20 ids (`domain.CheckModels`). It
  is empty until an administrator saves a list, and until then runs ask the cheapest model of each
  provider in `providers`, as before, so nothing changed when this was deployed.
- **The catalogue** (`getPromptModelCatalogue`, `domain.Catalogue`) lists every text model on
  OpenRouter: one that reads text and writes only text, with a fixed price. Image and audio models,
  `:batch` variants, moderation models (`-guard`), routers such as `openrouter/auto`, and aliases
  such as `~anthropic/claude-haiku-latest` are left out. It is read with the Prompt test's key at
  most once a day and kept in memory; a read that fails keeps the last list. Each model says when
  OpenRouter listed it (its `created`).
- **The recommended five.** The cheapest text model of OpenAI, Google, Anthropic, Mistral, and
  DeepSeek, free ones included, is marked Recommended and listed first, in that order
  (`domain.Recommended`); the rest follow cheapest first.
- **Prices.** A model the latest run asked is priced by its own responses, mode by mode. Any other
  model is priced from its list price for a typical response (300 tokens in, 900 out), and with web
  search its own search fee is added, or OpenRouter's ($0.02) for a model that does not search
  itself. Save the model list says what a run would cost against the saved list.
- **Each listed model is its own column.** A run's column key (`provider` in the API) stays the
  provider's id while the list holds one model of that provider, so a run before the list and a
  run after it compare column by column. Two models of one provider are each keyed by their model
  id, and the API's provider fields take ids of up to 200 characters for it. A free model keeps its
  provider's cheapest paid model as the fallback, as before.
- **A model OpenRouter stops listing** is skipped at the next run. A list with none of its models
  left starts no run (`domain.ErrNoModels`).
- **The charts break where the list changed.** The history gives the models each run asked, and
  the trend charts break at a run that asked other models than the run before it, with a note
  naming the models added and removed.
- The list is edited in Manage, Prompt test, Models: the list with each model's cost per run, Save
  the model list, and Add a model with a search, a provider filter with counts, and Free only.
