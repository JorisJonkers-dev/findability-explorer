# Competitor candidates fold by website, and Claude Haiku merges and checks them

The Prompt runs' answers name hundreds of companies the list does not hold (514 waited after the
first runs), and most are duplicates: "Taiwan Metiz Alliance", "Metiz Alliance", "Metiz" and "Wire
Mesh Conveyor Belts – Taiwan Metiz Alliance" are one company on metiz.tw. A candidate was one name
key, so each spelling was reviewed on its own, and an address on a subdomain or without a scheme
never matched the company on the list.

**A website is its registered domain.** Two addresses are one company's website when they share
the registered domain (the public-suffix list's eTLD+1), so `products.bandogrp.com` is
`bandogrp.com`'s and `www.x.co.uk` is `x.co.uk`'s. An address on a marketplace, directory or
social network (europages, made-in-china, wlw, facebook and the like) is nobody's website. This
holds for matching answers to the list, for the citations, and for the company list's own
duplicate check.

**Candidates fold into companies.** Names given with an address on one website are one company,
and so are names merged by Claude Haiku or a person. A company is listed under the name the
answers give most, with all its names, answers, providers and websites, and the home page to add
it with. One that shares a website or name with a company on the list, or that Haiku merged into
one, is that company and drops out of the review. A decision on a company is recorded on each of
its names, so a spelling that joins later is covered too, and undoing it undoes them all.

**Answers are matched again when they are read.** An answer's companies were matched once, when
it was read, so a company added later never counted in older answers. Now each name an answer gave
that was not matched is matched again against the list as it is, and against the names added or
merged into a company on it. The stored answer does not change.

**Claude Haiku merges and checks; a person decides.** Haiku on an Anthropic key (as in ADR-0005,
never OpenRouter, which costs more) reviews when an administrator asks. It first groups the waiting
companies that are one company, or a company on the list; then it checks each company it has not
checked, forty at a time, for whether it is a competitor of Tribelt, whether it is sure, and why.
Its merges take effect, and a person can split a merged name off again; it is then never merged
again. Its checks are suggestions only: the page filters on them, and a person adds or dismisses,
one at a time, by keyboard, or many at once.

## Consequences

- Merges are kept in `prompt_candidate_merges`, checks in `prompt_candidate_checks`; decisions stay
  in `prompt_candidate_decisions`, one row per name.
- Names on one website cannot be split apart. A company that really shares a website with another
  has to be told apart by hand when it is added.
- The review runs only on request, so it costs nothing until someone asks. Its tokens are not yet
  counted towards the month on the language-model settings page.
