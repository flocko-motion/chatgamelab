-- chatgamelab PostgreSQL schema
-- Generated from db/design conceptual model

-- NOTE: we avoid using the reserved word "user" as a table name.
-- We use app_user for the User entity.

-- User
-- Application user account (backed by Auth0 or participant token). Soft-deletable via deleted_at.
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
    -- Participant token for anonymous workshop participants (alternative auth method)
    -- Prefixed with "participant-" to distinguish from JWT tokens
    participant_token text UNIQUE,
    -- Default API key share to use when creating sessions without specifying one.
    -- References api_key_share instead of api_key to ensure the user has access to the key.
    default_api_key_share_id uuid NULL,
    -- User preference: show AI model selector when creating sessions
    show_ai_model_selector boolean NOT NULL DEFAULT false,
    -- User's preferred language (ISO 639-1 code: en, de, fr, etc.)
    language text NOT NULL DEFAULT 'en'
);

-- Institution
-- Organization that can run workshops and own games.
CREATE TABLE institution (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    name            text NOT NULL UNIQUE,
    deleted_at      timestamptz NULL,
    -- Free-use API key share for institution members (any member can use this key to play)
    free_use_api_key_share_id uuid NULL REFERENCES api_key_share(id)
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
    deleted_at      timestamptz NULL,
    -- Default API key share for workshop participants (set by staff/heads)
    default_api_key_share_id uuid NULL,
    -- Workshop settings (configured by staff/heads)
    use_specific_ai_model text NULL,  -- If set, use this AI model instead of system default
    show_ai_model_selector boolean NOT NULL DEFAULT false,  -- If true, participants can select AI model
    show_public_games boolean NOT NULL DEFAULT false,  -- If true, participants can see public games
    show_other_participants_games boolean NOT NULL DEFAULT true,  -- If true, participants can see other participants' games

    CONSTRAINT workshop_name_institution_uniq UNIQUE (name, institution_id)
);

-- UserRole
-- Role assignment for a user, optionally scoped to an institution.
-- admin: god-mode website owner, head: institution owner, staff: institution staff, participant: workshop participant.
CREATE TABLE user_role (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    user_id         uuid NOT NULL REFERENCES app_user(id),
    role            text NOT NULL,
    institution_id  uuid NULL REFERENCES institution(id),
    workshop_id     uuid NULL REFERENCES workshop(id),
    -- Active workshop for head/staff/individual when in "workshop mode"
    active_workshop_id uuid NULL REFERENCES workshop(id),

    CONSTRAINT user_role_role_chk CHECK (role IN ('admin', 'head', 'staff', 'participant', 'individual')),
    CONSTRAINT user_role_user_institution_workshop_uniq UNIQUE (user_id, role, institution_id, workshop_id)
);

