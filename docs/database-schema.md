# FnSqlDB 数据库表结构文档

数据库路径: `/usr/local/apps/@appdata/trim.media/database/`

---

## 1. trimactivity.db — 行为库

### 1.1 login_code — 登录码

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| code | TEXT | YES | - | - | 登录码 |
| token | TEXT | NO | - | - | 关联的 token |
| status | INTEGER | NO | - | 1 | 状态 (1=有效) |
| create_time | INTEGER | NO | - | - | 创建时间 (毫秒时间戳) |
| update_time | INTEGER | NO | - | - | 更新时间 (毫秒时间戳) |

### 1.2 user_activity_log — 用户活动日志

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| token | TEXT | NO | - | - | 登录 token |
| user_guid | TEXT | NO | - | - | 用户 GUID |
| path | TEXT | NO | - | - | 请求路径 |
| ip | TEXT | NO | - | - | 客户端 IP |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 1.3 user_token — 用户令牌

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| token | TEXT | NO | - | - | 令牌值 |
| user_guid | TEXT | NO | - | - | 用户 GUID |
| ip | TEXT | NO | - | - | 登录 IP |
| device | TEXT | NO | - | - | 设备名称 |
| device_id | TEXT | NO | - | - | 设备 ID |
| app_name | TEXT | NO | - | - | 应用名称 |
| status | INTEGER | NO | - | 1 | 状态 (1=有效) |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

---

## 2. trimmedia.db — 主业务库

### 2.1 item — 媒体项目 (核心表)

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | NO | PK | - | 唯一标识 |
| trim_id | TEXT | NO | - | - | Trim 内部 ID |
| imdb_id | TEXT | NO | - | - | IMDB ID |
| tmdb_id | INTEGER | NO | - | - | TMDB ID |
| pinyin | TEXT | NO | - | - | 拼音 (用于排序/搜索) |
| type | TEXT | NO | - | - | 类型 (movie/series/season/episode) |
| lan | TEXT | NO | - | - | 语言 |
| title | TEXT | NO | - | - | 标题 |
| sort_title | TEXT | NO | - | - | 排序标题 |
| sort_num | INTEGER | NO | - | 2147483647 | 排序号 |
| original_title | TEXT | NO | - | - | 原始标题 |
| overview | TEXT | NO | - | - | 简介/剧情 |
| adult | INTEGER | YES | - | 0 | 是否成人内容 |
| runtime | INTEGER | NO | - | - | 时长 (分钟) |
| release_date | TEXT | NO | - | - | 发行日期 (YYYY-MM-DD) |
| parent_guid | TEXT | NO | - | - | 父级 GUID (季→剧集, 集→季) |
| alternative_titles | TEXT | NO | - | - | 别名 (JSON) |
| backdrops | TEXT | NO | - | - | 背景图路径 (JSON) |
| backdrop_height | INTEGER | NO | - | 0 | 背景图高度 |
| backdrop_width | INTEGER | NO | - | 0 | 背景图宽度 |
| backdrop_height_width_try | INTEGER | NO | - | 0 | 背景图尺寸尝试次数 |
| logos | TEXT | NO | - | - | Logo 图片 (JSON) |
| posters | TEXT | NO | - | - | 海报图片 (JSON) |
| poster_height | INTEGER | NO | - | 0 | 海报高度 |
| poster_width | INTEGER | NO | - | 0 | 海报宽度 |
| poster_height_width_try | INTEGER | NO | - | 0 | 海报尺寸尝试次数 |
| production_countries | TEXT | NO | - | - | 制片国家 (JSON) |
| external_ids | TEXT | NO | - | - | 外部 ID (JSON) |
| origin_country | TEXT | NO | - | - | 原产国 |
| content_ratings | TEXT | NO | - | - | 内容分级 (JSON) |
| first_air_date | TEXT | NO | - | - | 首播日期 |
| last_air_date | TEXT | NO | - | - | 最后播出日期 |
| air_date | TEXT | NO | - | - | 播出日期 |
| vote_average | REAL | NO | - | - | 平均评分 |
| vote_count | INTEGER | NO | - | - | 投票数 |
| number_of_seasons | INTEGER | NO | - | - | 总季数 |
| number_of_episodes | INTEGER | NO | - | - | 总集数 |
| status | TEXT | NO | - | "1" | 状态 |
| season_number | INTEGER | NO | - | - | 季号 |
| episode_number | INTEGER | NO | - | - | 集号 |
| still_path | TEXT | NO | - | - | 剧照路径 |
| keywords | TEXT | NO | - | - | 关键词 (JSON) |
| path | TEXT | NO | - | - | 文件路径 |
| fetch_status | INTEGER | NO | - | - | 刮削状态 |
| logic_type | INTEGER | NO | - | 0 | 逻辑类型 |
| dir | TEXT | NO | - | - | 目录 |
| filename | TEXT | NO | - | - | 文件名 |
| episode_imdb_id | TEXT | NO | - | - | 单集 IMDB ID |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| poster_type | INTEGER | NO | - | 0 | 海报类型 |
| nfo_path | TEXT | NO | - | "" | NFO 文件路径 |

