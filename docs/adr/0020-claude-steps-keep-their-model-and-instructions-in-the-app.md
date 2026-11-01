# Claude steps keep their model and instructions in the app

Several pieces of work ask a Claude model: matching a scan's products to the belt families, reading
Prompt responses, tidying the candidates' names, checking companies, reviewing search terms, and,
in the spec, writing recommendations. Each one's model was Claude Haiku and its instructions a
constant in code, so trying a better prompt or a stronger model took a release, and nobody could see
what a step was told.

**Each Claude step keeps its model and its instructions (the system message) in the app, as numbered
versions; the data it sends and the format of its reply stay in code** (the spec's decision 27 and
section 8). Because the data goes in a separate message and the reply is read against a strict schema
in code, editing the instructions cannot break a step.

- **Storage.** `claude_step_versions` holds every version of every step: its model, instructions,
  note, who saved it, and when (migration `00038_claude_steps.sql`). A step without a saved version
  uses the instructions it came with on Claude Haiku 5.5; the first save stores that original as
  version 1, so the history keeps it. A step's next job uses its newest version.
- **Saving checks** that the model is a Claude model the app lists, and that the instructions are not
  empty, not longer than 20,000 characters, and not the current ones on the current model. Restoring
  an older version saves it as a new one, noting which.
- **Cost.** A step's job is priced by the tool's estimate for Claude Haiku 5.5, scaled by the chosen
  model's list price for the step's typical mix of tokens read and written (`claudestep/domain.Scale`).
- **One step is built so far: matching products.** The scans' matcher takes the step's newest
  instructions as its system message, and its model while an Anthropic key works; on another key the
  model from the language-model settings answers, with the same instructions. The operations
  `listClaudeSteps`, `getClaudeStep` and `saveClaudeStepVersion` serve its page in Manage.
- **The other steps follow** with their own ticket: reading responses, tidying names, checking
  companies, and reviewing terms. Reading responses is the exception the spec names: its instructions
  stay set in code so every run is read the same way, and only its model is chosen in the app (a
  definition marked `Fixed` refuses any other text). That choice gets an ADR of its own, which amends
  ADR-0004 and ADR-0007.
