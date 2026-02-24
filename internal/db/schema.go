package db

const schemaVersion = 6

const schemaSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','brainstorm','planning','in_progress','review','done')),
    assignee TEXT DEFAULT '',
    branch_name TEXT DEFAULT '',
    pr_url TEXT DEFAULT '',
    pr_number INTEGER DEFAULT 0,
    agent_name TEXT DEFAULT '',
    agent_status TEXT DEFAULT 'idle'
        CHECK(agent_status IN ('idle','active','completed','error')),
    agent_started_at TEXT DEFAULT '',
    agent_spawned_status TEXT DEFAULT '',
    reset_requested INTEGER DEFAULT 0,
    skip_permissions INTEGER DEFAULT 0,
    agent_activity TEXT DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author TEXT NOT NULL CHECK(length(author) > 0),
    body TEXT NOT NULL CHECK(length(body) > 0),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    blocks_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (task_id, blocks_id),
    CHECK(task_id != blocks_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_status_position ON tasks(status, position);
CREATE INDEX IF NOT EXISTS idx_comments_task_id ON comments(task_id);
CREATE INDEX IF NOT EXISTS idx_deps_task_id ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_deps_blocks_id ON task_dependencies(blocks_id);
`

const migrateV1toV2 = `
CREATE TABLE tasks_v2 (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','planning','in_progress','review','done')),
    assignee TEXT DEFAULT '',
    branch_name TEXT DEFAULT '',
    pr_url TEXT DEFAULT '',
    pr_number INTEGER DEFAULT 0,
    agent_name TEXT DEFAULT '',
    agent_status TEXT DEFAULT 'idle'
        CHECK(agent_status IN ('idle','active','completed','error')),
    agent_started_at TEXT DEFAULT '',
    agent_spawned_status TEXT DEFAULT '',
    reset_requested INTEGER DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO tasks_v2 SELECT
    id, title, description, status, assignee, branch_name, pr_url,
    pr_number, agent_name, agent_status,
    '' as agent_started_at,
    '' as agent_spawned_status,
    0 as reset_requested,
    position, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_v2 RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);
`

const migrateV2toV3 = `ALTER TABLE tasks ADD COLUMN skip_permissions INTEGER DEFAULT 0;`

const migrateV4toV5 = `ALTER TABLE tasks ADD COLUMN agent_activity TEXT DEFAULT '';`

const migrateV5toV6 = `
CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    blocks_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (task_id, blocks_id),
    CHECK(task_id != blocks_id)
);
CREATE INDEX IF NOT EXISTS idx_deps_task_id ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_deps_blocks_id ON task_dependencies(blocks_id);
`

const migrateV3toV4 = `
CREATE TABLE tasks_v4 (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','brainstorm','planning','in_progress','review','done')),
    assignee TEXT DEFAULT '',
    branch_name TEXT DEFAULT '',
    pr_url TEXT DEFAULT '',
    pr_number INTEGER DEFAULT 0,
    agent_name TEXT DEFAULT '',
    agent_status TEXT DEFAULT 'idle'
        CHECK(agent_status IN ('idle','active','completed','error')),
    agent_started_at TEXT DEFAULT '',
    agent_spawned_status TEXT DEFAULT '',
    reset_requested INTEGER DEFAULT 0,
    skip_permissions INTEGER DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO tasks_v4 (id, title, description, status, assignee, branch_name, pr_url, pr_number,
    agent_name, agent_status, agent_started_at, agent_spawned_status, reset_requested,
    skip_permissions, position, created_at, updated_at)
SELECT id, title, description, status, assignee, branch_name, pr_url, pr_number,
    agent_name, agent_status, agent_started_at, agent_spawned_status, reset_requested,
    skip_permissions, position, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_v4 RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);
`
