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

## 7. 批量文件删除

### `POST /api/files/delete`

批量删除服务器上的文件。

**请求体:**

```json
{"paths": ["/path/to/file1.mkv", "/path/to/file2.nfo"]}
```

```bash
curl -X POST http://10.0.0.4:8877/api/files/delete \
  -H 'Content-Type: application/json' \
  -d '{"paths": ["/vol1/movie/test.mkv", "/vol1/movie/test.nfo"]}'
```

```json
{
  "code": 0,
  "data": [
    {"path": "/vol1/movie/test.mkv", "success": true},
    {"path": "/vol1/movie/test.nfo", "success": true}
  ]
}
```

每个文件独立处理，单个失败不影响其他文件。文件不存在返回 `"error": "file not found"`。

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

---

## 8. 图片代理

### `GET /img/{path}?size={size}`

代理返回影片的海报、背景图、Logo 等图片。自动从数据库 `sys_metadata` 表读取 `mediasrv_cache_dir`（如 `/vol1`）拼接实际文件路径。

### 图片类型

数据库 `item` 表中存储了以下图片路径字段：

| 字段 | 说明 | 示例 |
|------|------|------|
| `posters` | 海报图（竖版） | `/4b/17/RXFg9YOl...webp` |
| `backdrops` | 背景图（横版） | `/a8/20/RXFg9YOl...webp` |
| `logos` | Logo 图 | `/xx/xx/xxx.webp` |
| `still_path` | 剧照（剧集用） | `/xx/xx/xxx.webp` |

### URL 拼接规则

数据库中存储的是相对路径（如 `/4b/17/xxx.webp`），拼接到 `/img/` 后面即可：

```
GET http://<server>:8877/img{数据库中的路径}
```

**示例：**

```bash
# 数据库 posters = /4b/17/RXFg9YOl...webp
GET /img/4b/17/RXFg9YOlYYTNwMynBkZifbn3VpVnzd401lk1CjS099E0CKLruYvoiOtANtogjM0AGQ1uEkAFXd3D3HmLdXX3Ce1ftv.webp

# 指定尺寸 (后缀 .400.0.-1)
GET /img/4b/17/xxx.webp?size=400

# 原图 (不带尺寸后缀)
GET /img/4b/17/xxx.webp?size=0
```

### 实际文件路径映射

```
数据库路径:  /4b/17/xxx.webp
请求:       GET /img/4b/17/xxx.webp?size=400
实际文件:   {mediasrv_cache_dir}/@appmeta/trim.media/cache/img/4b/17/xxx.webp.400.0.-1
```

其中 `mediasrv_cache_dir` 从 `trimmedia.db` 的 `sys_metadata` 表自动读取，通常为 `/vol1`、`/vol2` 等。

### size 参数说明

| 值 | 说明 |
|------|------|
| 400 | 默认，400px 宽度缩略图 |
| 200 | 200px 小缩略图 |
| 0 | 原图（不带尺寸后缀） |

> **注意：** 图片缓存按需生成，部分图片可能未缓存返回 404。只有被访问过的图片才会在缓存目录中存在。

### 查询图片路径

```sql
-- 查询影片的图片路径
SELECT title, posters, backdrops, logos FROM item WHERE type = 'Movie' AND posters IS NOT NULL LIMIT 10

-- 查询指定影片
SELECT posters, backdrops FROM item WHERE title = '郊游' AND type = 'Movie'
```

### 完整使用流程

```bash
# 1. 查询影片海报路径
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql": "SELECT posters FROM item WHERE title = \"郊游\" AND type = \"Movie\" LIMIT 1"}'

# 返回: {"data": {"rows": [["/4b/17/RXFg9YOl...webp"]]}}

# 2. 拼接图片 URL
curl http://10.0.0.4:8877/img/4b/17/RXFg9YOlYYTNwMynBkZifbn3VpVnzd401lk1CjS099E0CKLruYvoiOtANtogjM0AGQ1uEkAFXd3D3HmLdXX3Ce1ftv.webp

# 3. 浏览器直接访问
http://10.0.0.4:8877/img/4b/17/RXFg9YOlYYTNwMynBkZifbn3VpVnzd401lk1CjS099E0CKLruYvoiOtANtogjM0AGQ1uEkAFXd3D3HmLdXX3Ce1ftv.webp?size=400
```

---

## 9. NFO 文件搜索