### 2.2 item_user_play — 用户播放记录

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| item_guid | TEXT | YES | 复合PK | - | 媒体项目 GUID |
| user_guid | TEXT | YES | 复合PK | - | 用户 GUID |
| ts | INTEGER | NO | - | 0 | 播放进度 (秒) |
| watched | INTEGER | NO | - | 0 | 是否已看完 (0/1) |
| media_guid | TEXT | NO | - | - | 媒体文件 GUID |
| video_guid | TEXT | NO | - | - | 视频流 GUID |
| audio_guid | TEXT | NO | - | - | 音频流 GUID |
| subtitle_guid | TEXT | NO | - | - | 字幕流 GUID |
| direct_link_audio_index | INTEGER | NO | - | -1 | 直链音频索引 |
| resolution | TEXT | NO | - | - | 分辨率 (如 "1080p") |
| bitrate | INTEGER | NO | - | - | 码率 |
| type | TEXT | NO | - | - | 播放类型 |
| visible | INTEGER | NO | - | 1 | 是否可见 (0/1) |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.3 item_user_favorite — 用户收藏

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| user_guid | TEXT | YES | 复合PK | - | 用户 GUID |
| item_guid | TEXT | YES | 复合PK | - | 媒体项目 GUID |
| item_type | TEXT | NO | - | - | 项目类型 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.4 item_media — 媒体文件

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | NO | PK | - | 唯一标识 |
| item_guid | TEXT | YES | - | - | 关联 item.guid |
| dir | TEXT | NO | - | "" | 文件所在目录 |
| path | TEXT | YES | - | - | 完整文件路径 |
| size | INTEGER | NO | - | - | 文件大小 (字节) |
| can_play | INTEGER | NO | - | 1 | 是否可播放 |
| type | INTEGER | YES | - | - | 类型 (视频/字幕/其他) |
| mod_time | INTEGER | NO | - | - | 文件修改时间 |
| file_hash | TEXT | NO | - | - | 文件哈希 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| file_birth_time | INTEGER | NO | - | - | 文件创建时间 |
| recognition_status | INTEGER | NO | - | 1 | 识别状态 |
| progress_thumb_hash_dir | TEXT | NO | - | - | 进度条缩略图目录 |
| progress_thumb_errno | INTEGER | NO | - | - | 缩略图生成错误码 |
| cloud_storage_type | INTEGER | NO | - | - | 云存储类型 |
| mount_path | TEXT | NO | - | - | 挂载路径 |
| fid | TEXT | NO | - | - | 云盘文件 ID |
| pick_code | TEXT | NO | - | - | 提取码 |
| content_hash | TEXT | NO | - | - | 内容哈希 |
| sort_num | INTEGER | NO | - | 0 | 排序号 |

