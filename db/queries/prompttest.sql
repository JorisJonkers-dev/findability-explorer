-- name: GetPromptSettings :many
-- The settings, or no row before an administrator saved any.
SELECT coalesce(credential_id, '')::text AS credential_id, providers, models, monthly_budget_micros, cadence, next_due, fact_sheet,
    fact_sheet_version, coalesce(fact_sheet_updated_by, '')::text AS fact_sheet_updated_by, fact_sheet_updated_at
FROM prompt_settings
WHERE singleton;

-- name: SavePromptSettings :exec
INSERT INTO prompt_settings (
    singleton, credential_id, providers, models, monthly_budget_micros, cadence, next_due, fact_sheet, fact_sheet_version,
    fact_sheet_updated_by, fact_sheet_updated_at, updated_by, updated_at
)
VALUES (
    true, sqlc.narg(credential_id), @providers, @models::text [], @monthly_budget_micros, @cadence, @next_due, @fact_sheet, @fact_sheet_version,
    sqlc.narg(fact_sheet_updated_by), sqlc.narg(fact_sheet_updated_at), @updated_by, @updated_at
)
ON CONFLICT (singleton) DO UPDATE
SET credential_id = excluded.credential_id, providers = excluded.providers, models = excluded.models,
    monthly_budget_micros = excluded.monthly_budget_micros, cadence = excluded.cadence, next_due = excluded.next_due,
    fact_sheet = excluded.fact_sheet, fact_sheet_version = excluded.fact_sheet_version,
    fact_sheet_updated_by = excluded.fact_sheet_updated_by, fact_sheet_updated_at = excluded.fact_sheet_updated_at,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: LockPromptSchedule :many
-- The schedule's next due time, locked until the transaction ends; no row before any was saved.
SELECT next_due
FROM prompt_settings
WHERE singleton
FOR UPDATE;

-- name: MovePromptSchedule :exec
UPDATE prompt_settings
SET next_due = @next_due
WHERE singleton;

-- name: CreatePromptRun :execrows
-- A running run, unless one is running already: the partial unique index refuses it.
INSERT INTO prompt_runs (
    id, prompt_set_version, fact_sheet_version, fact_sheet, trigger, started_by, started_at, status, tasks, owner, heartbeat_at
)
VALUES (@id, @prompt_set_version, @fact_sheet_version, @fact_sheet, @trigger, @started_by, @started_at, 'running', @tasks, @owner, @heartbeat_at)
ON CONFLICT (status) WHERE status = 'running' DO NOTHING;

-- name: AddPromptRunModel :exec
INSERT INTO prompt_run_models (run_id, position, provider, model, free, fallback)
VALUES (@run_id, @position, @provider, @model, @free, @fallback);

-- name: AdoptStalePromptRun :many
-- Takes over the running run whose holder was last heard of before stale_before.
UPDATE prompt_runs
SET owner = @owner, heartbeat_at = @now
WHERE status = 'running' AND heartbeat_at < @stale_before
RETURNING id;

-- name: HeartbeatPromptRun :execrows
UPDATE prompt_runs
SET heartbeat_at = @at
WHERE id = @id AND owner = @owner AND status = 'running';

-- name: RequestPromptRunStop :execrows
-- An administrator asks a running run to stop; the instance running it reads this at its heartbeat.
UPDATE prompt_runs
SET stop_requested_at = @at
WHERE id = @id AND status = 'running';

-- name: PromptRunStopRequested :one
SELECT (stop_requested_at IS NOT NULL)::boolean AS requested
FROM prompt_runs
WHERE id = @id;

-- name: FinishPromptRun :exec
UPDATE prompt_runs
SET status = @status, failure = sqlc.narg(failure), finished_at = @finished_at
WHERE id = @id AND status = 'running';

-- name: ReopenPromptRun :execrows
-- Runs a finished run again, held by owner, unless any run is running: the run lock.
UPDATE prompt_runs AS p
SET status = 'running', finished_at = NULL, failure = NULL, stop_requested_at = NULL, owner = @owner, heartbeat_at = @at
WHERE p.id = @id AND p.status <> 'running' AND NOT EXISTS (SELECT 1 FROM prompt_runs AS r WHERE r.status = 'running');

-- name: GetPromptRun :one
SELECT id, prompt_set_version, fact_sheet_version, fact_sheet, trigger, started_by, started_at, finished_at, status, failure, tasks
FROM prompt_runs
WHERE id = @id;

