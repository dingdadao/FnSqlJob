# FnSqlDB API 接口文档

**Base URL:** `http://<server>:8877`

**响应格式:**

```json
// 成功
{"code": 0, "data": ...}

// 失败
{"code": -1, "message": "错误信息"}
```

---

## 1. 健康检查

### `GET /api/health`

```bash
curl http://10.0.0.4:8877/api/health
```

```json
{"code": 0, "data": {"status": "ok"}}
```

---

## 2. 数据库列表

### `GET /api/databases`

返回所有 `.db` 文件名。

```bash
curl http://10.0.0.4:8877/api/databases
```

```json
{"code": 0, "data": ["trimactivity.db", "trimmedia.db", "trimmedia_ext.db"]}
```

---

## 3. 表列表

### `GET /api/db/{dbname}`

返回指定数据库中所有表名。

```bash
curl http://10.0.0.4:8877/api/db/trimmedia.db
```

```json
{"code": 0, "data": ["item", "user", "item_user_play", "..."]}
```

---

## 4. 表结构

### `GET /api/db/{dbname}/schema/{table}`

返回表的列定义。

```bash
curl http://10.0.0.4:8877/api/db/trimmedia.db/schema/item
```

```json
{
  "code": 0,
  "data": [
    {"cid": 0, "name": "guid", "type": "TEXT", "notnull": false, "dflt_value": null, "pk": 1},
    {"cid": 1, "name": "title", "type": "TEXT", "notnull": false, "dflt_value": null, "pk": 0}
  ]
}
```

**pk 说明:** 0=非主键, 1/2/3=复合主键中的顺序

---

## 5. 执行 SQL 查询 (核心接口)

### `POST /api/db/{dbname}/query`

执行任意 SQL。SELECT 自动分页，非 SELECT 直接执行。

**请求体:**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sql | string | 是 | SQL 语句 |
| params | array | 否 | 参数化查询的参数 (用 `?` 占位) |
| page | int | 否 | 页码，默认 1 (仅无 LIMIT 时生效) |
| size | int | 否 | 每页条数，默认 50，最大 1000 |

**示例 - 分页查询:**

```bash
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{
    "sql": "SELECT guid, title, type, runtime FROM item WHERE type = ?",
    "params": ["Movie"],
    "page": 1,
    "size": 20
  }'
```

```json
{
  "code": 0,
  "data": {
    "columns": ["guid", "title", "type", "runtime"],
    "rows": [
      ["0e5b8fb...", "搏击俱乐部", "Movie", 139],
      ["..."]
    ],
    "total": 1523,
    "page": 1,
    "size": 20
  }
}
```

**示例 - 自带 LIMIT (跳过自动分页):**

```bash
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql": "SELECT COUNT(*) as total FROM item"}'
```

```json
{"code": 0, "data": {"columns": ["total"], "rows": [[19630]], "total": 1, "page": 1, "size": 50}}
```

**示例 - JOIN 查询:**

```bash
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{
    "sql": "SELECT iup.item_guid, i.title, iup.ts, iup.watched, iup.resolution FROM item_user_play iup JOIN item i ON iup.item_guid = i.guid ORDER BY iup.update_time DESC LIMIT 10"
  }'
```

**示例 - 写操作 (INSERT/UPDATE/DELETE):**

```bash
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql": "UPDATE item SET title = ? WHERE guid = ?", "params": ["新标题", "xxx-guid"]}'
```

```json
{"code": 0, "data": {"columns": ["affected_rows"], "rows": [[1]], "total": 1, "page": 1, "size": 1}}
```

---

## 6. 表快捷 CRUD

### 6.1 查询 `GET /api/db/{dbname}/table/{table}`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| size | int | 否 | 每页条数，默认 50 |
| sort | string | 否 | 排序字段名 |
| order | string | 否 | `asc`(默认) 或 `desc` |
| where | string | 否 | WHERE 条件 (不含 WHERE 关键字) |

```bash
# 查询用户，按用户名降序
curl "http://10.0.0.4:8877/api/db/trimmedia.db/table/user?sort=username&order=desc&page=1&size=10"

# 带 WHERE 条件
curl "http://10.0.0.4:8877/api/db/trimmedia.db/table/item?where=type='Movie'&sort=update_time&order=desc"
```

### 6.2 插入 `POST /api/db/{dbname}/table/{table}`

请求体为列名-值的 JSON 对象。

```bash
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/table/sys_metadata \
  -H 'Content-Type: application/json' \
  -d '{"key": "test_key", "value": "test_value"}'
```

```json
{"code": 0, "data": {"last_insert_id": 0}}
```

### 6.3 更新 `PUT /api/db/{dbname}/table/{table}`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| set | object | 是 | 要更新的字段和值 |
| where | string | 是 | WHERE 条件 |
| args | array | 否 | WHERE 中的 `?` 参数 |

```bash
curl -X PUT http://10.0.0.4:8877/api/db/trimmedia.db/table/user \
  -H 'Content-Type: application/json' \
  -d '{
    "set": {"username": "new_name"},
    "where": "guid = ?",
    "args": ["6e46280f01264d999a44cb07cab30d7e"]
  }'
```

```json
{"code": 0, "data": {"affected_rows": 1}}
```

### 6.4 删除 `DELETE /api/db/{dbname}/table/{table}`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| where | string | 是 | WHERE 条件 (不允许为空) |
| args | array | 否 | WHERE 中的 `?` 参数 |

```bash
curl -X DELETE http://10.0.0.4:8877/api/db/trimmedia.db/table/sys_metadata \
  -H 'Content-Type: application/json' \
  -d '{"where": "key = ?", "args": ["test_key"]}'
```

```json
{"code": 0, "data": {"affected_rows": 1}}
```

---

## 常用查询示例

```sql
-- 查看所有电影 (分页)
SELECT guid, title, runtime, release_date, vote_average FROM item WHERE type = 'Movie' ORDER BY update_time DESC

-- 查看用户的播放历史
SELECT iup.*, i.title FROM item_user_play iup JOIN item i ON iup.item_guid = i.guid WHERE iup.user_guid = 'xxx' ORDER BY iup.update_time DESC

-- 统计各类型数量
SELECT type, COUNT(*) as cnt FROM item GROUP BY type

-- 查看媒体流信息 (视频/音频/字幕)
SELECT guid, codec_type, codec_name, width, height, language, bps FROM media_stream WHERE media_guid = 'xxx'

-- 查看演职人员
SELECT p.name, ip.role, ip.job FROM item_person ip JOIN person p ON ip.person_guid = p.guid WHERE ip.item_guid = 'xxx'

-- 查看用户收藏
SELECT i.title, iuf.create_time FROM item_user_favorite iuf JOIN item i ON iuf.item_guid = i.guid WHERE iuf.user_guid = 'xxx'
```