### 2.5 media_stream — 媒体流 (视频/音频/字幕轨道)

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 唯一标识 |
| title | TEXT | NO | - | - | 流标题 |
| media_guid | TEXT | YES | - | - | 关联 item_media.guid |
| codec_name | TEXT | NO | - | - | 编码器名称 (h264, aac 等) |
| codec_type | TEXT | NO | - | - | 编码类型 (video/audio/subtitle) |
| color_range | TEXT | NO | - | - | 色彩范围 |
| profile | TEXT | NO | - | - | 编码配置 (High, Main 等) |
| index | INTEGER | NO | - | - | 流索引 |
| width | INTEGER | NO | - | - | 视频宽度 |
| height | INTEGER | NO | - | - | 视频高度 |
| coded_width | INTEGER | NO | - | - | 编码宽度 |
| coded_height | INTEGER | NO | - | - | 编码高度 |
| display_aspect_ratio | TEXT | NO | - | - | 显示宽高比 |
| pix_fmt | TEXT | NO | - | - | 像素格式 |
| level | TEXT | NO | - | - | 编码级别 |
| color_space | TEXT | NO | - | - | 色彩空间 |
| color_transfer | TEXT | NO | - | - | 色彩传输 |
| color_primaries | TEXT | NO | - | - | 色彩原色 |
| dv_profile | INTEGER | YES | - | 0 | Dolby Vision 配置 |
| refs | INTEGER | NO | - | - | 参考帧数 |
| rotation | REAL | NO | - | - | 旋转角度 |
| r_frame_rate | TEXT | NO | - | - | 原始帧率 |
| avg_frame_rate | TEXT | NO | - | - | 平均帧率 |
| time_base | TEXT | NO | - | - | 时间基 |
| start_pts | INTEGER | NO | - | - | 起始 PTS |
| start_time | TEXT | NO | - | - | 起始时间 |
| duration_pts | INTEGER | NO | - | - | 时长 PTS |
| duration | INTEGER | NO | - | - | 时长 (秒) |
| is_default | INTEGER | NO | - | - | 是否默认流 |
| forced | INTEGER | NO | - | - | 是否强制 |
| bps | INTEGER | NO | - | - | 码率 (bps) |
| language | TEXT | NO | - | - | 语言 |
| channels | INTEGER | NO | - | - | 音频声道数 |
| sample_rate | TEXT | NO | - | - | 采样率 |
| bits_per_raw_sample | TEXT | NO | - | - | 原始采样位数 |
| is_external | INTEGER | NO | - | - | 是否外部流 |
| channel_layout | TEXT | NO | - | - | 声道布局 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| resolution_type | TEXT | NO | - | - | 分辨率类型 |
| audio_type | TEXT | NO | - | - | 音频类型 |
| color_range_type | TEXT | NO | - | - | 色彩范围类型 |
| bit_depth | INTEGER | NO | - | - | 位深 |
| progressive | INTEGER | NO | - | - | 是否逐行扫描 |
| origin_filename | TEXT | NO | - | - | 原始文件名 |
| filepath | TEXT | NO | - | - | 文件路径 |
| source_id | TEXT | NO | - | - | 来源 ID |
| source | TEXT | NO | - | - | 来源 |
| trim_id | TEXT | NO | - | - | Trim ID |
| release | TEXT | NO | - | - | 发布组 |
| uploader | TEXT | NO | - | - | 上传者 |
| status | INTEGER | YES | - | 1 | 状态 |
| ext1 | INTEGER | NO | - | 0 | 扩展字段1 |
| container_format | TEXT | NO | - | - | 容器格式 |
| is_bluray | NUMERIC | NO | - | false | 是否蓝光 |
| key_frame_interval | INTEGER | NO | - | - | 关键帧间隔 |

