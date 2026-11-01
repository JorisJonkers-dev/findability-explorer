# The model that reads the responses is chosen in the app, and its instructions stay in the code

ADR-0004 fixed one extractor model in code, changed only by a release, so that every Prompt run is
read the same way and runs can be compared. ADR-0007 then had Claude Haiku on an Anthropic key read
the responses while one works, with OpenRouter's fixed model as the fallback. The redesign makes
every Claude step's model and instructions an administrator's to change (the spec's section 8), so
Tribelt can tune them after the handover without a release (ADR-0020, which covers the Claude
steps in general). Reading responses is the one step whose instructions decide what every figure
on the AI responses page means. This record amends ADR-0004 (one fixed extractor) and ADR-0007
(Claude Haiku reads on an Anthropic key) for that step.

**The reading step's model is chosen in the app; its instructions stay in the code.** Reading
responses is a Claude step like the others (ADR-0020), at `/manage/claude-steps/reading`, with versions,
Save the model, Undo, and a model history. Its instructions are shown read-only, marked "Set in the
code", and change only with a release (`claudestep/domain.ReadingInstructions`, which the
extractor sends). The API refuses a version with other instructions. On an Anthropic key the
newest version's model reads, where ADR-0007 named Claude Haiku; without one, OpenRouter's fixed
`domain.ExtractorModel` still reads, as ADR-0004 and ADR-0007 say.

**Each reading records the model that read it, and a change of reader breaks the charts.** A
reading stores its model (`prompt_extractions.model`); the One response page says "Checked by"
that model. The history names, per run, the model that read most of its responses, and the AI
responses charts break where it changes from one run to the next, as they do where the Prompt set
changes. Earlier runs keep how they were read; Read them again on Prompt test, Runs, reads them
with the model in use now. Readings made before this keep no model, and draw no break.

**Consequences.** Runs read by different models are no longer one measure; the chart says so
rather than joining the lines. The reading's cost, and the budget it counts against, follow the
chosen model's list price. A model change costs nothing to undo: Undo before saving, or restore
an earlier version, which saves a new one.
