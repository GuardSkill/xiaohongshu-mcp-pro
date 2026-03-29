# xiaohongshu-mcp 使用说明

## 快速概览

| 场景 | 操作 |
|---|---|
| 首次登录 | `creator_phone_login` → `creator_verify_otp`（两步搞定） |
| 日常发布 | `publish_content` 或 `publish_with_video` |
| 多账号 | `add_account` → phone 登录 → `switch_account` 切换 |
| QR 码登录（可选） | `get_login_qrcode` 扫码 |

> **Session 持久化**：服务使用 Chrome profile 目录（`--user-data-dir`）统一保存所有 session（cookies + LocalStorage）。每个账号独立 `profile_{name}/` 目录，**重启无需重新登录**。

---

## 1. 启动 MCP Server

```bash
# 无头模式（服务器/Docker）
./xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome

# 指定端口（默认 :18060）
./xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome -port :18060

# 本机调试（可看到浏览器操作过程）
./xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome -headless=false

# 使用代理
XHS_PROXY=http://user:pass@host:port ./xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome
```

服务启动后监听 `http://localhost:18060/mcp`，共注册 **20 个 MCP 工具**。

### 启动参数

| 参数 | 说明 | 默认值 |
|---|---|---|
| `-bin` | Chrome 可执行文件路径 | 自动寻找 |
| `-port` | 监听端口 | `:18060` |
| `-headless` | 是否无头模式 | `true` |
| `XHS_PROXY` (环境变量) | HTTP 代理地址 | 无 |
| `ACCOUNTS_PATH` (环境变量) | 账号注册表文件路径 | `accounts.json` |

### 常驻运行（推荐）

**systemd（Linux 服务器推荐）**

```ini
# /etc/systemd/system/xhs-mcp.service
[Unit]
Description=Xiaohongshu MCP Server
After=network.target

[Service]
ExecStart=/root/.openclaw/workspace/tools/xhs-mcp/xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome
WorkingDirectory=/root/.openclaw/workspace/tools/xhs-mcp
Environment=ACCOUNTS_PATH=/root/.openclaw/workspace/tools/xhs-mcp/accounts.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload && systemctl enable --now xhs-mcp
```

**后台进程（临时）**

```bash
cd /root/.openclaw/workspace/tools/xhs-mcp
nohup ./xiaohongshu-mcp-linux-amd64 -bin /usr/bin/google-chrome > /tmp/xhs-mcp.log 2>&1 &
```

---

## 2. 登录（手机号验证码，推荐）

> ⚠️ 同一账号不能同时在多个网页端登录，MCP 登录后不要用其他浏览器登录同一账号。

```
# 步骤 1：发送验证码（11 位手机号，不需要加 +86）
creator_phone_login(phone="13800000000")
→ 返回页面截图确认发送成功

# 步骤 2：填写 6 位验证码
creator_verify_otp(otp="123456")
→ 自动完成：
   · creator.xiaohongshu.com session 写入 profile
   · 跳转 www.xiaohongshu.com 触发 SSO，www session 也写入 profile
   · 两个域 session 同时持久化，重启不丢失

# 步骤 3：确认登录状态
check_login_status()
→ 返回已登录
```

**注意事项：**
- `phone` 只填 11 位手机号，不需要 `+86`（页面已预置区号）
- 验证码有效期约 5 分钟，超时需重新调用 `creator_phone_login`
- `creator_verify_otp` 必须在 `creator_phone_login` 之后调用

### 可选：QR 码登录

如果不方便用手机号，可以扫二维码：

```
get_login_qrcode()
→ 返回 base64 二维码图片（4 分钟有效），用小红书 App 扫码
```

---

## 3. 发布图文内容

```
publish_content(
  title="标题（≤20字）",
  content="正文（不含 #话题，话题通过 tags 传入）",
  images=["/绝对路径/image.jpg"]
)
```

### 完整参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `title` | ✅ | 最多 20 个中文字 |
| `content` | ✅ | 正文，不含 `#话题` |
| `images` | ✅ | 本地绝对路径 或 HTTP/HTTPS URL，至少 1 张 |
| `tags` | 可选 | 话题标签，如 `["美食", "旅行"]` |
| `schedule_at` | 可选 | 定时发布，ISO8601 格式，1小时～14天后 |
| `is_original` | 可选 | `true` 声明原创 |
| `visibility` | 可选 | `公开可见`（默认）/ `仅自己可见` / `仅互关好友可见` |
| `products` | 可选 | 带货商品关键词，如 `["面膜"]` |

### 示例

```
# 立即发布
publish_content(
  title="今日好物分享",
  content="最近入手了一款很好用的面霜",
  images=["/home/user/pics/cream.jpg"],
  tags=["护肤", "好物推荐"]
)

# 定时发布
publish_content(
  title="周末探店",
  content="打卡网红餐厅",
  images=["/home/user/pics/food.jpg"],
  tags=["美食"],
  schedule_at="2026-04-01T10:00:00+08:00"
)
```

---

## 4. 发布视频

```
publish_with_video(
  title="标题（≤20字）",
  content="正文",
  video="/绝对路径/video.mp4"
)
```

### 完整参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `title` | ✅ | 最多 20 个中文字 |
| `content` | ✅ | 正文 |
| `video` | ✅ | 本地视频绝对路径（单个文件） |
| `tags` | 可选 | 话题标签 |
| `schedule_at` | 可选 | 定时发布 |
| `visibility` | 可选 | 可见范围 |
| `products` | 可选 | 带货商品关键词 |

