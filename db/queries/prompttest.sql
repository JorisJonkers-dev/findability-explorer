-- name: GetPromptSettings :many
-- The settings, or no row before an administrator saved any.
SELECT coalesce(credential_id, '')::text AS credential_id, providers, monthly_budget_micros, cadence, next_due, fact_sheet,
    fact_sheet_version, coalesce(fact_sheet_updated_by, '')::text AS fact_sheet_updated_by, fact_sheet_updated_at
FROM prompt_settings
WHERE singleton;

-- name: SavePromptSettings :exec
INSERT INTO prompt_settings (
    singleton, credential_id, providers, monthly_budget_micros, cadence, next_due, fact_sheet, fact_sheet_version,
    fact_sheet_updated_by, fact_sheet_updated_at, updated_by, updated_at
)
VALUES (
    true, sqlc.narg(credential_id), @providers, @monthly_budget_micros, @cadence, @next_due, @fact_sheet, @fact_sheet_version,
    sqlc.narg(fact_sheet_updated_by), sqlc.narg(fact_sheet_updated_at), @updated_by, @updated_at
)
ON CONFLICT (singleton) DO UPDATE
SET credential_id = excluded.credential_id, providers = excluded.providers,
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

-- name: FinishPromptRun :exec
UPDATE prompt_runs
SET status = @status, failure = sqlc.narg(failure), finished_at = @finished_at
WHERE id = @id AND status = 'running';

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

-- name: AddPromptAnswer :exec
INSERT INTO prompt_answers (
    id, run_id, prompt_id, family, kind, scope, language, prompt_text, prompt_set_version, provider, model, mode, answer,
    citations, input_tokens, output_tokens, cost_micros, failure, answered_at
)
VALUES (
    @id, @run_id, @prompt_id, @family, @kind, @scope, @language, @prompt_text, @prompt_set_version, @provider, @model, @mode,
    @answer, @citations, @input_tokens, @output_tokens, @cost_micros, sqlc.narg(failure), @answered_at
);

-- name: AddPromptExtraction :exec
INSERT INTO prompt_extractions (
    answer_id, failed, addresses, names_tribelt, cites_tribelt, correctness, wrong_claims, input_tokens, output_tokens,
    cost_micros, extracted_at
)
VALUES (
    @answer_id, @failed, @addresses, @names_tribelt, @cites_tribelt, @correctness, @wrong_claims, @input_tokens,
    @output_tokens, @cost_micros, @extracted_at
);

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
    coalesce(x.wrong_claims, '{}')::text [] AS wrong_claims, coalesce(x.input_tokens, 0)::bigint AS extraction_input_tokens,
    coalesce(x.output_tokens, 0)::bigint AS extraction_output_tokens, coalesce(x.cost_micros, 0)::bigint AS extraction_cost_micros
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
-- What answers and extractions cost since a moment, in millionths of a dollar.
SELECT (
    coalesce((SELECT sum(cost_micros) FROM prompt_answers WHERE answered_at >= @since), 0)
    + coalesce((SELECT sum(cost_micros) FROM prompt_extractions WHERE extracted_at >= @since), 0)
)::bigint AS spent;

-- name: PromptSightings :many
-- Every name an answer gave that was not matched to a company.
SELECT c.name, c.url, a.provider, a.model, a.answered_at
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
