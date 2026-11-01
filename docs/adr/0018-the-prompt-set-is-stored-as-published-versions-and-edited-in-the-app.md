# The Prompt set is stored as published versions and edited in the app

ADR-0004 fixed the Prompt set in code: a change to a question was a new version, made by a release.
Version 1 asked 90 questions and version 2, from 27 October 2026, asks 20. Adding a language, a
region, or a better question meant a developer and a deploy, and the code held only the newest set,
so the history could not show what an earlier run asked.

**The Prompt set is data: published versions with their questions, which an administrator edits as
a draft and publishes as the next version** (the spec's decisions 9 and 29). This supersedes the
fixed set of ADR-0004; its models and reading stay as they are.

- **Storage.** `prompt_sets` holds each version with when, by whom, and a note; `prompt_set_questions`
  its questions in order (belt family, buyer intent, scope, text, and an English translation when one
  is known); `prompt_scopes` every scope, a language with or without a region (migration
  `00031_prompt_sets.sql`). A published version never changes. A run asks the newest, and each
  response keeps its question's id, text, scope and version as before.
- **Versions 1 and 2 were imported** from the repository's history (migration
  `00032_import_prompt_sets.sql`). Version 1's supplier questions became the buyer intent product,
  and the scopes kept the ids the stored responses use, so every response stays linked to its
  question. Only version 1 asks in the Netherlands and Germany scopes.
- **Editing is a draft in the browser.** Questions shows one editor grouped by belt family: each
  question's text can be changed or removed, and a question added in a scope the family lacks or in
  a new scope (a language and a region) made from the same row. Nothing changes until Publish, which
  stays in view and says what changes and what a run would cost more or less, per run and per year.
  Discard the draft resets it; any version can be restored into it. The draft is not stored on the
  server, so leaving the page drops it. The spec's section 11 has drafts in the database; that was
  not needed.
- **Publishing** (`publishPromptSet`, `domain.Publish`) checks the draft, keeps the id of every
  question it keeps, gives a new question an id no version has used, stores its new scopes, and
  writes the note ("Restores version 1. Adds 3 questions, changes 1, and removes 1. Adds the scope
  French, Lyon."). A draft that asks what the newest version asks is refused, as is a number another
  publication took first. The buyer intents a question may have are product, application and problem,
  the three kinds the results show.
- **A new scope works end to end.** The responses' checks on scope, language and kind name a form
  instead of a list (migration `00041_prompt_answers_any_scope.sql`, validated by
  `00045_validate_open_checks.sql`), and a scope's language comes from the stored scopes, not from
  code. The history counts a response in a language other than Dutch, German or English towards all
  responses only, and the response filters name a scope they do not know yet by its id.
- **The charts break at a new version.** The history lists the published versions, the lines break
  where a version changed with runs on both sides, and the charts can be filtered to one version.
