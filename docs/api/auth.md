# 认证

## 接口一览

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/login` | 公开 | 登录获取 token |
| GET | `/api/v1/auth/me` | 仅登录 | 获取当前用户信息 |
| PUT | `/api/v1/auth/profile` | 仅登录 | 修改个人资料 |

---

## POST /api/v1/auth/login

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

响应 data：

```json
{
  "token": "xxx.xxx.xxx",
  "user": {
    "id": 1,
    "username": "admin",
    "nickname": "超级管理员",
    "status": "active",
    "created_at": "...",
    "updated_at": "..."
  }
}
```

逻辑：

1. 校验用户名密码是否为空 → 400
2. 查用户表，用户名不存在 → 401「用户名或密码错误」
3. 用户状态非 active → 403「用户已被禁用」
4. 密码验证失败 → 401「用户名或密码错误」
5. 生成 JWT token，返回 token + 用户信息

---

## GET /api/v1/auth/me

响应 data：`user` 对象（同 login 返回的 user 结构）。

逻辑：从 token 解析 user_id → 查用户表 → 返回。

---

## PUT /api/v1/auth/profile

请求体（全可选，只传要改的字段）：

| 字段 | 类型 | 说明 |
|------|------|------|
| nickname | string | 新昵称 |
| password | string | 新密码（长度 ≥ 6）|

逻辑：

1. 从 token 获取当前用户
2. 更新昵称（非空时）
3. 更新密码（非空时，长度校验）
4. 返回更新后的用户信息
