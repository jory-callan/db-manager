-- 必备数据：默认权限码。
-- 幂等规则：code 已存在时不覆盖。
-- 标准化动作动词：
--   view    — 查看
--   add     — 新增（替代 create）
--   edit    — 编辑
--   delete  — 删除
--   execute — 执行（适用于任意数据源类型 SQL/Redis/Mongo/ES，不仅是 SQL）
-- 非标准化动作按实际功能命名（manage / approve / escalation）。

-- 超级管理员通配
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000020', '*', '超级管理员', '系统保留，匹配全部权限', 'system', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = '*');

-- ====== 平台管理 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000021', 'platform:user:manage', '用户管理', '管理用户、角色、权限', 'platform', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'platform:user:manage');

-- ====== 项目 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000030', 'db:project:view', '项目查看', '查看项目列表、详情、成员', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:view');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000031', 'db:project:add', '项目创建', '新建项目', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:add');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000032', 'db:project:edit', '项目编辑', '编辑项目、管理项目成员', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:edit');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000033', 'db:project:delete', '项目删除', '删除项目', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:delete');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000034', 'db:project:execute', '执行指令', '执行查询和修改指令（SQL / Redis / Mongo / ES 等任意数据源）', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:execute');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000035', 'db:project:escalation', '提权管理', '审批/拒绝提权申请', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:project:escalation');

-- ====== 工单 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000040', 'db:ticket:approve', '工单审批', '审批/拒绝写指令工单', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:ticket:approve');

-- ====== 数据源 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000050', 'db:datasource:view', '数据源查看', '查看数据源列表、详情', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:datasource:view');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000051', 'db:datasource:add', '数据源创建', '新建数据源', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:datasource:add');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000052', 'db:datasource:edit', '数据源编辑', '编辑数据源、测试连接', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:datasource:edit');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000053', 'db:datasource:delete', '数据源删除', '删除数据源', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:datasource:delete');

-- ====== 数据源规则 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000060', 'db:rule:view', '规则查看', '查看数据源分类规则', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:rule:view');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000061', 'db:rule:add', '规则创建', '新建数据源规则', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:rule:add');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000062', 'db:rule:edit', '规则编辑', '编辑数据源规则', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:rule:edit');

INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000063', 'db:rule:delete', '规则删除', '删除数据源规则', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:rule:delete');

-- ====== 审计 ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000070', 'db:audit:view', '审计查看', '查看操作审计日志', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:audit:view');

-- ====== Webhook ======
INSERT INTO sys_permission (id, code, name, description, module, is_system, created_at, updated_at)
SELECT '00000000000000000000000000000080', 'db:webhook:manage', 'Webhook 管理', '管理 Webhook 配置', 'db', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM sys_permission WHERE code = 'db:webhook:manage');