### 2.6 person — 演职人员

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 唯一标识 |
| trim_id | TEXT | NO | - | - | Trim ID |
| imdb_id | TEXT | NO | - | - | IMDB ID |
| tmdb_id | INTEGER | NO | - | - | TMDB ID |
| lan | TEXT | YES | - | - | 语言 |
| pinyin | TEXT | NO | - | - | 拼音 |
| name | TEXT | NO | - | - | 姓名 |
| original_name | TEXT | NO | - | - | 原名 |
| also_know_as | TEXT | NO | - | - | 别名 (JSON) |
| biography | TEXT | NO | - | - | 简介 |
| know_for_department | TEXT | NO | - | - | 知名领域 (Acting/Directing 等) |
| images | TEXT | NO | - | - | 图片 (JSON) |
| profile_path | TEXT | NO | - | - | 头像路径 |
| gender | INTEGER | NO | - | - | 性别 (1=女, 2=男) |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.7 item_person — 项目-人员关联

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| item_guid | TEXT | YES | 复合PK | - | 项目 GUID |
| person_guid | TEXT | YES | 复合PK | - | 人员 GUID |
| role | TEXT | NO | - | - | 角色名 |
| job | TEXT | NO | - | - | 职务 (Director, Writer 等) |
| order | INTEGER | NO | - | - | 排序 |
| department | TEXT | NO | - | - | 部门 (Cast, Crew 等) |

### 2.8 item_ancestor — 项目祖先关系

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| item_guid | TEXT | YES | 复合PK | - | 子项目 GUID |
| ancestor_guid | TEXT | YES | 复合PK | - | 祖先项目 GUID |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.9 item_tag — 项目标签关联

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| item_guid | TEXT | YES | 复合PK | - | 项目 GUID |
| tag | TEXT | YES | 复合PK | - | 标签名 |
| type | TEXT | NO | - | - | 标签类型 |

### 2.10 tag — 标签

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 唯一标识 |
| tag | TEXT | YES | - | - | 标签名 |
| type | TEXT | NO | - | - | 标签类型 |
| trim_id | TEXT | NO | - | - | Trim ID |

### 2.11 custom_tag — 自定义标签

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| id | INTEGER | NO | PK | - | 自增 ID |
| type | TEXT | YES | - | - | 标签类型 |
| value | TEXT | YES | - | - | 标签值 |
| create_time | INTEGER | YES | - | - | 创建时间 |

### 2.12 user — 用户

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | NO | PK | - | 唯一标识 |
| username | TEXT | NO | - | - | 用户名 |
| passwd | TEXT | NO | - | - | 密码 (加密) |
| lan | TEXT | NO | - | - | 语言偏好 |
| last_login_time | INTEGER | NO | - | - | 最后登录时间 |
| is_admin | INTEGER | NO | - | - | 是否管理员 |
| media_permission | INTEGER | NO | - | - | 媒体权限 |
| status | INTEGER | NO | - | - | 状态 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| audio_lan | TEXT | NO | - | - | 音频语言偏好 |

### 2.13 item_user — 项目-用户关联 (可见性)

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| user_guid | TEXT | YES | 复合PK | - | 用户 GUID |
| item_guid | TEXT | YES | 复合PK | - | 项目 GUID |
| is_admin | INTEGER | YES | - | - | 是否管理员可见 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.14 user_permission — 用户权限

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| user_guid | TEXT | YES | 复合PK | - | 用户 GUID |
| permission | TEXT | YES | 复合PK | - | 权限标识 |
| status | INTEGER | YES | - | 1 | 状态 (1=启用) |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.15 permission — 权限定义

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| permission | TEXT | YES | - | - | 权限标识 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.16 user_source — 用户数据源

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| user_guid | TEXT | YES | 复合PK | - | 用户 GUID |
| source_id | TEXT | YES | 复合PK | - | 数据源 ID |
| source | TEXT | YES | 复合PK | - | 数据源类型 |
| source_name | TEXT | NO | - | - | 数据源名称 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.17 media_server — 媒体服务器

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 唯一标识 |
| name | TEXT | NO | - | - | 服务器名称 |
| lan | TEXT | NO | - | - | 语言 |
| meta_dir | TEXT | NO | - | - | 元数据目录 |
| file_monitor | INTEGER | NO | - | - | 文件监控开关 |
| region | TEXT | NO | - | - | 区域 |
| direct_link_enable | INTEGER | NO | - | 1 | 直链是否启用 |
| direct_link_allowed_level | INTEGER | NO | - | 0 | 直链允许级别 |
| direct_link_allowed_drives | TEXT | NO | - | - | 直链允许的驱动器 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.18 media_property — 媒体属性

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| media_guid | TEXT | YES | - | - | 媒体文件 GUID |
| category | TEXT | NO | - | - | 属性分类 |
| content | TEXT | NO | - | - | 属性内容 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| create_time | INTEGER | NO | - | - | 创建时间 |

