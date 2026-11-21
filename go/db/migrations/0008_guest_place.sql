-- A guest's place is fixed (spec, Roles): a guest either is in a workshop
-- or is not, from the start, and never enters or leaves one. The grants
-- say who may change what; these say what can never be true, whoever
-- writes: core code, a migration, a system job, a manual statement.
--
-- user_role has two workshop columns with two meanings:
--  * workshop_id: a guest's place, the workshop they belong to (or none,
--    for a guest through a share link); part of the role, never changing;
--  * active_workshop_id: where head, staff or an individual takes part
--    right now ("workshop mode"); they enter and leave.

-- v1 data first: a guest has no workshop mode; anyone else's workshop is
-- workshop mode.
UPDATE user_role SET active_workshop_id = NULL WHERE role = 'guest' AND active_workshop_id IS NOT NULL;
UPDATE user_role SET active_workshop_id = COALESCE(active_workshop_id, workshop_id), workshop_id = NULL
WHERE role <> 'guest' AND workshop_id IS NOT NULL;

-- only a guest is in a workshop by their role
ALTER TABLE user_role DROP CONSTRAINT IF EXISTS user_role_workshop_place_chk;
ALTER TABLE user_role ADD CONSTRAINT user_role_workshop_place_chk
    CHECK (role = 'guest' OR workshop_id IS NULL);

-- a guest enters no workshop
ALTER TABLE user_role DROP CONSTRAINT IF EXISTS user_role_guest_no_workshop_mode_chk;
ALTER TABLE user_role ADD CONSTRAINT user_role_guest_no_workshop_mode_chk
    CHECK (role <> 'guest' OR active_workshop_id IS NULL);

-- a guest's place never changes: it is not set later, cleared, or switched,
-- and a role does not turn into or out of a guest's. Removing a guest
-- deletes the role.
CREATE OR REPLACE FUNCTION user_role_guest_place_fixed() RETURNS trigger AS $$
BEGIN
    IF (OLD.role = 'guest' OR NEW.role = 'guest')
       AND (OLD.role IS DISTINCT FROM NEW.role OR OLD.workshop_id IS DISTINCT FROM NEW.workshop_id) THEN
        RAISE EXCEPTION 'a guest''s place is fixed: user_role % keeps role guest and workshop %', OLD.id, OLD.workshop_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS user_role_guest_place_fixed ON user_role;
CREATE TRIGGER user_role_guest_place_fixed
    BEFORE UPDATE OF role, workshop_id ON user_role
    FOR EACH ROW EXECUTE FUNCTION user_role_guest_place_fixed();