-- name: ListPromptRuns :many
SELECT id, prompt_set_version, fact_sheet_version, fact_sheet, trigger, started_by, started_at, finished_at, status, failure, tasks
FROM prompt_runs
ORDER BY started_at DESC, id
LIMIT @lim;

-- name: ListPromptRunModels :many
SELECT run_id, provider, model, free, fallback
FROM prompt_run_models
WHERE run_id = ANY(sqlc.arg(ids)::uuid [])
ORDER BY run_id, position;

-- name: AddPromptAnswer :one
-- An answer, or, for a task whose earlier answer failed, that answer replaced: what the failed try
-- cost stays counted. A task answered already keeps its answer, and no row comes back.
INSERT INTO prompt_answers (
    id, run_id, prompt_id, family, kind, scope, language, prompt_text, prompt_set_version, provider, model, mode, answer,
    citations, input_tokens, output_tokens, cost_micros, failure, answered_at
)
VALUES (
    @id, @run_id, @prompt_id, @family, @kind, @scope, @language, @prompt_text, @prompt_set_version, @provider, @model, @mode,
    @answer, @citations, @input_tokens, @output_tokens, @cost_micros, sqlc.narg(failure), @answered_at
)
ON CONFLICT (run_id, prompt_id, provider, mode) DO UPDATE
SET model = excluded.model, answer = excluded.answer, citations = excluded.citations,
    input_tokens = prompt_answers.input_tokens + excluded.input_tokens,
    output_tokens = prompt_answers.output_tokens + excluded.output_tokens,
    cost_micros = prompt_answers.cost_micros + excluded.cost_micros, failure = excluded.failure,
    answered_at = excluded.answered_at
WHERE prompt_answers.failure IS NOT NULL
RETURNING id;

-- name: AddPromptExtraction :execrows
-- An extraction, or, for an answer whose reading failed, that reading replaced: what the failed
-- reading used stays counted. An answer read already keeps its reading, and no row is touched.
INSERT INTO prompt_extractions (
    answer_id, failed, addresses, names_tribelt, cites_tribelt, correctness, wrong_claims, left_out, input_tokens,
    output_tokens, cost_micros, haiku_tokens, extracted_at, model
)
VALUES (
    @answer_id, @failed, @addresses, @names_tribelt, @cites_tribelt, @correctness, @wrong_claims, @left_out, @input_tokens,
    @output_tokens, @cost_micros, @haiku_tokens, @extracted_at, @model
)
ON CONFLICT (answer_id) DO UPDATE
SET failed = excluded.failed, addresses = excluded.addresses, names_tribelt = excluded.names_tribelt,
    cites_tribelt = excluded.cites_tribelt, correctness = excluded.correctness, wrong_claims = excluded.wrong_claims,
    left_out = excluded.left_out, input_tokens = prompt_extractions.input_tokens + excluded.input_tokens,
    output_tokens = prompt_extractions.output_tokens + excluded.output_tokens,
    cost_micros = prompt_extractions.cost_micros + excluded.cost_micros,
    haiku_tokens = prompt_extractions.haiku_tokens + excluded.haiku_tokens, extracted_at = excluded.extracted_at,
    model = excluded.model
WHERE prompt_extractions.failed;

-- name: AddPromptAnswerCompany :exec
INSERT INTO prompt_answer_companies (answer_id, position, name, url, competitor_id)
VALUES (@answer_id, @position, @name, @url, sqlc.narg(competitor_id));

-- name: ListPromptAnswers :many
-- A run's answers with how they were read; the answer's text only when with_text is true.
SELECT a.id, a.prompt_id, a.family, a.kind, a.scope, a.language, a.prompt_text, a.prompt_set_version, a.provider, a.model,
    a.mode, CASE WHEN @with_text::boolean THEN a.answer ELSE '' END::text AS answer, a.citations, a.input_tokens,
    a.output_tokens, a.cost_micros, coalesce(a.failure, '')::text AS failure, a.answered_at,
    (x.answer_id IS NOT NULL)::boolean AS extracted, coalesce(x.failed, false)::boolean AS extraction_failed,
    coalesce(x.addresses, '{}')::text [] AS addresses, coalesce(x.names_tribelt, false)::boolean AS names_tribelt,
    coalesce(x.cites_tribelt, false)::boolean AS cites_tribelt, coalesce(x.correctness, '')::text AS correctness,
    coalesce(x.wrong_claims, '{}')::text [] AS wrong_claims, coalesce(x.left_out, '{}')::text [] AS left_out,
    coalesce(x.input_tokens, 0)::bigint AS extraction_input_tokens,
    coalesce(x.output_tokens, 0)::bigint AS extraction_output_tokens, coalesce(x.cost_micros, 0)::bigint AS extraction_cost_micros,
    coalesce(x.model, '')::text AS reading_model
