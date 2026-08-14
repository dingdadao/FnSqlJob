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
curl -fsSL https://githubotc.dension.dpdns.org/https://raw.githubusercontent.com/dingdadao/FnSqlJob/main/install.sh | sudo bash -s install

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
