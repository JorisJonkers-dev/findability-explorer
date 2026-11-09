-- name: ListPlanTicks :many
SELECT improvement_id, catalogue_version, ticked_by, ticked_at
FROM plan_ticks
ORDER BY improvement_id;

-- name: TickPlanImprovement :exec
INSERT INTO plan_ticks (improvement_id, catalogue_version, ticked_by, ticked_at)
VALUES (@improvement_id, @catalogue_version, @ticked_by, @ticked_at)
ON CONFLICT (improvement_id) DO UPDATE
SET catalogue_version = excluded.catalogue_version, ticked_by = excluded.ticked_by, ticked_at = excluded.ticked_at;

-- name: UntickPlanImprovement :exec
DELETE FROM plan_ticks WHERE improvement_id = @improvement_id;

-- name: ListPlanTargets :many
SELECT target, value, set_by, set_at
FROM plan_targets
ORDER BY target;

-- name: SetPlanTarget :exec
INSERT INTO plan_targets (target, value, set_by, set_at)
VALUES (@target, @value, @set_by, @set_at)
ON CONFLICT (target) DO UPDATE
SET value = excluded.value, set_by = excluded.set_by, set_at = excluded.set_at;

-- name: ListRecommendations :many
-- Every recommendation the runs wrote that is kept, the oldest first.
SELECT id, run_id, gap_key, kind, language, family, specifics, rule_benefit, rule_effort, title, where_url, where_note, why, steps,
    done_when, benefit, effort, score_reason, created_at, coalesce(done_by, '')::text AS done_by, done_at, edited_at
FROM recommendations
ORDER BY created_at, id;

-- name: ListRecommendationRuns :many
-- The newest lim Recommendation runs, the newest first.
SELECT id, started_at, finished_at, coalesce(ending, '')::text AS ending, started_by, model, gaps, added, replaced, kept,
    input_tokens, output_tokens
FROM recommendation_runs
ORDER BY started_at DESC, id
LIMIT @lim;

-- name: BeginRecommendationRun :execrows
-- A run begun, heard of as it begins, unless another runs: one heard of since stale_before, or one
-- begun without a heartbeat (by a server before runs beat) since unbeaten_before.
INSERT INTO recommendation_runs (id, started_at, started_by, model, heartbeat_at)
SELECT @id::uuid, @started_at::timestamptz, @started_by::text, @model::text, @started_at::timestamptz
WHERE NOT EXISTS (
    SELECT 1 FROM recommendation_runs
    WHERE finished_at IS NULL AND (
        (heartbeat_at IS NOT NULL AND heartbeat_at >= @stale_before::timestamptz)
        OR (heartbeat_at IS NULL AND started_at >= @unbeaten_before::timestamptz)
    )
);

-- name: BeatRecommendationRun :execrows
UPDATE recommendation_runs
SET heartbeat_at = @at
WHERE id = @id AND finished_at IS NULL;

-- name: RecommendationRunRunning :one
-- Whether a run runs: unfinished, and heard of since stale_before (or, without a heartbeat, begun
-- since unbeaten_before).
SELECT EXISTS (
    SELECT 1 FROM recommendation_runs
    WHERE finished_at IS NULL AND (
        (heartbeat_at IS NOT NULL AND heartbeat_at >= @stale_before::timestamptz)
        OR (heartbeat_at IS NULL AND started_at >= @unbeaten_before::timestamptz)
    )
)::boolean AS running;

-- name: RequestRecommendationRunStop :execrows
-- An administrator asks the running run to stop; the instance running it reads this between batches.
UPDATE recommendation_runs
SET stop_requested_at = @at
WHERE finished_at IS NULL AND (
        (heartbeat_at IS NOT NULL AND heartbeat_at >= @stale_before::timestamptz)
        OR (heartbeat_at IS NULL AND started_at >= @unbeaten_before::timestamptz)
    );

-- name: RecommendationRunStopRequested :one
SELECT (stop_requested_at IS NOT NULL)::boolean AS requested
FROM recommendation_runs
WHERE id = @id;

-- name: EndAbandonedRecommendationRuns :execrows
-- Runs left unfinished by a server that stopped: unheard of since stale_before, or begun without a
-- heartbeat before unbeaten_before. They end as failed and change no recommendation.
UPDATE recommendation_runs
SET finished_at = @finished_at, ending = 'failed'
WHERE finished_at IS NULL AND NOT (
        (heartbeat_at IS NOT NULL AND heartbeat_at >= @stale_before::timestamptz)
        OR (heartbeat_at IS NULL AND started_at >= @unbeaten_before::timestamptz)
    );

-- name: FinishRecommendationRun :exec
UPDATE recommendation_runs
SET finished_at = @finished_at, ending = @ending, gaps = @gaps, added = @added, replaced = @replaced, kept = @kept,
    input_tokens = @input_tokens, output_tokens = @output_tokens
WHERE id = @id;

-- name: DeleteRecommendation :exec
DELETE FROM recommendations
WHERE id = @id;

-- name: AddRecommendation :exec
INSERT INTO recommendations (
    id, run_id, gap_key, kind, language, family, specifics, rule_benefit, rule_effort, title, where_url, where_note, why, steps,
    done_when, benefit, effort, score_reason, created_at
)
VALUES (
    @id, @run_id, @gap_key, @kind, @language, @family, @specifics, @rule_benefit, @rule_effort, @title, @where_url, @where_note, @why,
    @steps, @done_when, @benefit, @effort, @score_reason, @created_at
);

-- name: SetRecommendationDone :execrows
UPDATE recommendations
SET done_by = sqlc.narg(done_by), done_at = sqlc.narg(done_at)
WHERE id = @id;

-- name: RecommendationTokensSince :one
-- The tokens in and out the Recommendation runs used since a moment.
SELECT coalesce(sum(input_tokens), 0)::bigint AS input, coalesce(sum(output_tokens), 0)::bigint AS output
FROM recommendation_runs
WHERE started_at >= @since;

-- name: UpdateRecommendation :execrows
-- An administrator's wording and scores of a recommendation a run wrote.
UPDATE recommendations
SET title = @title, where_url = @where_url, where_note = @where_note, why = @why, steps = @steps, done_when = @done_when,
    benefit = @benefit, effort = @effort, edited_at = @edited_at
WHERE id = @id;
