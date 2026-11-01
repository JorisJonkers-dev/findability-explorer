-- name: BeginWork :exec
INSERT INTO work_runs (id, kind, started_by, started_at)
VALUES (@id, @kind, @started_by, @started_at);

-- name: EndWork :execrows
UPDATE work_runs
SET finished_at = @finished_at, ending = @ending, done = @done, total = @total, failed = @failed,
    input_tokens = @input_tokens, output_tokens = @output_tokens
WHERE id = @id AND finished_at IS NULL;

-- name: EndAbandonedWork :execrows
-- Work the journal records runs in memory: a server that stops ends it.
UPDATE work_runs
SET finished_at = @finished_at, ending = 'failed'
WHERE finished_at IS NULL;

-- name: ListWork :many
-- The running work and the newest lim ended pieces.
(SELECT id, kind, started_at, finished_at, ending, done, total, failed, input_tokens, output_tokens
 FROM work_runs WHERE finished_at IS NULL)
UNION ALL
(SELECT id, kind, started_at, finished_at, ending, done, total, failed, input_tokens, output_tokens
 FROM work_runs WHERE finished_at IS NOT NULL ORDER BY finished_at DESC, id LIMIT @lim);

-- name: ActivityPromptRuns :many
-- The running Prompt run and the newest lim others, with how many responses each got and failed,
-- and what they cost: answers and OpenRouter's readings in micros, Claude Haiku's readings in tokens.
SELECT r.id, r.started_at, r.finished_at, r.status, (r.stop_requested_at IS NOT NULL)::boolean AS stopped_by_hand, r.tasks,
    count(a.id) FILTER (WHERE a.failure IS NULL)::bigint AS answered,
    count(a.id) FILTER (WHERE a.failure IS NOT NULL)::bigint AS failed,
    coalesce(sum(a.cost_micros), 0)::bigint AS answer_cost,
    coalesce(sum(x.cost_micros) FILTER (WHERE x.haiku_tokens = 0), 0)::bigint AS openrouter_reading_cost,
    coalesce(sum(x.input_tokens) FILTER (WHERE x.haiku_tokens > 0), 0)::bigint AS haiku_input,
    coalesce(sum(x.output_tokens) FILTER (WHERE x.haiku_tokens > 0), 0)::bigint AS haiku_output
FROM prompt_runs AS r
LEFT JOIN prompt_answers AS a ON a.run_id = r.id
LEFT JOIN prompt_extractions AS x ON x.answer_id = a.id
WHERE r.id IN (
    (SELECT id FROM prompt_runs WHERE status = 'running')
    UNION
    (SELECT id FROM prompt_runs ORDER BY started_at DESC, id LIMIT @lim)
)
GROUP BY r.id;

-- name: ActivityScans :many
-- The waiting and running scans and the newest lim others.
SELECT id, rescan, created_by, created_at, finished_at, status, (stop_requested_at IS NOT NULL)::boolean AS stop_requested,
    pages_crawled, max_pages, llm_input_tokens, llm_output_tokens
FROM analyses
WHERE id IN (
    (SELECT id FROM analyses WHERE status IN ('queued', 'running'))
    UNION
    (SELECT id FROM analyses ORDER BY created_at DESC, id LIMIT @lim)
);

-- name: ActivityMonthUsage :one
-- What each kind of work used since a moment.
SELECT
    coalesce((SELECT sum(cost_micros) FROM prompt_answers WHERE answered_at >= @since), 0)::bigint AS prompt_answers,
    coalesce((SELECT sum(cost_micros) FROM prompt_extractions WHERE extracted_at >= @since AND haiku_tokens = 0), 0)::bigint AS openrouter_readings,
    coalesce((SELECT sum(input_tokens) FROM prompt_extractions WHERE extracted_at >= @since AND haiku_tokens > 0), 0)::bigint AS haiku_reading_input,
    coalesce((SELECT sum(output_tokens) FROM prompt_extractions WHERE extracted_at >= @since AND haiku_tokens > 0), 0)::bigint AS haiku_reading_output,
    coalesce((SELECT sum(llm_input_tokens) FROM analyses WHERE created_at >= @since), 0)::bigint AS scan_input,
    coalesce((SELECT sum(llm_output_tokens) FROM analyses WHERE created_at >= @since), 0)::bigint AS scan_output,
    coalesce((SELECT sum(input_tokens) FROM term_review_usage WHERE used_at >= @since), 0)::bigint AS term_review_input,
    coalesce((SELECT sum(output_tokens) FROM term_review_usage WHERE used_at >= @since), 0)::bigint AS term_review_output,
    coalesce((SELECT sum(input_tokens) FROM prompt_review_usage WHERE used_at >= @since), 0)::bigint AS candidate_check_input,
    coalesce((SELECT sum(output_tokens) FROM prompt_review_usage WHERE used_at >= @since), 0)::bigint AS candidate_check_output;
