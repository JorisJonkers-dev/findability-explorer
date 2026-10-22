-- name: CreateAnalysis :exec
INSERT INTO analyses (id, url, max_pages, status, matching, created_by, created_at)
VALUES ($1, $2, $3, 'queued', 'pending', $4, $5);

-- name: GetAnalysis :one
SELECT id, url, max_pages, status, failure, pages_crawled, matching, created_by, created_at, finished_at
FROM analyses
WHERE analyses.id = $1;

-- name: ListAnalyses :many
SELECT id, url, max_pages, status, failure, pages_crawled, matching, created_by, created_at, finished_at
FROM analyses
ORDER BY created_at DESC, id
LIMIT $1;

-- name: CountAnalysesCreatedSince :one
SELECT count(*)::bigint
FROM analyses
WHERE created_by = $1 AND created_at >= $2;

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
    INSERT INTO analysis_pages (analysis_id, position, url, title, description, language, word_count)
    SELECT a.id, a.pages_crawled, $2, $3, $4, $5, $6
    FROM analyses AS a
    WHERE a.id = $1
)
UPDATE analyses
SET pages_crawled = pages_crawled + 1
WHERE analyses.id = $1;

-- name: AddAnalysisKeyword :exec
INSERT INTO analysis_keywords (analysis_id, position, phrase, language, score, pages)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: AddAnalysisKnownTerm :exec
INSERT INTO analysis_known_terms (analysis_id, position, term, language, family, tier, occurrences, used_by_tribelt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: SetAnalysisMatching :exec
UPDATE analyses
SET matching = $2, llm_input_tokens = $3, llm_output_tokens = $4
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
SELECT url, title, description, language, word_count
FROM analysis_pages
WHERE analysis_id = $1
ORDER BY position;

-- name: ListAnalysisKeywords :many
SELECT phrase, language, score, pages
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
