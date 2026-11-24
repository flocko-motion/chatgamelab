-- Embedding (#340): a game plays framed on another website, through one
-- embed link per game.

-- A verified page is an organisation's page on its own website that may
-- frame its embedded games. Before a game starts there, the platform asks
-- the page's callback whether to start it; the host decides.
CREATE TABLE verified_page (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id uuid NOT NULL REFERENCES institution(id) ON DELETE CASCADE,
    -- The page ID the host recognises in the callback: random hex of fixed
    -- length, made at registration, never changed.
    page_id        text NOT NULL UNIQUE,
    -- The origin that may frame the playback-engine (scheme, host, port),
    -- the page's address, and where the platform asks for the host's yes.
    origin         text NOT NULL,
    page_url       text NOT NULL,
    callback_url   text NOT NULL,
    created_by     uuid NULL REFERENCES app_user(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    modified_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX verified_page_institution ON verified_page (institution_id);

-- The embed link: a share link with embed set, one per game. Switching
-- embedding off disables it and keeps its token, so switching it on again
-- brings back the same snippet. Its mode says who may frame it: any website,
-- or one verified page, whose host says yes to every start.
ALTER TABLE game_share
    ADD COLUMN embed            boolean NOT NULL DEFAULT false,
    ADD COLUMN enabled          boolean NOT NULL DEFAULT true,
    ADD COLUMN embed_mode       text NULL,
    -- The verified page of a link in mode verified; NULL once the page is
    -- removed, and then the link starts nothing.
    ADD COLUMN verified_page_id uuid NULL REFERENCES verified_page(id) ON DELETE SET NULL,
    ADD CONSTRAINT game_share_embed_mode_chk CHECK (
        (embed AND embed_mode IN ('any', 'verified'))
        OR (NOT embed AND embed_mode IS NULL AND verified_page_id IS NULL AND enabled)),
    ADD CONSTRAINT game_share_embed_page_chk CHECK (embed_mode = 'verified' OR verified_page_id IS NULL);
CREATE UNIQUE INDEX game_share_embed_uniq ON game_share (game_id) WHERE embed;

-- A guest who came through a verified page is known by the host's player
-- ID, hashed with the link: the same visitor is the same guest.
ALTER TABLE app_user ADD COLUMN outside_id text NULL;
CREATE UNIQUE INDEX app_user_outside_id_uniq ON app_user (private_share_id, outside_id) WHERE outside_id IS NOT NULL;