FROM prompt_answers AS a
LEFT JOIN prompt_extractions AS x ON x.answer_id = a.id
WHERE a.run_id = @run_id
ORDER BY a.answered_at, a.id;

-- name: ListPromptAnswerCompanies :many
SELECT c.answer_id, c.position, c.name, c.url, coalesce(c.competitor_id, '')::text AS competitor_id
FROM prompt_answer_companies AS c
JOIN prompt_answers AS a ON a.id = c.answer_id
WHERE a.run_id = @run_id
ORDER BY c.answer_id, c.position;

-- name: PromptSpentSince :one
-- What the OpenRouter key paid since a moment, in millionths of a dollar: the answers, and the
-- readings OpenRouter did. Claude Haiku's readings are the Anthropic key's (PromptReadingTokensSince).
SELECT (
    coalesce((SELECT sum(cost_micros) FROM prompt_answers WHERE answered_at >= @since), 0)
    + coalesce((SELECT sum(cost_micros) FROM prompt_extractions WHERE extracted_at >= @since AND haiku_tokens = 0), 0)
)::bigint AS spent;

-- name: PromptSightings :many
-- Every name an answer gave that was not matched to a company.
SELECT c.answer_id, c.name, c.url, a.provider, a.model, a.answered_at
FROM prompt_answer_companies AS c
JOIN prompt_answers AS a ON a.id = c.answer_id
WHERE c.competitor_id IS NULL
ORDER BY a.answered_at, c.answer_id, c.position;

-- name: ListPromptCandidateDecisions :many
SELECT id, name, status, coalesce(competitor_id, '')::text AS competitor_id, decided_by, decided_at
FROM prompt_candidate_decisions
ORDER BY decided_at, id;

-- name: AddPromptCandidateDecision :exec
INSERT INTO prompt_candidate_decisions (id, name, status, competitor_id, decided_by, decided_at)
VALUES (@id, @name, @status, sqlc.narg(competitor_id), @decided_by, @decided_at);

-- name: DeletePromptCandidateDecision :execrows
DELETE FROM prompt_candidate_decisions
WHERE id = @id;

-- name: ListPromptCandidateMerges :many
SELECT candidate_id, coalesce(into_id, '')::text AS into_id, coalesce(competitor_id, '')::text AS competitor_id,
    kept_apart, merged_by, merged_at
FROM prompt_candidate_merges
ORDER BY merged_at, candidate_id;