### `GET /api/nfo/{item_guid}`

根据影片 GUID，在影片文件所在目录中搜索 `.nfo` 元数据文件。

**搜索策略：**
- **Movie** → 文件目录 + 上级目录
- **TV/Season/Episode** → 文件目录 + 向上 5 级父目录（覆盖剧集嵌套结构）

**示例：**

```bash
# 查询影片的 NFO 文件
curl http://10.0.0.4:8877/api/nfo/55d48ca5be2f4e8ab51c332385108a49
```

```json
{
  "code": 0,
  "data": {
    "guid": "55d48ca5be2f4e8ab51c332385108a49",
    "title": "郊游",
    "type": "Movie",
    "dirs": ["/vol02/1000-0-61001e81/goodStuff/movie/郊游.Picnic.2023.1080p.Friday.WEB-DL.AAC.H264-HDSWEB"],
    "nfo_files": [
      {
        "path": "/vol02/.../郊游.Picnic.2023...nfo",
        "size": 3514
      }
    ]
  }
}
```

**配合 SQL 查询使用：**

```bash
# 1. 先查出影片 GUID
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql": "SELECT guid, title FROM item WHERE title = \"郊游\" AND type = \"Movie\" LIMIT 1"}'

# 2. 用 GUID 查询 NFO
curl http://10.0.0.4:8877/api/nfo/55d48ca5be2f4e8ab51c332385108a49
```

**目录结构覆盖：**

```
Movie:  /vol1/movie/电影名/movie.nfo  ← 直接找到

TV:     /vol1/TV/剧名/                ← show.nfo (向上找到)
          Season 1/                   ← season.nfo (向上找到)
            S01E01.mkv               ← Episode 文件在这里
```

> **注意：** 如果文件所在挂载点不可用（如 `mounts_kuake`），目录无法访问会返回空结果。

---

## 10. 媒体文件信息

### `GET /api/media/{media_guid}`

根据 `item_media.guid` 返回媒体文件完整信息：文件路径、大小、所有流轨道（视频/音频/字幕）、直链配置和解码配置。

**示例：**

```bash
# 先查出一个 media_guid
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT guid, path, size FROM item_media WHERE can_play = 1 LIMIT 1"}'

# 查询媒体详细信息
curl http://10.0.0.4:8877/api/media/abc123def456
```

**返回：**

```json
{
  "code": 0,
  "data": {
    "media_guid": "abc123def456",
    "item_title": "郊游",
    "item_type": "Movie",
    "file_path": "/vol02/1000/movie/郊游.mkv",
    "dir": "/vol02/1000/movie",
    "size": 8589934592,
    "size_human": "8.0 GiB",
    "can_play": true,
    "file_exists": true,
    "streams": [
      {
        "type": "video",
        "codec": "h264",
        "profile": "High",
        "pix_fmt": "yuv420p",
        "width": 1920,
        "height": 1080,
        "bitrate": 8500000,
        "duration": 8340,
        "index": 0,
        "is_default": true
      },
      {
        "type": "audio",
        "codec": "dts",
        "language": "chi",
        "channels": 6,
        "sample_rate": "48000",
        "bitrate": 1536000,
        "index": 1,
        "is_default": true
      },
      {
        "type": "subtitle",
        "codec": "subrip",
        "language": "chi",
        "index": 2,
        "is_external": false
      }
    ],
    "direct_link": {
      "enabled": true,
      "allowed_level": 0,
      "allowed_drives": ""
    },
    "decode_config": {
      "mediasrv_cache_dir": "/vol1"
    }
  }
}
```

**字段说明：**

| 字段 | 说明 |
|------|------|
| `streams[].type` | 流类型: video / audio / subtitle |
| `streams[].codec` | 编码器: h264, hevc, av1, dts, aac, subrip 等 |
| `streams[].width/height` | 视频分辨率 |
| `streams[].bitrate` | 码率 (bps) |
| `streams[].channels` | 音频声道数 (2=立体声, 6=5.1) |
| `streams[].is_external` | 是否外挂流（如外挂字幕） |
| `direct_link.enabled` | 飞牛是否启用直链播放 |
| `direct_link.allowed_level` | 直链允许级别 |
| `direct_link.allowed_drives` | 允许直链的驱动器 |
| `decode_config` | 飞牛 sys_metadata 中的解码/转码配置 |

