-- v2: the admin's overrides of the engine's catalogue (#589), read and
-- written by the secure core (core/engine_overrides.go).

-- name: V2EngineModelAssignments :many
SELECT platform, tier, model_type, model, modified_at, modified_by
FROM engine_model_assignment
ORDER BY platform, tier, model_type;

-- name: V2SetEngineModelAssignment :exec
INSERT INTO engine_model_assignment (platform, tier, model_type, model, modified_by)
VALUES (@platform, @tier, @model_type, @model, @modified_by)
ON CONFLICT (platform, tier, model_type)
DO UPDATE SET model = EXCLUDED.model, modified_at = now(), modified_by = EXCLUDED.modified_by;

-- name: V2DeleteEngineModelAssignment :exec
DELETE FROM engine_model_assignment
WHERE platform = @platform AND tier = @tier AND model_type = @model_type;

-- name: V2EngineModelFacts :many
SELECT platform, model, facts, app_version, modified_at, modified_by
FROM engine_model_facts
ORDER BY platform, model;

-- name: V2SetEngineModelFacts :exec
INSERT INTO engine_model_facts (platform, model, facts, app_version, modified_by)
VALUES (@platform, @model, @facts, @app_version, @modified_by)
ON CONFLICT (platform, model)
DO UPDATE SET facts = EXCLUDED.facts, app_version = EXCLUDED.app_version,
              modified_at = now(), modified_by = EXCLUDED.modified_by;

-- name: V2DeleteEngineModelFacts :exec
DELETE FROM engine_model_facts WHERE platform = @platform AND model = @model;
