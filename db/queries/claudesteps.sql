-- name: ClaudeStepVersions :many
-- A Claude step's saved versions, oldest first.
SELECT step, number, model, instructions, note, created_by, created_at
FROM claude_step_versions
WHERE step = $1
ORDER BY number;

-- name: AddClaudeStepVersion :exec
INSERT INTO claude_step_versions (step, number, model, instructions, note, created_by, created_at)
VALUES (@step, @number, @model, @instructions, @note, @created_by, @created_at);
