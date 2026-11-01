-- name: ListTermDecisions :many
SELECT candidate_id, phrase, status, decided_by, decided_at
FROM term_decisions
ORDER BY decided_at, candidate_id;

-- name: ListAcceptedTerms :many
SELECT t.candidate_id, t.term, t.language, t.family, t.tier, t.used_by, t.source
FROM accepted_terms AS t
JOIN term_decisions AS d ON d.candidate_id = t.candidate_id
ORDER BY d.decided_at, t.candidate_id;

-- name: InsertTermDecision :execrows
INSERT INTO term_decisions (candidate_id, phrase, status, decided_by, decided_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (candidate_id) DO NOTHING;

-- name: InsertAcceptedTerm :exec
INSERT INTO accepted_terms (candidate_id, term, language, family, tier, used_by, source)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteTermDecision :execrows
DELETE FROM term_decisions
WHERE candidate_id = $1;

-- name: ListTermVerdicts :many
SELECT candidate_id, phrase, accept, sure, reason, term, language, family, tier, model, reviewed_at
FROM term_verdicts
ORDER BY reviewed_at, candidate_id;

-- name: InsertTermVerdict :exec
INSERT INTO term_verdicts (candidate_id, phrase, accept, sure, reason, term, language, family, tier, model, reviewed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (candidate_id) DO NOTHING;

-- name: InsertTermReviewUsage :exec
INSERT INTO term_review_usage (input_tokens, output_tokens, used_at, model)
VALUES (@input_tokens, @output_tokens, @used_at, nullif(@model::text, ''));

-- name: TermReviewUsageSince :many
-- What the term review used since a moment, per model that answered; "" for rows from before
-- requests recorded their model.
SELECT coalesce(model, '')::text AS model, coalesce(sum(input_tokens), 0)::bigint AS input_tokens,
    coalesce(sum(output_tokens), 0)::bigint AS output_tokens
FROM term_review_usage
WHERE used_at >= $1
GROUP BY 1
ORDER BY 1;

-- name: TermReviewOn :one
SELECT coalesce((SELECT review_on FROM term_review_switch WHERE singleton), true)::boolean AS review_on;

-- name: SwitchTermReview :exec
INSERT INTO term_review_switch (singleton, review_on, updated_by, updated_at)
VALUES (true, @review_on, @updated_by, @updated_at)
ON CONFLICT (singleton) DO UPDATE
SET review_on = excluded.review_on, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: ListHandTerms :many
SELECT term, language, family, tier, added_by, added_at
FROM hand_terms
ORDER BY added_at, term, language;

-- name: InsertHandTerm :execrows
INSERT INTO hand_terms (term, language, family, tier, added_by, added_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (term, language) DO NOTHING;

-- name: ListRemovedTerms :many
SELECT term, language, removed_by, removed_at
FROM removed_terms
ORDER BY removed_at, term, language;

-- name: InsertRemovedTerm :exec
INSERT INTO removed_terms (term, language, removed_by, removed_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (term, language) DO NOTHING;

-- name: DeleteRemovedTerm :execrows
DELETE FROM removed_terms
WHERE term = $1 AND language = $2;