-- UserRoleInvite
-- Invite a user to assume a role in an organization (institution).
-- Supports two use cases:
-- 1. Targeted invite: inviting a specific user (by user_id or email) to assume a role
-- 2. Open invite: creating an invite token that allows unspecified users to claim a role (e.g., participant)
CREATE TABLE user_role_invite (
    id              uuid PRIMARY KEY,
    created_by      uuid NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    modified_by     uuid NULL,
    modified_at     timestamptz NOT NULL DEFAULT now(),

    -- The institution this invite is for
    institution_id  uuid NOT NULL REFERENCES institution(id),
    -- The role being offered (admin, head, staff, participant)
    role            text NOT NULL,
    -- Optional workshop scope (e.g., for participant invites to a specific workshop)
    workshop_id     uuid NULL REFERENCES workshop(id),

    -- Targeted invite fields (at least one should be set for targeted invites)
    invited_user_id uuid NULL REFERENCES app_user(id),
    invited_email   text NULL,

    -- Open invite fields (for unspecified users)
    -- A secure random token that can be shared via link
    invite_token    text NULL UNIQUE,
    -- Maximum number of times this invite can be used (NULL = unlimited)
    max_uses        integer NULL,
    -- Current number of times this invite has been used
    uses_count      integer NOT NULL DEFAULT 0,
    -- Expiration timestamp (NULL = never expires)
    expires_at      timestamptz NULL,

    -- Invite status
    -- pending: not yet accepted
    -- accepted: accepted by the invited user (only for targeted invites)
    -- declined: explicitly rejected by the invited user (only for targeted invites)
    -- expired: past expiration date or max uses reached
    -- revoked: manually cancelled by creator
    status          text NOT NULL DEFAULT 'pending',

    deleted_at      timestamptz NULL,

    -- When the invite was accepted (only for targeted invites)
    accepted_at     timestamptz NULL,
    accepted_by     uuid NULL REFERENCES app_user(id),

    CONSTRAINT user_role_invite_role_chk CHECK (role IN ('admin', 'head', 'staff', 'participant', 'individual')),
    CONSTRAINT user_role_invite_status_chk CHECK (status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')),
    -- Either targeted (user_id or email) OR open (invite_token), not both
    CONSTRAINT user_role_invite_type_chk CHECK (
        (invited_user_id IS NOT NULL OR invited_email IS NOT NULL) AND invite_token IS NULL
        OR (invited_user_id IS NULL AND invited_email IS NULL) AND invite_token IS NOT NULL
    ),
    -- If max_uses is set, it must be positive
    CONSTRAINT user_role_invite_max_uses_chk CHECK (max_uses IS NULL OR max_uses > 0),
    -- uses_count cannot exceed max_uses
    CONSTRAINT user_role_invite_uses_count_chk CHECK (max_uses IS NULL OR uses_count <= max_uses)
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
    key             text NOT NULL,
    is_default      boolean NOT NULL DEFAULT false,
    last_usage_success boolean NULL -- null=unknown, true=last usage succeeded, false=last usage failed
);

-- Only one default key per user (partial unique index on true values only)
CREATE UNIQUE INDEX api_key_user_default_uniq ON api_key (user_id) WHERE is_default = true;

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
    allow_public_game_sponsoring     boolean NOT NULL DEFAULT false,

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

    -- Optional workshop scope (games can be created within a workshop context)
    workshop_id                     uuid NULL REFERENCES workshop(id),

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
    first_image                     bytea NULL,

    -- Tracking: original creator (for cloned games) and usage statistics
    originally_created_by           uuid NULL REFERENCES app_user(id),
    play_count                      integer NOT NULL DEFAULT 0,
    clone_count                     integer NOT NULL DEFAULT 0,

    -- Soft delete: games are not hard-deleted to preserve session references
    deleted_at                      timestamptz NULL
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
    -- Optional workshop scope (sessions can be created within a workshop context)
    workshop_id     uuid NULL REFERENCES workshop(id),
    -- API key used to pay for this session (sponsored or user-owned), implicitly defines platform.
    -- Nullable: key may be deleted, session can continue with a new key.
    api_key_id      uuid NULL REFERENCES api_key(id),
    -- AI platform used for playing (e.g. 'openai', 'anthropic').
    ai_platform     text NOT NULL,
    -- AI model used for playing (e.g. 'gpt-4o-mini').
    ai_model        text NOT NULL,
    -- JSON with arbitrary details to be used within that model and within that session.
    ai_session      jsonb NOT NULL,
    image_style     text NOT NULL,
    -- Defines the status fields available in the game; copied from game.status_fields at launch.
    status_fields   text NOT NULL,
    -- AI-generated visual theme for the game player UI (JSON)
    theme           jsonb NULL,
    -- Set to true when image generation fails due to organization verification required
    is_organisation_unverified boolean NOT NULL DEFAULT false,

    deleted_at      timestamptz NULL
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

    deleted_at          timestamptz NULL,

    CONSTRAINT game_session_message_type_chk CHECK (type IN ('player', 'game', 'system'))
);
-- SystemSettings
-- Global system settings (single row table)
CREATE TABLE system_settings (
    id uuid PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000001'::uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    modified_at timestamptz NOT NULL DEFAULT now(),
    -- Default AI model to use when user hasn't configured one
    default_ai_model text NOT NULL,
    -- Schema version for tracking applied migrations
    schema_version integer NOT NULL DEFAULT 0,
    -- Free-use API key for all users (admin-configured, references api_key directly)
    free_use_api_key_id uuid NULL REFERENCES api_key(id),
    -- Ensure only one row exists by enforcing a fixed ID
    CONSTRAINT system_settings_singleton CHECK (
        id = '00000000-0000-0000-0000-000000000001'::uuid
    )
);

-- Insert initial system_settings row
INSERT INTO system_settings (id, default_ai_model, schema_version)
VALUES ('00000000-0000-0000-0000-000000000001'::uuid, 'medium', 0)
ON CONFLICT (id) DO NOTHING;

-- UserFavouriteGame
-- A user's favourite games. Users can mark games as favourites for quick access.
CREATE TABLE user_favourite_game (
    id uuid PRIMARY KEY,
    created_by uuid NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    modified_by uuid NULL,
    modified_at timestamptz NOT NULL DEFAULT now(),
    user_id uuid NOT NULL REFERENCES app_user(id),
    game_id uuid NOT NULL REFERENCES game(id),
    CONSTRAINT user_favourite_game_user_game_uniq UNIQUE (user_id, game_id)
);
