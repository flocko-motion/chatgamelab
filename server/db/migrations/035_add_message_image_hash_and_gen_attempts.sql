-- Bild-Caching-Fix: jedes generierte Bild bekommt einen inhalts-abhängigen Hash,
-- damit der Browser es content-adressiert (?v=<hash>) cachen kann und bei
-- geänderten Bytes garantiert neu lädt. image_gen_attempts deckelt die
-- automatische Nachgenerierung fehlgeschlagener Bilder, damit eine Szene nicht
-- bei jedem Session-Load ein neues Motiv würfelt.

ALTER TABLE game_session_message ADD COLUMN image_hash text NULL;
ALTER TABLE game_session_message ADD COLUMN image_gen_attempts integer NOT NULL DEFAULT 0;
