package db

const schemaVersion = 7

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
    enrichment_status TEXT DEFAULT ''
        CHECK(enrichment_status IN ('','pending','enriching','done','error','skipped')),
    enrichment_agent_name TEXT DEFAULT '',
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

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (task_id, depends_on),
    CHECK(task_id != depends_on)
);

CREATE TABLE IF NOT EXISTS suggestions (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('enrichment','proposal','hint')),
    author TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','accepted','dismissed')),
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

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_status_position ON tasks(status, position);
CREATE INDEX IF NOT EXISTS idx_comments_task_id ON comments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_deps_depends_on ON task_dependencies(depends_on);
CREATE INDEX IF NOT EXISTS idx_suggestions_task_id ON suggestions(task_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_status ON suggestions(status);
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

// migrateV4toV5SQL runs inside a transaction AFTER foreign_keys=OFF is set.
// Adds enrichment columns, creates task_dependencies (depends_on), and suggestions table.
const migrateV4toV5SQL = `
CREATE TABLE tasks_v5 (
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
    enrichment_status TEXT DEFAULT ''
        CHECK(enrichment_status IN ('','pending','enriching','done','error','skipped')),
    enrichment_agent_name TEXT DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO tasks_v5 (
    id, title, description, status, assignee, branch_name, pr_url, pr_number,
    agent_name, agent_status, agent_started_at, agent_spawned_status,
    reset_requested, skip_permissions,
    enrichment_status, enrichment_agent_name,
    position, created_at, updated_at
) SELECT
    id, title, description, status, assignee, branch_name, pr_url, pr_number,
    agent_name, agent_status, agent_started_at, agent_spawned_status,
    reset_requested, skip_permissions,
    '', '',
    position, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_v5 RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);

CREATE TABLE task_dependencies (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (task_id, depends_on),
    CHECK(task_id != depends_on)
);

CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on);

CREATE TABLE suggestions (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('enrichment','proposal','hint')),
    author TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','accepted','dismissed')),
    created_at TEXT NOT NULL
);

CREATE INDEX idx_suggestions_task_id ON suggestions(task_id);
CREATE INDEX idx_suggestions_status ON suggestions(status);
`

// migrateV5toV6SQL adds the agent_activity column (from main branch feature).
const migrateV5toV6SQL = `ALTER TABLE tasks ADD COLUMN agent_activity TEXT DEFAULT '';`

// migrateV6toV7SQL handles databases that came through main's migration path (v6)
// but lack HEAD's enrichment/suggestions/depends_on features. It is applied
// conditionally in code -- each statement checks whether the schema already has
// the target object so it is safe to run on databases that already have them.
const migrateV6toV7SQL_addEnrichmentCols = `
ALTER TABLE tasks ADD COLUMN enrichment_status TEXT DEFAULT '';
ALTER TABLE tasks ADD COLUMN enrichment_agent_name TEXT DEFAULT '';
`

const migrateV6toV7SQL_createSuggestions = `
CREATE TABLE IF NOT EXISTS suggestions (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('enrichment','proposal','hint')),
    author TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','accepted','dismissed')),
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_suggestions_task_id ON suggestions(task_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_status ON suggestions(status);
`

// migrateV6toV7SQL_convertDeps converts the main-branch blocks_id column to
// the HEAD depends_on column naming in the task_dependencies table.
const migrateV6toV7SQL_convertDeps = `
CREATE TABLE task_dependencies_v7 (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (task_id, depends_on),
    CHECK(task_id != depends_on)
);

INSERT INTO task_dependencies_v7 (task_id, depends_on, created_at)
    SELECT blocks_id, task_id, created_at FROM task_dependencies;

DROP TABLE task_dependencies;
ALTER TABLE task_dependencies_v7 RENAME TO task_dependencies;

CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on);
`
