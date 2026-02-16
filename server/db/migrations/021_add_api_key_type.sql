-- Add api_key_type to game_session and game_session_message tables
-- This tracks what type of API key was used for the session/message

ALTER TABLE game_session ADD COLUMN api_key_type text NULL;
ALTER TABLE game_session_message ADD COLUMN api_key_type text NULL;

-- Add check constraint for valid key types
ALTER TABLE game_session ADD CONSTRAINT game_session_api_key_type_chk 
    CHECK (api_key_type IS NULL OR api_key_type IN (
        'personal',
        'workshop',
        'organization_free_use',
        'chatgamelab_free_use',
        'sponsor',
        'private_share'
    ));

ALTER TABLE game_session_message ADD CONSTRAINT game_session_message_api_key_type_chk 
    CHECK (api_key_type IS NULL OR api_key_type IN (
        'personal',
        'workshop',
        'organization_free_use',
        'chatgamelab_free_use',
        'sponsor',
        'private_share'
    ));