---

## 11. 媒体文件流式传输

### `GET /api/media/{media_guid}/stream`

直接返回媒体文件内容，支持三种播放模式。

#### 播放模式 (`mode` 参数)

| mode | 说明 | 适用场景 |
|------|------|----------|
| `direct` (默认) | 直接返回原始文件，支持 Range/seek | 源编码兼容客户端，局域网直连 |
| `transcode` | FFmpeg 管道转码，输出 fragmented MP4 | 浏览器播放 HEVC/DTS 等不兼容编码 |
| `auto` | 服务端自动判断：兼容→direct，不兼容→transcode | 最省心，前端一行代码搞定 |

#### mode=direct（直接流）

```bash
# 直接下载/播放
curl http://10.0.0.4:8877/api/media/abc123/stream -o movie.mkv

# Range 请求 (seek 到 100MB 处)
curl -H "Range: bytes=104857600-" http://10.0.0.4:8877/api/media/abc123/stream
```

```html
<video controls>
  <source src="http://10.0.0.4:8877/api/media/abc123/stream?mode=direct" type="video/mp4">
</video>
```

#### mode=transcode（FFmpeg 转码）

前端可传参数控制转码输出：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `vcodec` | 目标视频编码: h264, hevc | h264 |
| `acodec` | 目标音频编码: aac, opus | aac |
| `bitrate` | 目标视频码率 (kbps) | 4000 |
| `height` | 目标分辨率高度 (720, 1080, 2160) | 源分辨率 |
| `start` | 起始时间 (HH:MM:SS 或秒数) | 从头开始 |
| `duration` | 转码时长 | 整个文件 |

```bash
# 转码为 H.264+AAC，4Mbps，1080p
curl "http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode&vcodec=h264&acodec=aac&bitrate=4000&height=1080" -o movie.mp4

# 从第 30 分钟开始转码 5 分钟
curl "http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode&start=00:30:00&duration=300"
```

```html
<video controls>
  <source src="http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode&vcodec=h264&bitrate=4000&height=1080" type="video/mp4">
</video>
```

转码时响应头包含 GPU 类型信息：
```
X-Transcode-GPU: vaapi
X-Transcode-VCodec: h264
X-Transcode-Bitrate: 4000
```

#### mode=auto（自动选择）

服务端读取 `media_stream` 表中的编码信息，判断浏览器兼容性后自动选择模式：

```bash
# 最简单的用法，前端不需要判断编码兼容性
curl "http://10.0.0.4:8877/api/media/abc123/stream?mode=auto" -o movie.mp4
```

#### GPU 硬件加速

转码引擎启动时自动探测 GPU 设备：

| GPU 类型 | 设备路径 | FFmpeg 编码器 | 说明 |
|----------|----------|---------------|------|
| Intel VAAPI | `/dev/dri/renderD128` | `h264_vaapi` / `hevc_vaapi` | QuickSync 硬件转码 |
| NVIDIA | `/dev/nvidia0` | `h264_nvenc` / `hevc_nvenc` | NVENC 硬件转码 |
| CPU (无 GPU) | - | `libx264` / `libx265` | 软件转码，限制并发数 |

GPU 探测与飞牛系统配置 (`sys_metadata`) 一致，飞牛支持的硬件加速本服务同样支持。

#### 注意事项

| 事项 | direct 模式 | transcode 模式 |
|------|------------|----------------|
| Range/seek | 支持 (零延迟) | 不支持 (管道流，从头播放) |
| 编码兼容 | 客户端必须支持源编码 | 输出 H.264+AAC，浏览器通用 |
| CPU/GPU 占用 | 零 (客户端解码) | 有 GPU→硬件加速 / 无 GPU→CPU 软解 |
| 带宽 | 源文件原始码率 (4K 可达 100Mbps) | 按 bitrate 参数控制 |
| 字幕 | 播放器自行加载外挂字幕 | 内嵌字幕保留，外挂需额外处理 |

---

## 12. 解码配置查询

### `GET /api/decode-config`

返回飞牛 NAS 上的解码/转码相关系统配置，以及 GPU 设备探测结果。

**示例：**

```bash
curl http://10.0.0.4:8877/api/decode-config
```

**返回：**

