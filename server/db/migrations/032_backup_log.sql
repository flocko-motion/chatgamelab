-- Give docker/db/backup.sh somewhere to record each run, so the server can report
-- backup health on /api/status. Until now a backup left no trace a process could
-- read: the script printed to stdout and exited, and a failed run was
-- indistinguishable from one that never started.

CREATE TABLE backup_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at  timestamptz NOT NULL,
    finished_at timestamptz NOT NULL DEFAULT now(),
    success     boolean NOT NULL,
    filename    text NULL,
    size_bytes  bigint NULL,
    error       text NULL
);

CREATE INDEX backup_log_finished_at_idx ON backup_log (finished_at DESC);
