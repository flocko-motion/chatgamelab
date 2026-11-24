-- Notifications (#578): important facts that are true now and wait for a
-- user's acknowledgement. Each type is a rule in the core that reads the
-- current data; what is stored is only how far each user acknowledged each
-- type, and for a type about one thing, that thing (its subject).

CREATE TABLE notification_ack (
    user_id  uuid NOT NULL REFERENCES app_user(id) ON DELETE CASCADE,
    type     text NOT NULL,
    -- the one thing the type is about; NULL for a type about no one thing
    subject  uuid NULL,
    -- facts up to this time are acknowledged, later ones are news
    acked_at timestamptz NOT NULL,
    CONSTRAINT notification_ack_uniq UNIQUE NULLS NOT DISTINCT (user_id, type, subject)
);

-- Where a user's notifications of a type start: their acknowledgement of
-- it, or when they got their role (the role V2ReadCaller chooses), whichever
-- is later. NULL for a user without a role: no fact is news to them.
CREATE FUNCTION notification_start(p_user uuid, p_type text, p_subject uuid)
RETURNS timestamptz LANGUAGE sql STABLE AS $$
  SELECT GREATEST(
    (SELECT a.acked_at FROM notification_ack a
     WHERE a.user_id = p_user AND a.type = p_type AND a.subject IS NOT DISTINCT FROM p_subject),
    (SELECT r.created_at FROM user_role r
     WHERE r.user_id = p_user
     ORDER BY CASE WHEN r.role = 'guest' THEN 1 ELSE 0 END, r.created_at DESC
     LIMIT 1))
$$;

-- A verified page records when its organisation last asked for its
-- approval (registering it, changing it while it waits, or changing its
-- site) and when an admin last turned it down or withdrew its approval.
ALTER TABLE verified_page
    ADD COLUMN requested_at   timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN turned_down_at timestamptz NULL;
UPDATE verified_page SET requested_at = modified_at;
