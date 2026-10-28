-- +goose Up
-- The plan's target for citations was a count of a run's web answers (50 of them); it is now the
-- share of a run's web answers that cite tribelt.nl (ADR-0014). A count of 50 becomes 50%, and
-- any other count a share of the 100 web answers a run asks for, at most 100%.
UPDATE plan_targets
SET target = 'citationShare', value = LEAST(value / 100.0, 1.0)
WHERE target = 'citations';