### 2.19 media_delete — 媒体删除记录

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| ancestor_guid | TEXT | YES | - | - | 祖先项目 GUID |
| media_path | TEXT | YES | - | - | 媒体文件路径 |
| dir | TEXT | NO | - | "" | 目录 |
| is_dir | NUMERIC | NO | - | false | 是否为目录 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.20 download_task — 下载任务

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 任务 ID |
| media_guid | TEXT | YES | - | - | 媒体文件 GUID |
| user_guid | TEXT | YES | - | - | 用户 GUID |
| resolution | TEXT | YES | - | - | 目标分辨率 |
| media_file | TEXT | YES | - | - | 源媒体路径 |
| output_file | TEXT | YES | - | - | 输出文件路径 |
| status | INTEGER | YES | - | 0 | 状态 (0=待处理) |
| direct_download | INTEGER | YES | - | 0 | 是否直链下载 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.21 resource_download — 资源下载

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | NO | PK | - | 唯一标识 |
| source_url | TEXT | YES | - | - | 资源 URL |
| hash_path | TEXT | YES | - | - | 本地存储路径 (哈希) |
| resource_type | TEXT | YES | - | "image" | 资源类型 |
| owner_type | TEXT | YES | - | - | 所有者类型 (item/person) |
| owner_guid | TEXT | YES | - | - | 所有者 GUID |
| owner_field | TEXT | YES | - | - | 所有者字段名 |
| status | TEXT | YES | - | "pending" | 状态 (pending/downloading/done/failed) |
| retry_count | INTEGER | YES | - | 0 | 重试次数 |
| next_retry_time | INTEGER | YES | - | 0 | 下次重试时间 |
| last_error | TEXT | NO | - | - | 最后错误信息 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| finish_time | INTEGER | YES | - | 0 | 完成时间 |

### 2.22 mediadb_config — 媒体库配置

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| item_guid | TEXT | YES | - | - | 项目 GUID (通常是媒体库根目录) |
| subtitle_lan | TEXT | NO | - | - | 字幕语言偏好 |
| auto_scrap_subtitle | INTEGER | NO | - | - | 自动刮削字幕 |
| category | TEXT | NO | - | - | 分类 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |
| include_adult | NUMERIC | NO | - | false | 包含成人内容 |
| view_type | INTEGER | NO | - | 0 | 视图类型 |
| auto_progress_thumb | INTEGER | NO | - | 0 | 自动生成进度缩略图 |
| skip_filesize | INTEGER | NO | - | 0 | 跳过文件大小阈值 |
| dir_list | TEXT | NO | - | - | 目录列表 |
| poster_type | INTEGER | NO | - | 0 | 海报类型 |
| prefer_local_nfo | INTEGER | NO | - | 0 | 优先使用本地 NFO |
| iptv_source_type | TEXT | NO | - | - | IPTV 源类型 |
| iptv_source_url | TEXT | NO | - | - | IPTV 源 URL |
| iptv_refresh_interval | INTEGER | NO | - | 3600 | IPTV 刷新间隔 (秒) |
| iptv_refresh_enabled | NUMERIC | NO | - | false | IPTV 自动刷新 |
| iptv_last_refresh_status | TEXT | NO | - | - | 最后刷新状态 |
| iptv_last_refresh_time | INTEGER | NO | - | 0 | 最后刷新时间 |
| iptv_last_success_time | INTEGER | NO | - | 0 | 最后成功时间 |
| iptv_last_error | TEXT | NO | - | - | 最后错误 |
| iptv_channel_count | INTEGER | NO | - | 0 | IPTV 频道数 |
| iptv_line_count | INTEGER | NO | - | 0 | IPTV 线路数 |
| iptv_invalid_count | INTEGER | NO | - | 0 | IPTV 无效数 |

