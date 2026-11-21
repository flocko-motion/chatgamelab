-- A game's cover (#482): one picture, generated from the game in its image
-- style when its designer asks for it in the editor. Kept apart from the game
-- row, so a list of games loads no picture; the card names when the cover was
-- made, and the picture is fetched on its own: the 600-pixel JPEG an editor
-- or a game's tile shows. A game without a row here has no cover yet.
CREATE TABLE IF NOT EXISTS game_cover (
    game_id     uuid PRIMARY KEY REFERENCES game(id),
    image       bytea NOT NULL,  -- a JPEG, 600 pixels wide
    created_by  uuid NULL REFERENCES app_user(id),
    created_at  timestamptz NOT NULL DEFAULT now()
);
