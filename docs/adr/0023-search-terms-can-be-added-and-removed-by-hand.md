# Search terms can be added and removed by hand

ADR-0003 grows the search term list (the terminology map) only through a review queue: scans find
candidate terms, and an administrator or the term review (ADR-0005) accepts or rejects each. A term
nobody's site uses yet, such as one Tribelt wants to rank for, could not join the list, and a term
on the list that turned out wrong could not leave it.

**Administrators add search terms to the list by hand, and take any term off it** (the spec's
decision 20). This amends ADR-0003: the review queue stays the way scans' terms join, and is no
longer the only way.

- **Adding** (`addTerm`) checks the term against every term the list has, removed ones included,
  takes Dutch, German or English, a belt family and a buyer intent, and stores it in `hand_terms`
  with who added it and when (migration `00039_hand_terms.sql`). It is marked "Added by hand" on
  Every term and on its page. Search and the figures count it at once, and scans look for it from
  the next scan on.
- **Removing** (`removeTerm`) takes a term off the list whatever its origin, stored in
  `removed_terms`: it is no longer counted or looked for. `putBackTerm` puts it back. A term that was
  removed is put back rather than added again.
- `listHandTerms` lists both, for Manage.
