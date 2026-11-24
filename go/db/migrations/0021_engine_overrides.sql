-- The admin's overrides of the engine's catalogue (#589): which model a tier
-- runs a task on, and a model's entry in the model table (what it takes and
-- makes, its price, its context window). The engine checks them before they
-- are stored; the core hands them to it at start and after each change.

CREATE TABLE engine_model_assignment (
    platform    text        NOT NULL,
    tier        text        NOT NULL,
    model_type  text        NOT NULL,
    model       text        NOT NULL,
    modified_at timestamptz NOT NULL DEFAULT now(),
    modified_by uuid        NULL REFERENCES app_user(id) ON DELETE SET NULL,
    PRIMARY KEY (platform, tier, model_type)
);

-- facts is the engine's ModelFactsOverride as JSON. A change to a model the
-- code's table has applies under the app version it was set under; a new
-- version brings the code's figures back.
CREATE TABLE engine_model_facts (
    platform    text        NOT NULL,
    model       text        NOT NULL,
    facts       jsonb       NOT NULL,
    app_version text        NOT NULL,
    modified_at timestamptz NOT NULL DEFAULT now(),
    modified_by uuid        NULL REFERENCES app_user(id) ON DELETE SET NULL,
    PRIMARY KEY (platform, model)
);
