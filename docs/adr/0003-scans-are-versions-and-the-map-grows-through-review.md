# Scans are versions of a company, and the terminology map grows through a review queue

The operator wants each competitor's search terms for Tribelt's belt families listed in depth,
kept per competitor, and re-scanned on demand with a history; and the terms scans find that the
desk research missed should reach the terminology map. Three choices follow.

**A scan is a version.** Every analysis of a company's website is kept, never overwritten, and
numbered per company (`analyses.scan_number`: 1 for the first, assigned when the scan is queued,
unique per company, never renumbered). A company's profile is its latest finished scan, grouped by
Tribelt family (eyelink, open spiral, closed spiral, Tri-Flex, other) with broad and specification
terms apart; for each term the profile names the first finished scan that found it. Any two
finished scans compare as terms added, removed and moved (score changed by at least a quarter and
5 points, or pages by 2). Re-scans run only when an administrator asks (one company, a list, or
every company with data); there is no schedule, so the crawler's load on a site stays a person's
decision. A company never has two scans queued or running at once, and three scans of different
companies run side by side.

**Only administrators change what is stored.** Starting a scan or re-scan, adding, editing,
archiving or removing a company, and reviewing terms need auth-api's `ROLE_ADMIN`, with the roles
re-read for the request, as the language-model settings do (ADR-0001). Everyone signed in reads
the results and the history. An added company is deleted with its scans; a researched company is
the desk research's record, so an administrator's change to it (name, website, archived) is stored
beside it in `company_overrides`, and removing one archives it instead of deleting anything.

**The map grows through a review queue, not by itself.** (ADR-0005 lets a language model decide the candidates it is sure about.) A phrase a finished scan found that was
not one of the map's spellings is a candidate, keyed by its folded phrase and aggregated across
companies with its evidence (companies, scans, languages, best score) and the classifier's
suggested family and tier. An administrator accepts it, writing the term, language, family, tier
and companies as the map should show them, or rejects it; either decision records who and when and
can be undone. A rejected phrase stays rejected, even when later scans find it again. Accepted
terms are stored in Postgres and join the embedded seed's terms in search, facets and the next
scans' classification; they carry `origin: review`.

## Consequences

- The seed stays the reproducible record of the desk research of 22 October 2026; the map a user
  sees is that record plus reviewed additions, and `origin` tells them apart.
- Phrases are not re-classified when the map grows: a stored scan keeps the judgement it was made
  with, and a scan after an acceptance marks the phrase as in the map.
- The candidate list is computed from every finished scan on each request. That is cheap at the
  course's scale (a few thousand phrases); a far larger history would need it materialised.
- Deleting an added company deletes its scans, so its numbers are gone with it; a researched
  company's scans are never deleted by the app.