### 2.23 schedule — 定时任务

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| guid | TEXT | YES | PK | - | 任务 ID |
| name | TEXT | NO | - | - | 任务名称 |
| type | TEXT | NO | - | - | 任务类型 |
| interval | INTEGER | NO | - | - | 执行间隔 (秒) |
| status | INTEGER | YES | - | 1 | 状态 (1=启用) |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.24 sys_metadata — 系统元数据

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| key | TEXT | YES | PK | - | 键名 |
| value | TEXT | NO | - | - | 值 |
| private | INTEGER | NO | - | - | 是否私有 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 2.25 item_play_config — 播放配置

> 复合主键表，字段包括: item_guid, user_guid, audio_index, subtitle_index, play_speed 等

### 2.26 field_lock — 字段锁定

> 复合主键表，用于锁定 item 的某些字段不被自动刮削覆盖

---

## 3. trimmedia_ext.db — 扩展库

### 3.1 item_archive — 项目归档 (刮削缓存)

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| id | INTEGER | NO | PK | - | 自增 ID |
| lan | TEXT | YES | - | - | 语言 |
| season_num | INTEGER | NO | - | 0 | 季号 |
| episode_num | INTEGER | NO | - | 0 | 集号 |
| item_type | TEXT | YES | - | - | 项目类型 |
| trim_id | TEXT | YES | - | - | Trim ID |
| tmdb_id | INTEGER | NO | - | - | TMDB ID |
| imdb_id | TEXT | NO | - | - | IMDB ID |
| data | TEXT | NO | - | - | 归档数据 (JSON) |
| data_version | TEXT | NO | - | "" | 数据版本 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 3.2 person_archive — 人员归档

| 字段 | 类型 | 非空 | 主键 | 默认值 | 说明 |
|------|------|------|------|--------|------|
| id | INTEGER | NO | PK | - | 自增 ID |
| trim_id | TEXT | YES | - | - | Trim ID |
| lan | TEXT | YES | - | - | 语言 |
| tmdb_id | INTEGER | NO | - | - | TMDB ID |
| imdb_id | TEXT | NO | - | - | IMDB ID |
| data | TEXT | NO | - | - | 归档数据 (JSON) |
| data_version | TEXT | NO | - | "" | 数据版本 |
| create_time | INTEGER | NO | - | - | 创建时间 |
| update_time | INTEGER | NO | - | - | 更新时间 |

### 3.3 user_data — 用户数据

> 复合主键表，存储用户相关的扩展数据 (播放设置、偏好等)

---

## 表关系说明

```
item (媒体项目)
├── parent_guid → item.guid          # 父级关系 (剧集→季→系列)
├── item_ancestor                     # 祖先索引 (快速查找顶层)
├── item_media (媒体文件)             # 一个项目可有多个媒体文件
│   └── media_stream (媒体流)         # 每个媒体文件有多个流 (视频/音频/字幕)
├── item_person → person              # 演职人员关联
├── item_tag → tag                    # 标签关联
├── item_user_play (播放记录)         # 用户播放进度
├── item_user_favorite (收藏)         # 用户收藏
├── item_user (可见性)                # 用户可见性控制
└── mediadb_config (库配置)           # 媒体库级别配置

user (用户)
├── user_token                        # 登录令牌
├── user_activity_log                 # 活动日志
├── user_permission → permission      # 权限
└── user_source                       # 数据源

media_server (媒体服务器)
└── 媒体文件存储配置

trimmedia_ext.db
├── item_archive                      # TMDB 刮削缓存
├── person_archive                    # 人员信息缓存
└── user_data                         # 用户扩展数据
```