-- name: SavePromptCandidateMerge :exec
INSERT INTO prompt_candidate_merges (candidate_id, into_id, competitor_id, kept_apart, merged_by, merged_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (candidate_id) DO UPDATE SET
    into_id = excluded.into_id, competitor_id = excluded.competitor_id, kept_apart = excluded.kept_apart,
    merged_by = excluded.merged_by, merged_at = excluded.merged_at;

-- name: ListPromptCandidateChecks :many
SELECT candidate_id, competitor, sure, reason, model, check_version, checked_at
FROM prompt_candidate_checks
ORDER BY checked_at, candidate_id;

-- name: SavePromptCandidateCheck :exec
INSERT INTO prompt_candidate_checks (candidate_id, competitor, sure, reason, model, check_version, checked_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (candidate_id) DO UPDATE SET
    competitor = excluded.competitor, sure = excluded.sure, reason = excluded.reason,
    model = excluded.model, check_version = excluded.check_version, checked_at = excluded.checked_at;

-- name: ListPromptAnswerTexts :many
-- The text of some answers, to quote what they say around a company's name.
SELECT id, answer
FROM prompt_answers
WHERE id = ANY(sqlc.arg(ids)::uuid []);

-- name: InsertPromptReviewUsage :exec
INSERT INTO prompt_review_usage (input_tokens, output_tokens, used_at)
VALUES (@input_tokens, @output_tokens, @used_at);

-- name: PromptReviewTokensSince :one
SELECT coalesce(sum(input_tokens), 0)::bigint AS input, coalesce(sum(output_tokens), 0)::bigint AS output
FROM prompt_review_usage
WHERE used_at >= @since;

-- name: ListPromptCompanyChecks :many
SELECT company_id, competitor, sure, reason, model, check_version, checked_at
FROM prompt_company_checks
ORDER BY checked_at, company_id;

-- name: SavePromptCompanyCheck :exec
INSERT INTO prompt_company_checks (company_id, competitor, sure, reason, model, check_version, checked_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (company_id) DO UPDATE SET
    competitor = excluded.competitor, sure = excluded.sure, reason = excluded.reason,
    model = excluded.model, check_version = excluded.check_version, checked_at = excluded.checked_at;

-- name: PromptNamings :many
-- Every name an answer gave, matched or not, the newest answer first: what the answers say of a
-- company on the list.
SELECT c.answer_id, c.name, c.url, coalesce(c.competitor_id, '')::text AS competitor_id
FROM prompt_answer_companies AS c
JOIN prompt_answers AS a ON a.id = c.answer_id
ORDER BY a.answered_at DESC, c.answer_id, c.position;

-- name: PromptReadingTokensSince :one
-- The tokens in and out Claude Haiku on an Anthropic key used reading answers since a moment: the
-- readings whose tokens count as Haiku's.
SELECT coalesce(sum(input_tokens), 0)::bigint AS input, coalesce(sum(output_tokens), 0)::bigint AS output
FROM prompt_extractions
WHERE extracted_at >= @since AND haiku_tokens > 0;

-- name: PromptReadingTokensSinceByModel :many
-- The same per model that read, so each is priced at its own list price; a reading from before
-- readings recorded their model has none.
SELECT model, coalesce(sum(input_tokens), 0)::bigint AS input, coalesce(sum(output_tokens), 0)::bigint AS output
FROM prompt_extractions
WHERE extracted_at >= @since AND haiku_tokens > 0
GROUP BY model
ORDER BY model;

-- name: ListPromptSets :many
-- Every published Prompt set, newest first.
SELECT version, published_at, published_by, note
FROM prompt_sets
ORDER BY version DESC;

-- name: ListPromptSetQuestions :many
-- Every question of every published set with its scope, the newest set first, each set in order.
SELECT q.version, q.prompt_id, q.family, q.intent, q.text, coalesce(q.english, '')::text AS english,
    s.id AS scope_id, s.language, coalesce(s.region, '')::text AS region, s.name AS scope_name
FROM prompt_set_questions q
JOIN prompt_scopes s ON s.id = q.scope_id
ORDER BY q.version DESC, q.position;

-- name: ListPromptScopes :many
-- Every scope a question can be asked in, by name.
SELECT id, language, coalesce(region, '')::text AS region, name
FROM prompt_scopes
ORDER BY name;

-- name: InsertPromptScope :exec
-- A scope a newly published version asks in.
INSERT INTO prompt_scopes (id, language, region, name)
VALUES (@id, @language, nullif(@region::text, ''), @name);

-- name: InsertPromptSet :execrows
-- A newly published version; no row when another publication took the number first.
INSERT INTO prompt_sets (version, published_at, published_by, note)
VALUES (@version, @published_at, @published_by, @note)
ON CONFLICT (version) DO NOTHING;

-- name: InsertPromptSetQuestion :exec
-- A question of a newly published version, at its position.
INSERT INTO prompt_set_questions (version, position, prompt_id, family, intent, scope_id, text, english)
VALUES (@version, @position, @prompt_id, @family, @intent, @scope_id, @text, nullif(@english::text, ''));

-- name: ListCompetitorDefinitions :many
-- Every version of the competitor definition, the newest first.
SELECT version, text, saved_at, saved_by, coalesce(restored_from, 0)::bigint AS restored_from
FROM competitor_definitions
ORDER BY version DESC;

-- name: AddCompetitorDefinition :one
-- The next version of the competitor definition.
INSERT INTO competitor_definitions (version, text, saved_at, saved_by, restored_from)
SELECT coalesce(max(version), 0) + 1, @text, @saved_at, @saved_by, sqlc.narg(restored_from)
FROM competitor_definitions
RETURNING version;

-- name: PromptCheckUsage :one
-- What every request of the reviews and checks used, and how many companies they checked.
SELECT
    coalesce((SELECT sum(input_tokens) FROM prompt_review_usage), 0)::bigint AS input,
    coalesce((SELECT sum(output_tokens) FROM prompt_review_usage), 0)::bigint AS output,
    ((SELECT count(*) FROM prompt_candidate_checks) + (SELECT count(*) FROM prompt_company_checks))::bigint AS checked;

-- name: PromptRunReadingModels :many
-- The model that read most of each run's responses, for the runs whose readings recorded one.
SELECT DISTINCT ON (a.run_id) a.run_id, x.model
FROM prompt_answers AS a
JOIN prompt_extractions AS x ON x.answer_id = a.id
WHERE x.model <> '' AND NOT x.failed
GROUP BY a.run_id, x.model
ORDER BY a.run_id, count(*) DESC, x.model;
