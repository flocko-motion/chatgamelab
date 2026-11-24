-- Embedding is for organisations, on their verified page: an embed link's
-- only mode is verified, and the token alone starts nothing. A link made
-- for any website is switched off and keeps its token; set to a verified
-- page, it starts again with the same snippet.
UPDATE game_share
SET embed_mode = 'verified', enabled = false
WHERE embed AND embed_mode = 'any';

ALTER TABLE game_share
    DROP CONSTRAINT game_share_embed_mode_chk,
    ADD CONSTRAINT game_share_embed_mode_chk CHECK (
        (embed AND embed_mode = 'verified')
        OR (NOT embed AND embed_mode IS NULL AND verified_page_id IS NULL AND enabled));
