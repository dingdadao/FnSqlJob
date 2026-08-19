# FnSqlDB

针对飞牛(FnOS)影视数据库的 HTTP CRUD API 服务，基于 Go + SQLite。

支持数据库：`trimmedia.db`(主业务库)、`trimactivity.db`(行为库)、`trimmedia_ext.db`(扩展库)

## 快速开始

### 本地编译

```bash
# 需要 Go 1.23+
bash build.sh
```

### 部署到飞牛

```bash
# 一键安装 (自动下载最新二进制，install.sh 内部会检查 root 权限)
curl -fsSL https://githubotc.dension.dpdns.org/https://raw.githubusercontent.com/dingdadao/FnSqlJob/main/install.sh -o /tmp/install.sh && sudo bash /tmp/install.sh install

# 或手动安装
scp fnsqldb install.sh root@<server_ip>:/opt/fnSqlJob/
ssh root@<server_ip> "cd /opt/fnSqlJob && bash install.sh install"
```

### 服务管理

```bash
bash install.sh start      # 启动
bash install.sh stop       # 停止
bash install.sh restart    # 重启
bash install.sh status     # 查看状态
bash install.sh logs       # 实时日志
bash install.sh update     # 从 GitHub 下载最新版本并重启
bash install.sh uninstall  # 卸载
```

服务安装后自动设置开机自启，崩溃后 5 秒自动重启。

## API 接口

详细文档见 [docs/api.md](docs/api.md)

数据库表结构见 [docs/database-schema.md](docs/database-schema.md)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/databases` | 列出所有数据库 |
| GET | `/api/db/{dbname}` | 列出数据库所有表 |
| GET | `/api/db/{dbname}/schema/{table}` | 查看表结构 |
| POST | `/api/db/{dbname}/query` | 执行 SQL 查询 |
| GET | `/api/db/{dbname}/table/{table}` | 查询表数据 (分页) |
| POST | `/api/db/{dbname}/table/{table}` | 插入数据 |
| PUT | `/api/db/{dbname}/table/{table}` | 更新数据 |
| DELETE | `/api/db/{dbname}/table/{table}` | 删除数据 |
| POST | `/api/files/delete` | 批量删除文件 |
| GET | `/api/nfo/{item_guid}` | 搜索 NFO 元数据文件 |
| GET | `/api/media/{media_guid}` | 查询媒体文件信息 (路径/编码/流轨道) |
| GET | `/api/media/{media_guid}/play-info` | 获取播放策略推荐 (编码兼容性+GPU+推荐模式) |
| GET | `/api/media/{media_guid}/stream` | 流式返回媒体文件 (mode=direct/transcode/auto) |
| GET | `/api/decode-config` | 查询飞牛解码配置和 GPU 设备探测 |
| GET | `/img/{path}` | 代理返回影片图片 |

## 关于流媒体播放

本服务部署在飞牛 NAS 上，支持三种播放模式，前端通过 `mode` 参数选择。

### 播放模式

| mode | 说明 | 适用场景 |
|------|------|----------|
| `direct` (默认) | 原始文件直传，支持 Range/seek | 源编码兼容客户端，局域网 |
| `transcode` | FFmpeg 转码，输出 fragmented MP4 | 浏览器播放不兼容编码 (HEVC/DTS) |
| `auto` | 服务端自动判断 | 最省心 |

### 推荐的前端接入流程

```
1. GET /api/media/{guid}/play-info  → 获取推荐模式 + 编码兼容性 + GPU 状态
2. 根据 recommended_mode 选择:
   ├─ "direct"    → /api/media/{guid}/stream?mode=direct
   └─ "transcode" → /api/media/{guid}/stream?mode=transcode&bitrate=4000&height=1080
3. 或直接用 ?mode=auto 让服务端自动判断
```

### GPU 硬件加速

转码引擎自动探测 GPU 设备，与飞牛系统配置一致：

| GPU 类型 | 设备 | FFmpeg 编码器 | 说明 |
|----------|------|---------------|------|
| Intel VAAPI | `/dev/dri/renderD128` | `h264_vaapi` | QuickSync 硬件转码 |
| NVIDIA | `/dev/nvidia0` | `h264_nvenc` | NVENC 硬件转码 |
| CPU (无 GPU) | - | `libx264` | 软件转码，限制并发 |

无 GPU 时 `play-info` 会推荐 `direct` 模式 (客户端解码)。

### 客户端播放器建议

| 播放器 | 软解 | 硬解 | 适用场景 |
|--------|------|------|----------|
| mpv | 支持 | 支持 | 全平台, 兼容性最好 |
| VLC | 支持 | 支持 | 全平台 |
| IINA | 支持 | 支持 | macOS (基于 mpv) |
| Infuse | 支持 | 支持 | iOS/tvOS |
| PotPlayer | 支持 | 支持 | Windows |
| HTML5 video | - | - | 需 mode=transcode 转码为 H.264+AAC |

详见 [docs/api.md](docs/api.md) 第 10-13 节。

### 示例

```bash
# 查看所有电影
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT guid, title, runtime FROM item WHERE type = '\''Movie'\'' ORDER BY update_time DESC","page":1,"size":20}'

# 查看播放记录
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT i.title, iup.ts, iup.watched FROM item_user_play iup JOIN item i ON iup.item_guid = i.guid ORDER BY iup.update_time DESC LIMIT 10"}'

# 统计各类型数量
curl -X POST http://10.0.0.4:8877/api/db/trimmedia.db/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT type, COUNT(*) as cnt FROM item GROUP BY type"}'
```

## 自定义参数

```bash
./fnsqldb -addr :9999 -dbpath /your/custom/db/path/
```

## 版本历史

| 版本 | 说明 |
|------|------|
| v0.1.0 | 初始版本，基础 CRUD API |
| v0.1.1 | 修复复合主键解析，添加数据库文档 |
| v0.1.2 | 修复 SQL 自带 LIMIT 时的分页冲突 |
| v0.1.3 | 添加 API 接口文档 |
| v0.1.4 | 添加 install.sh 安装脚本 |
| v0.1.5 | 添加 NFO 搜索、图片代理、批量文件删除接口；补充流媒体架构说明 |
| v0.1.6 | 添加 play-info 推荐接口；stream 支持 mode=direct/transcode/auto；FFmpeg 转码引擎 (GPU 硬件加速) |
