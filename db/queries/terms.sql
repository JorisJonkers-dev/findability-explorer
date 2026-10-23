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