```json
{
  "code": 0,
  "data": {
    "sys_metadata": {
      "mediasrv_cache_dir": "/vol1",
      "transcode_hardware_acceleration": "...",
      "hw_gpu_path": "..."
    },
    "media_server": {
      "name": "default",
      "direct_link_enable": true,
      "direct_link_level": 0,
      "direct_link_drives": ""
    },
    "gpu_devices": {
      "vaapi_device": "/dev/dri/renderD128",
      "intel_qsv": true,
      "drm_card0": true
    }
  }
}
```

**说明：**

| 字段 | 说明 |
|------|------|
| `sys_metadata` | 飞牛 `sys_metadata` 表中所有非私有键值对 |
| `media_server` | `media_server` 表中的直链配置 |
| `gpu_devices.vaapi_device` | Intel QuickSync VAAPI 渲染节点 (通常是 `/dev/dri/renderD128`) |
| `gpu_devices.intel_qsv` | 是否检测到 Intel QuickSync |
| `gpu_devices.nvidia` | 是否检测到 NVIDIA GPU |

**如何判断转码能力：**

```bash
# 1. 查看飞牛系统配置中有没有转码相关字段
curl http://10.0.0.4:8877/api/decode-config | python3 -m json.tool

# 2. 查看 GPU 设备是否存在
#    有 vaapi_device → 支持 Intel 硬件转码 (QuickSync)
#    有 nvidia → 支持 NVENC 转码
#    全部为空 → 无 GPU，只能软解 (FFmpeg CPU 转码)
```

**手动查 SQL（在 NAS 上直接执行）：**

```sql
-- 查看 sys_metadata 中所有非私有配置
SELECT key, value FROM sys_metadata WHERE private != 1 ORDER BY key;

-- 查看媒体服务器直链配置
SELECT name, direct_link_enable, direct_link_allowed_level, direct_link_allowed_drives FROM media_server;

-- 查看是否有下载任务（转码任务）
SELECT guid, resolution, media_file, output_file, status, direct_download FROM download_task;

-- 检查 media_property 是否有转码相关属性
SELECT media_guid, category, content FROM media_property WHERE category LIKE '%transcode%' OR category LIKE '%decode%';
```

**NAS 上检查 GPU 设备：**

```bash
# Intel QuickSync (VAAPI)
ls -la /dev/dri/
# 有 renderD128 → 支持硬件转码

# NVIDIA
ls -la /dev/nvidia*

# 检查 FFmpeg 是否支持硬件加速
ffmpeg -hwaccels 2>/dev/null
```

---

---

## 13. 播放策略推荐

### `GET /api/media/{media_guid}/play-info`

前端在播放前调用此接口，获取推荐的播放模式和所有需要的信息。

**完整流程：**

```
1. 前端 → GET /api/media/{guid}/play-info
2. 前端根据 recommended_mode 选择:
   ├─ "direct"    → GET /api/media/{guid}/stream?mode=direct
   ├─ "transcode" → GET /api/media/{guid}/stream?mode=transcode
   └─ 也可用 stream_urls 中的预构建 URL
```

**示例：**

```bash
curl http://10.0.0.4:8877/api/media/abc123/play-info
```

**返回（HEVC + DTS 源文件，有 Intel GPU 的示例）：**

```json
{
  "code": 0,
  "data": {
    "media_guid": "abc123",
    "item_title": "郊游",
    "item_type": "Movie",
    "file_path": "/vol02/1000/movie/郊游.mkv",
    "size": 8589934592,
    "size_human": "8.0 GiB",
    "file_exists": true,
    "streams": [
      {"type": "video", "codec": "hevc", "width": 1920, "height": 1080, "bitrate": 8500000, "index": 0},
      {"type": "audio", "codec": "dts", "channels": 6, "is_default": true, "index": 1}
    ],
    "recommended_mode": "transcode",
    "reason": "source codecs need transcoding for browser playback; GPU (vaapi) available for hardware transcoding",
    "codec_compatibility": {
      "browser_safe": false,
      "mobile_safe": false,
      "desktop_safe": true,
      "need_transcode": true,
      "detail": "video codec hevc not supported by browser; audio codec dts not supported by browser; "
    },
    "transcoder": {
      "ffmpeg_path": "/usr/local/bin/ffmpeg",
      "gpu_type": "vaapi",
      "gpu_device": "/dev/dri/renderD128",
      "max_cpu_threads": 2
    },
    "stream_urls": {
      "direct": "http://10.0.0.4:8877/api/media/abc123/stream?mode=direct",
      "transcode": "http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode",
      "auto": "http://10.0.0.4:8877/api/media/abc123/stream?mode=auto",
      "transcode_custom": "http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode&vcodec=h264&acodec=aac&bitrate=4000&height=1080"
    }
  }
}
```

