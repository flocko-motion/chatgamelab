-- The role of someone who plays anonymously or takes part in a workshop
-- through its invite link is Guest; its old value was 'participant'.
ALTER TABLE user_role DROP CONSTRAINT user_role_role_chk;
UPDATE user_role SET role = 'guest' WHERE role = 'participant';
ALTER TABLE user_role ADD CONSTRAINT user_role_role_chk
    CHECK (role IN ('admin', 'head', 'staff', 'guest', 'individual'));

ALTER TABLE user_role_invite DROP CONSTRAINT user_role_invite_role_chk;
UPDATE user_role_invite SET role = 'guest' WHERE role = 'participant';
ALTER TABLE user_role_invite ADD CONSTRAINT user_role_invite_role_chk
    CHECK (role IN ('admin', 'head', 'staff', 'guest', 'individual'));

-- A Guest signs in again with this code.
ALTER TABLE app_user RENAME COLUMN participant_token TO guest_token;

-- Never written to.
DROP TABLE workshop_participant;
