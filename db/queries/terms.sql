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
INSERT INTO term_review_usage (input_tokens, output_tokens, used_at)
VALUES ($1, $2, $3);

-- name: TermReviewTokensSince :one
SELECT coalesce(sum(input_tokens + output_tokens), 0)::bigint AS tokens
FROM term_review_usage
WHERE used_at >= $1;
