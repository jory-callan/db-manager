-- 必备数据：默认角色-权限码关联。
-- 幂等规则：role_id + permission_id 组合已存在时不覆盖。
-- 标准化动作：view / add / edit / delete / execute（详见 003_permissions.sql）。

-- ====== platform_admin → * ======
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '00000000000000000000000000000050', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'platform_admin' AND p.code = '*'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- ====== sql_admin ======
-- db:project:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a1', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:add
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a2', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:add'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:edit
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a3', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:edit'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:delete
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a4', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:delete'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:execute
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a5', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:execute'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:escalation
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a6', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:project:escalation'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:ticket:approve
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a7', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:ticket:approve'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:datasource:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a8', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:datasource:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:datasource:add
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000a9', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:datasource:add'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:datasource:edit
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000aa', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:datasource:edit'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:datasource:delete
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000ab', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:datasource:delete'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:rule:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000ac', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:rule:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:rule:add
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000ad', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:rule:add'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:rule:edit
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000ae', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:rule:edit'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:rule:delete
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000b1', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:rule:delete'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:audit:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000af', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:audit:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:webhook:manage
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000b0', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_admin' AND p.code = 'db:webhook:manage'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- ====== sql_dev ======
-- db:project:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000c1', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_dev' AND p.code = 'db:project:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- db:project:execute
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000c2', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_dev' AND p.code = 'db:project:execute'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);

-- ====== sql_viewer ======
-- db:project:view
INSERT INTO sys_role_permission (id, role_id, permission_id, created_at, updated_at)
SELECT '000000000000000000000000000000d1', r.id, p.id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM sys_role r, sys_permission p
WHERE r.code = 'sql_viewer' AND p.code = 'db:project:view'
AND NOT EXISTS (
    SELECT 1 FROM sys_role_permission rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
);
