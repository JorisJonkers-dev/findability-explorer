# The competitor definition is edited in the app

ADR-0009 has Claude Haiku decide each competitor candidate, and ADR-0010 each company on the list,
against Tribelt's target market: who makes what, for which uses, sold to whom. That text lived in
code, and a check recorded only a number for it (`check_version`). Sharpening who counts as a
competitor, for example after a run named makers of plastic belts, took a release.

**The competitor definition is stored as versions and edited in Manage; every check is made against
the newest and records which** (the spec's decision 19). This amends ADR-0009, whose target market
becomes the first stored version.

- **Storage.** `competitor_definitions` holds each version's text (up to 8,000 characters), who saved
  it and when, and the version it restored, if any (migration `00040_competitor_definitions.sql`). A
  saved version never changes. Version 1 was the check by name and website alone (ADR-0006), with no
  text of its own; version 2, the target market the code held, is imported as the first stored
  version.
- **Saving starts no check.** `saveCompetitorDefinition` stores a new version; `getCompetitorDefinition`
  says what checking again would take, the competitors on the list and the waiting candidates
  checked against an older version and what checking them again should cost on the Anthropic key.
  The next check of the candidates or of the list checks those again. Restoring an older version
  (`restoreCompetitorDefinition`) saves its text as a new version and records which.
- A text that is the one in use is refused, so every version differs from the one before.
