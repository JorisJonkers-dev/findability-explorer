# Claude Haiku archives the companies on the list that do not compete

ADR-0009 has Claude Haiku decide the competitor candidates against Tribelt's target market, and
left the companies already on the competitor list unchecked. Some of them are no competitors of
Tribelt: the desk research or a person added makers of rubber belting, plastic belts or conveyor
systems. Only a researched company could be archived. An added one could only be removed, and
removing deleted every scan of it, so a wrong addition cost its history. The owner decided to
archive them and keep their scans.

**Any company on the list can be archived.** An archived company leaves the list and the
re-scans, its scans stay, and an administrator restores it. A researched company is archived
beside the research's record, as before (`company_overrides`); an added one in its own row
(`companies.archived_by`, `archived_at`). Deleting an added company with its scans stays
possible, for a company added by mistake, and the web app asks before it does. Archive is the
default action for every company.

**Haiku checks the companies on the list as it checks a candidate.** When an administrator asks,
on the all-competitors page, Haiku on an Anthropic key checks each competitor on the list that is
not archived and that no person kept. It uses the same check as ADR-0009: the same `TargetMarket`,
the same prompt, and the same evidence, which is the company's names, its website, its home page,
and up to three snippets of the answers that named it, matched as the answers are read. A company
on the list is handed to the check in the shape of a candidate (`AsCandidate`), so the check is
one piece of code. Tribelt, the mirror and the group's companies (De Spiraal, Lesjöfors) are never
checked. The check lives in the prompt-test context and the list in the companies context; the
prompt test reads the list and archives through its `Directory` port.

- A sure "not a competitor" archives the company, by `language model`, as Haiku decides a
  candidate (ADR-0005). Its reason is kept with the check and shown.
- An unsure verdict leaves the company on the list, flagged with Haiku's reason, for a person to
  keep or archive.
- A sure competitor stays, with its reason shown.

**A person's decision stands.** Restoring an archived company, or keeping a flagged one, records
who did it (`kept_by`, `kept_at`, on the company's row or beside the research's record). Haiku
never archives a kept company, and does not check it again. The companies context reads the
company just before it archives it, so one a person kept or archived meanwhile is left as the
person decided it.

**The check has a version.** Each check of a company is kept in `prompt_company_checks` with the
`CheckVersion` that made it, the one ADR-0009 introduced. A company still on the list whose check
is older is checked again on the next run, so a change to `TargetMarket` with a new version
re-checks the list. An archived or kept company keeps the check it had.

## Consequences

- The check of the list runs only when an administrator asks, apart from the review of the
  candidates. Its requests count towards the month's tokens through `prompt_review_usage`, as the
  review's do. Checking about 50 companies fetches about 50 home pages, six at a time.
- A wrong archive by Haiku takes a competitor off the list and out of the re-scans until someone
  restores it. The all-competitors page lists what Haiku archived, with its reason, each with
  Restore, and the company's page names Haiku as who archived it.
- An archived company still counts as known: the answers that name it do not make it a candidate
  again, and its website cannot be added a second time.
- `updateCompetitor` archives and restores any company and keeps one (`kept`);
  `deleteCompetitor` still archives a researched company and deletes an added one.