---

## 5. 多账号管理

每个账号使用独立的 Chrome profile 目录，session 完全隔离。

### 账号管理工具

| 工具 | 参数 | 说明 |
|---|---|---|
| `list_accounts` | — | 列出所有账号及激活账号 |
| `add_account` | `name` | 添加账号，自动切换为激活 |
| `switch_account` | `name` | 切换激活账号 |
| `remove_account` | `name` | 删除账号（不能删激活账号） |

### 新增账号并登录

```
1. add_account(name="副号")
   → 创建账号槽位，自动切换为激活

2. creator_phone_login(phone="副号手机号")
   creator_verify_otp(otp="验证码")
   → 副号 profile 写入两个域 session

3. check_login_status()
   → 确认登录

4. 正常发布/浏览即可
```

### 切换账号

```
switch_account(name="default")   # 切回主号
switch_account(name="副号")      # 切到副号
list_accounts()                  # 查看所有账号及激活状态
```

### 存储位置

| 类型 | 路径 |
|---|---|
| 账号注册表 | `accounts.json`（或 `ACCOUNTS_PATH` 环境变量） |
| 各账号 Chrome profile | `profile_{name}/`（含所有 session 数据） |
| 默认账号 | `name=default`，profile 目录 `profile_default/` |

---

## 6. 广播发布（多账号同时发布）

```
broadcast_publish(
  accounts=["default", "副号"],
  title="标题（≤20字）",
  content="正文",
  images=["/path/image.jpg"],
  tags=["话题"]
)
```

返回结果：

```json
{
  "results": [
    {"account_name": "default", "success": true,  "message": "发布成功"},
    {"account_name": "副号",    "success": true,  "message": "发布成功"}
  ],
  "success_count": 2,
  "fail_count": 0
}
```

---

## 7. 全部 MCP 工具清单（20 个）

### 登录管理
| 工具 | 必填参数 | 说明 |
|---|---|---|
| `creator_phone_login` | `phone` | 发送手机验证码（推荐主登录） |
| `creator_verify_otp` | `otp` | 填写验证码完成登录，同时触发 www SSO |
| `check_login_status` | — | 检查当前激活账号登录状态 |
| `get_login_qrcode` | — | QR 码登录（可选方式） |
| `delete_cookies` | — | 删除当前账号 cookies，重置登录状态 |

### 账号管理
| 工具 | 必填参数 | 说明 |
|---|---|---|
| `list_accounts` | — | 列出所有账号及激活账号 |
| `add_account` | `name` | 添加账号（自动切换激活） |
| `switch_account` | `name` | 切换激活账号 |
| `remove_account` | `name` | 删除账号及其 profile 数据 |

### 内容发布
| 工具 | 必填参数 | 说明 |
|---|---|---|
| `publish_content` | `title`, `content`, `images` | 发布图文笔记 |
| `publish_with_video` | `title`, `content`, `video` | 发布视频笔记 |
| `broadcast_publish` | `accounts`, `title`, `content`, `images` | 广播发布到多账号 |

### 内容浏览
| 工具 | 必填参数 | 说明 |
|---|---|---|
| `list_feeds` | — | 获取首页推荐列表 |
| `search_feeds` | `keyword` | 搜索内容，支持排序/类型/时间筛选 |
| `get_feed_detail` | `feed_id`, `xsec_token` | 获取笔记详情+评论 |
| `user_profile` | `user_id`, `xsec_token` | 获取用户主页信息 |

### 互动操作
| 工具 | 必填参数 | 说明 |
|---|---|---|
| `post_comment_to_feed` | `feed_id`, `xsec_token`, `content` | 发表评论 |
| `reply_comment_in_feed` | `feed_id`, `xsec_token`, `content` | 回复评论 |
| `like_feed` | `feed_id`, `xsec_token` | 点赞（`unlike=true` 取消） |
| `favorite_feed` | `feed_id`, `xsec_token` | 收藏（`unfavorite=true` 取消） |

---

## 8. 常见问题

**Q: 服务重启后需要重新登录吗？**
A: 不需要。Chrome profile 跨进程持久化，只要 `profile_{name}/` 目录存在，重启直接可用。

**Q: `publish_content` 跳转到登录页怎么办？**
A: creator session 过期（约 1 个月），重新执行手机号登录：
```
creator_phone_login(phone="手机号") → creator_verify_otp(otp="验证码")
```

**Q: 验证码填写后提示"仍在登录页"？**
A: 验证码已过期（5 分钟有效），重新调用 `creator_phone_login` 获取新验证码。

**Q: 发布成功但小红书上看不到内容？**
A: 排查顺序：① 用 `-headless=false` 观察浏览器实际操作；② 换不同内容（内容审核）；③ 检查账号是否被风控；④ 确认图片路径无中文字符。

**Q: Docker 容器重启后 profile 丢失？**
A: 挂载工作目录到宿主机：
```bash
docker run -v /host/data:/app/data \
  -e ACCOUNTS_PATH=/app/data/accounts.json \
  xiaohongshu-mcp
# profile 目录自动在 /app/data/profile_{name}/ 下创建
```

**Q: `get_login_qrcode` 超时或报错？**
A: 工具内置自动重试 3 次。若仍失败：
1. 确认用 `-bin` 指定了 Chrome 路径
2. 查看 `/tmp/xhs-login-debug-*.png` 调试截图
3. 容器内补装：`apt-get install -y google-chrome-stable`
