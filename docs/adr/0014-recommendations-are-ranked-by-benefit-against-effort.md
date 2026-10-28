# Recommendations are ranked by benefit against effort

The plan listed its improvements in the order the authors wrote them. Nothing said why one came
before another, and a reader could not tell a quick change from a project of several weeks.

**Every recommendation carries two scores, each a whole number from 1 to 5: its benefit and its
effort.** Benefit is how much it helps Tribelt appear in AI answers and searches (1 is very small,
5 is very large). Effort is how much work it takes (1 is hours, 5 is several weeks). The scores
are a judgement written into the catalogue, from each recommendation's why and steps, and are
changed there, as the catalogue's version records.

**The list is ranked by benefit divided by effort, highest first.** A tie goes to the larger
benefit, then to the catalogue's order. `ByValue` in `internal/insight/domain` is the one place
this rule lives, and the API returns the recommendations already in this order.

- **A ratio, not a sum.** A quick change that helps a little should come before a project that
  helps a lot but takes weeks, and the ratio says so. Ranking by benefit alone would put the
  largest projects first, which is not where a small team should start.
- **Done recommendations drop out of the ranking**, so the next one moves up to number 1. The
  page does this; the scores do not change.
- **The page also lets a reader rank by benefit alone or by effort alone**, with the same
  tie-breaks. The API's order is the default, "best value".

## Consequences

- `PlanImprovement` in the contract gains `benefit` and `effort`, both integers from 1 to 5.
- The catalogue is version 2, as its order changed.
