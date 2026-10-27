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

-- name: TokensSince :one
SELECT coalesce(sum(llm_input_tokens + llm_output_tokens), 0)::bigint
FROM analyses
WHERE created_at >= $1;

-- name: TokensPerMatchedScan :one
-- How many scans the language model answered for, and the tokens one of them used on average.
SELECT count(*)::bigint AS scans,
    coalesce(round(avg(llm_input_tokens + llm_output_tokens)), 0)::bigint AS average
FROM analyses
WHERE matching = 'done' AND llm_input_tokens + llm_output_tokens > 0;

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

-- name: KnownTermSightings :many
-- The map terms each company's newest finished scan found, once per company.
WITH latest AS (
    SELECT DISTINCT ON (competitor_id) id, competitor_id
    FROM analyses
    WHERE competitor_id IS NOT NULL AND status = 'done'
    ORDER BY competitor_id, created_at DESC, id
)
SELECT DISTINCT latest.competitor_id::text AS competitor_id, k.term, k.language
FROM analysis_known_terms k
JOIN latest ON latest.id = k.analysis_id
ORDER BY competitor_id, k.term, k.language;

-- name: LatestMatchedProducts :many
-- The products the language model matched to Tribelt's families on each company's newest finished
-- scan whose matching ran.
WITH latest AS (
    SELECT DISTINCT ON (competitor_id) id, competitor_id
    FROM analyses
    WHERE competitor_id IS NOT NULL AND status = 'done' AND matching = 'done'
    ORDER BY competitor_id, created_at DESC, id
)
SELECT latest.competitor_id::text AS competitor_id, e.product, e.family
FROM analysis_equivalents e
JOIN latest ON latest.id = e.analysis_id
ORDER BY competitor_id, e.position;

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

-- name: AddAnalysisReadiness :exec
INSERT INTO analysis_readiness (
    analysis_id, robots, sitemap_declared, sitemap_files, sitemap_addresses, sitemap_languages, llms_present,
    llms_bytes, llms_links, llms_full_present, markdown, home_read, structured_data, languages, cms, seo_plugin,
    bot_protection, checked_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: DeleteAnalysisReadiness :exec
-- Drops an analysis's readiness, its crawler rules and raw files, so a new check can replace it.
DELETE FROM analysis_readiness
WHERE analysis_id = $1;

-- name: AddAnalysisBotRule :exec
INSERT INTO analysis_bot_rules (analysis_id, position, bot, access, named)
VALUES ($1, $2, $3, $4, $5);

-- name: AddAnalysisReadinessFiles :exec
INSERT INTO analysis_readiness_files (
    analysis_id, robots_txt, robots_truncated, llms_txt, llms_truncated, llms_full_txt, llms_full_truncated
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetAnalysisReadiness :many
-- An analysis's readiness, or no row when it has none.
SELECT analysis_id, robots, sitemap_declared, sitemap_files, sitemap_addresses, sitemap_languages, llms_present,
    llms_bytes, llms_links, llms_full_present, markdown, home_read, structured_data, languages, cms, seo_plugin,
    bot_protection, checked_at
FROM analysis_readiness
WHERE analysis_id = $1;

-- name: ListAnalysisBotRules :many
-- The crawler rules of the analyses named, in each one's order.
SELECT analysis_id, bot, access, named
FROM analysis_bot_rules
WHERE analysis_id = ANY(sqlc.arg(ids)::uuid [])
ORDER BY analysis_id, position;

-- name: GetAnalysisReadinessFiles :one
SELECT robots_txt, robots_truncated, llms_txt, llms_truncated, llms_full_txt, llms_full_truncated
FROM analysis_readiness_files
WHERE analysis_id = $1;

-- name: LatestReadinessPerCompetitor :many
-- Each company's newest scan that recorded its AI readiness, with that readiness.
SELECT DISTINCT ON (a.competitor_id) a.competitor_id::text AS competitor_id, a.id, a.url, a.max_pages, a.status,
    a.failure, a.pages_crawled, a.matching, a.created_by, a.created_at, a.finished_at, a.matching_model,
    coalesce(a.scan_number, 0)::bigint AS scan_number,
    r.robots, r.sitemap_declared, r.sitemap_files, r.sitemap_addresses, r.sitemap_languages, r.llms_present,
    r.llms_bytes, r.llms_links, r.llms_full_present, r.markdown, r.home_read, r.structured_data, r.languages, r.cms,
    r.seo_plugin, r.bot_protection, r.checked_at
FROM analyses AS a
JOIN analysis_readiness AS r ON r.analysis_id = a.id
WHERE a.competitor_id IS NOT NULL
ORDER BY a.competitor_id, a.created_at DESC, a.id;
