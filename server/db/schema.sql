-- chatgamelab PostgreSQL schema
-- Generated from db/design conceptual model

-- NOTE: we avoid using the reserved word "user" as a table name.
-- We use app_user for the User entity.

-- User
-- Application user account (backed by Auth0). Soft-deletable via deleted_at.
CREATE TABLE app_user (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    name            text NOT NULL UNIQUE,
    email           text UNIQUE,
    deleted_at      timestamptz NULL,
    auth0_id        text UNIQUE,
    -- Default API key share to use when creating sessions without specifying one.
    -- References api_key_share instead of api_key to ensure the user has access to the key.
    default_api_key_share_id uuid NULL
);

-- Institution
-- Organization that can run workshops and own games.
CREATE TABLE institution (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    name            text NOT NULL UNIQUE
);

-- UserRole
-- Role assignment for a user, optionally scoped to an institution.
-- admin: god-mode website owner, head: institution owner, staff: institution staff.
CREATE TABLE user_role (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    user_id         uuid NOT NULL REFERENCES app_user(id),
    role            text NOT NULL,
    institution_id  uuid NULL REFERENCES institution(id),

    CONSTRAINT user_role_role_chk CHECK (role IN ('admin', 'head', 'staff')),
    CONSTRAINT user_role_user_institution_uniq UNIQUE (user_id, role, institution_id)
);

-- Workshop
-- A workshop belongs to an institution; the owner is defined by created_by.
-- If not active, the workshop cannot be joined by participants.
-- If public, it can be discovered by visitors, but they only see games marked public.
CREATE TABLE workshop (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    name            text NOT NULL,
    institution_id  uuid NOT NULL REFERENCES institution(id),
    active          boolean NOT NULL DEFAULT true,
    public          boolean NOT NULL DEFAULT false,

    CONSTRAINT workshop_name_institution_uniq UNIQUE (name, institution_id)
);

-- WorkshopParticipant
-- Anonymous guest user participating in a workshop.
CREATE TABLE workshop_participant (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    workshop_id     uuid NOT NULL REFERENCES workshop(id),
    name            text NOT NULL,
    access_token    text NOT NULL,
    active          boolean NOT NULL DEFAULT true,

    CONSTRAINT workshop_participant_workshop_token_uniq UNIQUE (workshop_id, access_token),
    CONSTRAINT workshop_participant_workshop_name_uniq UNIQUE (workshop_id, name)
);

-- ApiKey
-- An API key for an LLM provider (e.g. OpenAI, Anthropic) owned by a user.
CREATE TABLE api_key (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    user_id         uuid NOT NULL REFERENCES app_user(id),
    name            text NOT NULL DEFAULT '',
    platform        text NOT NULL, -- e.g. 'openai', 'anthropic', ..
    key             text NOT NULL
);

-- ApiKeyShare
-- A unified share table for API keys. An API key can be shared with:
-- - A user (user_id set)
-- - A workshop (workshop_id set)
-- - An institution (institution_id set)
-- At least one target must be set.
CREATE TABLE api_key_share (
    id                              uuid PRIMARY KEY,
    created_by                      uuid NULL,
    created_at                      timestamptz NOT NULL DEFAULT now(),
    modified_by                     uuid NULL,
    modified_at                     timestamptz NOT NULL DEFAULT now(),

    api_key_id                      uuid NOT NULL REFERENCES api_key(id),
    user_id                         uuid NULL REFERENCES app_user(id),
    workshop_id                     uuid NULL REFERENCES workshop(id),
    institution_id                  uuid NULL REFERENCES institution(id),
    allow_public_sponsored_plays    boolean NOT NULL DEFAULT false,

    CONSTRAINT api_key_share_target_chk CHECK (
        user_id IS NOT NULL OR workshop_id IS NOT NULL OR institution_id IS NOT NULL
    )
);

-- Game
-- Description and configuration of a game.
CREATE TABLE game (
    id                              uuid PRIMARY KEY,
    created_by                      uuid NULL,
    created_at                      timestamptz NOT NULL DEFAULT now(),
    modified_by                     uuid NULL,
    modified_at                     timestamptz NOT NULL DEFAULT now(),

    name                            text NOT NULL UNIQUE,
    description                     text NOT NULL,
    icon                            bytea NULL,

    -- Access rights and payments. public = true: discoverable on the website and playable by anyone.
    public                          boolean NOT NULL DEFAULT false,
    -- If public, a sponsored API key can be provided to pay for any public plays.
    public_sponsored_api_key_id     uuid NULL REFERENCES api_key(id),
    -- Private share links contain secret random tokens to limit access to the game.
    -- They are sponsored, so invited players don't require their own API key.
    private_share_hash              text NULL,
    private_sponsored_api_key_id    uuid NULL REFERENCES api_key(id),

    -- Game details and system messages for the LLM.
    -- What is the game about? How does it work? Player role? World description?
    system_message_scenario         text NOT NULL,
    -- How should the game start? First scene? How is the player welcomed?
    system_message_game_start       text NOT NULL,
    -- What style should the images have?
    image_style                     text NOT NULL,
    -- Additional CSS for the game, probably generated by the LLM.
    -- Should be validated/parsed strictly to avoid arbitrary code execution.
    css                             text NOT NULL,
    -- The status fields available to the LLM, shaping the JSON format for status.
    status_fields                   text NOT NULL,

    -- Quick start: pre-generated first scene of the game.
    -- This is generated content (first output after the system message) and may be
    -- regenerated from time to time to avoid being too static.
    first_message                   text NULL,
    first_status                    text NULL,
    first_image                     bytea NULL
);

-- GameTag
-- Anybody who is allowed to edit a game can also set arbitrary tags for that game.
CREATE TABLE game_tag (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    game_id         uuid NOT NULL REFERENCES game(id),
    tag             text NOT NULL,

    CONSTRAINT game_tag_game_tag_uniq UNIQUE (game_id, tag)
);

-- GameSession
-- A session is created when a user plays a game -> it's the instance of a game.
CREATE TABLE game_session (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    game_id         uuid NOT NULL REFERENCES game(id),
    user_id         uuid NOT NULL REFERENCES app_user(id),
    -- API key used to pay for this session (sponsored or user-owned), implicitly defines platform.
    api_key_id      uuid NOT NULL REFERENCES api_key(id),
    -- AI platform used for playing (e.g. 'openai', 'anthropic').
    ai_platform     text NOT NULL,
    -- AI model used for playing (e.g. 'gpt-4o-mini').
    ai_model        text NOT NULL,
    -- JSON with arbitrary details to be used within that model and within that session.
    ai_session      jsonb NOT NULL,
    image_style     text NOT NULL,
    -- Defines the status fields available in the game; copied from game.status_fields at launch.
    status_fields   text NOT NULL
);

-- GameSessionMessage
-- Messages of a game session: system message, player actions, and game responses.
CREATE TABLE game_session_message (
    id                  uuid PRIMARY KEY,
    created_by          uuid NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    modified_by         uuid NULL,
    modified_at         timestamptz NOT NULL DEFAULT now(),

    game_session_id     uuid NOT NULL REFERENCES game_session(id),
    -- Sequence number within the session, starting at 1
    seq                 integer NOT NULL,
    -- player: user message; game: LLM/game response; system: initial system/context messages.
    type                text NOT NULL,
    -- Plain text of the scene (system message, player action, or game response).
    message             text NOT NULL,
    -- JSON encoded status fields.
    status              text NULL,
    image_prompt        text NULL,
    image               bytea NULL,

    CONSTRAINT game_session_message_type_chk CHECK (type IN ('player', 'game', 'system'))
);