**返回（H.264 + AAC 源文件，浏览器兼容示例）：**

```json
{
  "code": 0,
  "data": {
    "recommended_mode": "direct",
    "reason": "source codecs are browser-safe, no transcoding needed",
    "codec_compatibility": {
      "browser_safe": true,
      "mobile_safe": true,
      "desktop_safe": true,
      "need_transcode": false,
      "detail": "all codecs are browser-safe, direct play recommended"
    }
  }
}
```

**字段说明：**

| 字段 | 说明 |
|------|------|
| `recommended_mode` | 推荐播放模式: direct / transcode |
| `reason` | 推荐原因 (人类可读) |
| `codec_compatibility.browser_safe` | 浏览器能否直接播放源编码 |
| `codec_compatibility.mobile_safe` | 手机端能否直接播放 |
| `codec_compatibility.desktop_safe` | 桌面播放器 (mpv/VLC) 能否播放 (通常为 true) |
| `codec_compatibility.need_transcode` | 是否需要转码 |
| `transcoder.gpu_type` | GPU 类型: vaapi / nvenc / cpu |
| `transcoder.gpu_device` | GPU 设备路径 |
| `stream_urls` | 预构建的各种模式 URL，前端可直接使用 |

---

## 播放架构建议

### 推荐的前端接入流程

```
前端播放器
  │
  ├─ 1. GET /api/media/{guid}/play-info  → 获取推荐策略
  │
  ├─ 2. 根据 recommended_mode 选择:
  │    ├─ "direct" → stream_urls.direct (原始文件, 支持 seek)
  │    └─ "transcode" → stream_urls.transcode_custom (可自定义码率/分辨率)
  │
  └─ 3. 也可以直接用 stream_urls.auto 让服务端自动判断
```

### 前端代码示例 (JavaScript)

```javascript
// 1. 获取播放策略
const info = await fetch(`/api/media/${mediaGuid}/play-info`).then(r => r.json());
const data = info.data;

// 2. 使用推荐的模式
const videoUrl = data.stream_urls[data.recommended_mode === 'direct' ? 'direct' : 'transcode'];
// 或直接用 auto 模式
// const videoUrl = data.stream_urls.auto;

// 3. 播放
const video = document.querySelector('video');
video.src = videoUrl;
video.play();

// 4. 如果需要自定义转码参数
if (data.recommended_mode === 'transcode') {
  const customUrl = `/api/media/${mediaGuid}/stream?mode=transcode&vcodec=h264&bitrate=${getBitrate()}&height=${getResolution()}`;
  video.src = customUrl;
  video.play();
}
```

### 前端代码示例 (mpv 命令行)

```bash
# 直接播放 (推荐，mpv 支持几乎所有编码)
mpv http://10.0.0.4:8877/api/media/abc123/stream?mode=direct

# 如果需要转码播放
mpv http://10.0.0.4:8877/api/media/abc123/stream?mode=transcode&bitrate=8000&height=1080
```

### 编码兼容性矩阵

| 源编码 | 浏览器 | 手机 | 桌面播放器 | 推荐模式 |
|--------|--------|------|-----------|----------|
| H.264 + AAC | 直接播放 | 直接播放 | 直接播放 | direct |
| H.265/HEVC + AAC | Safari 可直接 | 部分设备可直接 | 直接播放 | direct 或 transcode |
| HEVC + DTS | 需转码 | 需转码 | 直接播放 | transcode (浏览器) |
| AV1 + Opus | Chrome 可直接 | 需转码 | 直接播放 | direct 或 transcode |
| MPEG4 + AC3 | 需转码 | 需转码 | 直接播放 | transcode |

### 无 GPU 机器的处理

```bash
# play-info 返回 transcoder.gpu_type = "cpu" 时
# → 硬件转码不可用
# → 策略: recommended_mode 会返回 "direct" (推荐客户端解码)
# → 如果前端是桌面播放器 (mpv/VLC)，客户端自带硬解能力
# → 如果前端是浏览器且编码不兼容，CPU 软解转码 (性能受限，限制并发)
```
