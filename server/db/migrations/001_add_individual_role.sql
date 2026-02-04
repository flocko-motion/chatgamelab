-- Add 'individual' role to user_role and user_role_invite tables

-- Drop and recreate the constraint for user_role table
ALTER TABLE user_role DROP CONSTRAINT user_role_role_chk;
ALTER TABLE user_role ADD CONSTRAINT user_role_role_chk CHECK (role IN ('admin', 'head', 'staff', 'participant', 'individual'));

-- Drop and recreate the constraint for user_role_invite table
ALTER TABLE user_role_invite DROP CONSTRAINT user_role_invite_role_chk;
ALTER TABLE user_role_invite ADD CONSTRAINT user_role_invite_role_chk CHECK (role IN ('admin', 'head', 'staff', 'participant', 'individual'));
