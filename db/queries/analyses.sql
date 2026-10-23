-- name: CreateAnalysis :one
-- A company's scan takes the next number among its scans; the unique index refuses a duplicate.
INSERT INTO analyses (id, url, max_pages, status, matching, created_by, created_at, competitor_id, rescan, scan_number)
VALUES (
    sqlc.arg(id), sqlc.arg(url), sqlc.arg(max_pages), 'queued', 'pending', sqlc.arg(created_by), sqlc.arg(created_at),
    sqlc.narg(competitor_id), sqlc.arg(rescan),
    CASE WHEN sqlc.narg(competitor_id)::text IS NULL THEN NULL ELSE (
        SELECT coalesce(max(prior.scan_number), 0) + 1 FROM analyses AS prior WHERE prior.competitor_id = sqlc.narg(competitor_id)::text
    ) END
)
RETURNING coalesce(scan_number, 0)::bigint AS scan_number;

-- name: GetAnalysis :one
SELECT id, url, max_pages, status, failure, pages_crawled, matching, created_by, created_at, finished_at, competitor_id, matching_model, coalesce(scan_number, 0)::bigint AS scan_number
FROM analyses
WHERE analyses.id = $1;

-- name: ListAnalyses :many
SELECT id, url, max_pages, status, failure, pages_crawled, matching, created_by, created_at, finished_at, competitor_id, matching_model, coalesce(scan_number, 0)::bigint AS scan_number
FROM analyses
ORDER BY created_at DESC, id
LIMIT $1;

-- name: CountAnalysesCreatedSince :one
SELECT count(*)::bigint
FROM analyses
WHERE created_by = $1 AND created_at >= $2 AND NOT rescan;

-- name: TokensSince :one
SELECT coalesce(sum(llm_input_tokens + llm_output_tokens), 0)::bigint
FROM analyses
WHERE created_at >= $1;

-- name: QueuedAnalyses :many
SELECT id
FROM analyses
WHERE status = 'queued'
ORDER BY created_at, id;

-- name: FailRunningAnalyses :exec
UPDATE analyses
SET status = 'failed', failure = $1, finished_at = $2
WHERE status = 'running';

-- name: MarkAnalysisRunning :exec
UPDATE analyses
SET status = 'running'
WHERE id = $1 AND status = 'queued';

-- name: AddAnalysisPage :exec
WITH page AS (
    INSERT INTO analysis_pages (analysis_id, position, url, title, description, language, word_count, family)
    SELECT a.id, a.pages_crawled, $2, $3, $4, $5, $6, $7
    FROM analyses AS a
    WHERE a.id = $1
)
UPDATE analyses
SET pages_crawled = pages_crawled + 1
WHERE analyses.id = $1;

-- name: AddAnalysisKeyword :exec
INSERT INTO analysis_keywords (analysis_id, position, phrase, language, score, pages, tier, family, closest_term, in_map)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: AddAnalysisKnownTerm :exec
INSERT INTO analysis_known_terms (analysis_id, position, term, language, family, tier, occurrences, used_by_tribelt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: SetAnalysisMatching :exec
UPDATE analyses
SET matching = $2, llm_input_tokens = $3, llm_output_tokens = $4, matching_model = $5
WHERE analyses.id = $1;

-- name: AddAnalysisEquivalent :exec
INSERT INTO analysis_equivalents (analysis_id, position, product, family, reason)
VALUES ($1, $2, $3, $4, $5);

-- name: AddAnalysisEquivalentTerm :exec
INSERT INTO analysis_equivalent_terms (analysis_id, equivalent_position, position, term, language)
VALUES ($1, $2, $3, $4, $5);

-- name: FinishAnalysis :exec
UPDATE analyses
SET status = 'done', finished_at = $2
WHERE id = $1 AND status = 'running';

-- name: FailAnalysis :exec
UPDATE analyses
SET status = 'failed', failure = $2, finished_at = $3
WHERE id = $1 AND status IN ('queued', 'running');

-- name: ListAnalysisPages :many
SELECT url, title, description, language, word_count, coalesce(family, '')::text AS family
FROM analysis_pages
WHERE analysis_id = $1
ORDER BY position;

-- name: ListAnalysisKeywords :many
SELECT phrase, language, score, pages, coalesce(tier, 'broad')::text AS tier, coalesce(family, '')::text AS family,
    coalesce(closest_term, '')::text AS closest_term, coalesce(in_map, false)::boolean AS in_map
FROM analysis_keywords
WHERE analysis_id = $1
ORDER BY position;

-- name: ListAnalysisKnownTerms :many
SELECT term, language, family, tier, occurrences, used_by_tribelt
FROM analysis_known_terms
WHERE analysis_id = $1
ORDER BY position;

-- name: ListAnalysisEquivalents :many
SELECT position, product, family, reason
FROM analysis_equivalents
WHERE analysis_id = $1
ORDER BY position;

-- name: ListAnalysisEquivalentTerms :many
SELECT equivalent_position, term, language
FROM analysis_equivalent_terms
WHERE analysis_id = $1
ORDER BY equivalent_position, position;

-- name: AnalysisStatsByCompetitor :many
SELECT competitor_id::text AS competitor_id, count(*)::bigint AS scans
FROM analyses
WHERE competitor_id IS NOT NULL
GROUP BY competitor_id;

-- name: LatestAnalysisPerCompetitor :many
SELECT DISTINCT ON (competitor_id) competitor_id::text AS competitor_id, id, url, max_pages, status,
    failure, pages_crawled, matching, created_by, created_at, finished_at, matching_model,
    coalesce(scan_number, 0)::bigint AS scan_number
FROM analyses
WHERE competitor_id IS NOT NULL
ORDER BY competitor_id, created_at DESC, id;

-- name: ListAnalysesByCompetitor :many
SELECT id, url, max_pages, status, failure, pages_crawled, matching, created_by, created_at, finished_at, competitor_id, matching_model, coalesce(scan_number, 0)::bigint AS scan_number
FROM analyses
WHERE competitor_id = $1
ORDER BY created_at DESC, id
LIMIT $2;

-- name: CountActiveAnalysesOfCompetitor :one
SELECT count(*)::bigint
FROM analyses
WHERE competitor_id = $1 AND status IN ('queued', 'running');

-- name: DeleteAnalysesOfCompetitor :exec
DELETE FROM analyses
WHERE competitor_id = $1;

-- name: KeywordFirstScans :many
-- The number of the first finished scan of a company that found each phrase.
SELECT k.phrase, k.language, min(a.scan_number)::bigint AS first_scan
FROM analysis_keywords AS k
JOIN analyses AS a ON a.id = k.analysis_id
WHERE a.competitor_id = $1 AND a.status = 'done' AND a.scan_number IS NOT NULL
GROUP BY k.phrase, k.language;

-- name: CandidateEvidence :many
-- Every phrase a finished scan of a company found outside the terminology map.
SELECT a.id AS analysis_id, a.competitor_id::text AS competitor_id, a.url, a.created_at, k.phrase, k.language, k.score,
    k.pages, coalesce(k.tier, 'broad')::text AS tier, coalesce(k.family, '')::text AS family,
    coalesce(k.closest_term, '')::text AS closest_term
FROM analysis_keywords AS k
JOIN analyses AS a ON a.id = k.analysis_id
WHERE a.status = 'done' AND a.competitor_id IS NOT NULL AND NOT coalesce(k.in_map, false)
ORDER BY a.created_at, a.id, k.position;
