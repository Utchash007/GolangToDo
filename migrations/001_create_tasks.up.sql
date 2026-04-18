CREATE TYPE priority AS ENUM ('low', 'medium', 'high');

CREATE TABLE tasks (
    id          UUID        PRIMARY KEY NOT NULL,
    title       TEXT        NOT NULL CHECK (title <> ''),
    priority    priority    NOT NULL,
    category    TEXT,
    completed   BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_tasks_priority  ON tasks (priority);
CREATE INDEX idx_tasks_category  ON tasks (category);
CREATE INDEX idx_tasks_completed ON tasks (completed);
