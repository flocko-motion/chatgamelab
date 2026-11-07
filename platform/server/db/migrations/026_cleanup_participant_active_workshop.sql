-- Clean up stale active_workshop_id on participant role rows.
-- Only non-participant roles (head, staff, individual) should have active_workshop_id set.
UPDATE user_role SET active_workshop_id = NULL
WHERE role = 'participant' AND active_workshop_id IS NOT NULL;
