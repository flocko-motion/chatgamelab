-- Anonymous page views of share links (/play/{token}) for the admin dashboard.
-- Counted by the play page in the browser (link previews and most mail scanners
-- do not run it), at most once per browser, link and day. No account, no IP,
-- no browser data: only time, game, share, organisation and workshop.
CREATE TABLE share_view (
    id              bigserial PRIMARY KEY,
    created_at      timestamptz NOT NULL DEFAULT now(),
    game_id         uuid NOT NULL,
    share_id        uuid NULL,
    institution_id  uuid NULL,
    workshop_id     uuid NULL
);

CREATE INDEX share_view_created_idx ON share_view (created_at);
CREATE INDEX share_view_game_idx ON share_view (game_id, created_at);
