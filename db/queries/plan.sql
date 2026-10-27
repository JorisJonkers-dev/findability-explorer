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
