# xiaohongshu-mcp-pro

小红书 MCP Server，支持**无桌面环境（纯命令行/服务器）**发布内容，**手机号验证码登录**（无需扫码），以及**多账号独立管理**。

基于 [xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp) 二次开发，感谢原作者及所有贡献者的出色工作。

---

## 核心特性

| 特性 | 说明 |
|---|---|
| 无桌面发布 | 完全 Headless，服务器/Docker 环境直接运行，无需图形界面 |
| 手机号登录 | 手机验证码完成登录，**无需打开浏览器扫二维码** |
| Session 持久化 | Chrome profile 目录保存完整 session，**重启无需重新登录** |
| 多账号管理 | 每个账号独立 profile 目录，session 完全隔离，一键切换 |
| 广播发布 | 一次调用同时向多个账号发布相同内容 |
| 图文 / 视频 | 支持图文笔记和视频笔记发布 |
| 内容浏览 | 首页推荐、搜索、笔记详情、用户主页 |
| 互动操作 | 评论、回复、点赞、收藏 |

---

## 快速开始

### 1. 启动服务

```bash
# 无头模式（服务器/Docker 推荐）
./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome

# 本机调试（可见浏览器操作过程）
./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome -headless=false

# 使用代理
XHS_PROXY=http://user:pass@host:port ./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome
```

服务启动后监听 `http://localhost:18060/mcp`，注册 **20 个 MCP 工具**。

### 启动参数

| 参数 | 说明 | 默认值 |
|---|---|---|
| `-bin` | Chrome 可执行文件路径 | 自动寻找 |
| `-port` | 监听端口 | `:18060` |
| `-headless` | 是否无头模式 | `true` |
| `XHS_PROXY` | HTTP 代理地址 | 无 |
| `ACCOUNTS_PATH` | 账号注册表文件路径 | `accounts.json` |

### 2. 登录（手机号，推荐）

```
# 步骤 1：发送验证码（11 位手机号，不需要加 +86）
creator_phone_login(phone="13800000000")

# 步骤 2：填写 6 位验证码
creator_verify_otp(otp="123456")
→ 自动完成 creator + www 两个域的 session，写入 profile，重启不丢失

# 步骤 3：确认登录状态
check_login_status()
```

### 3. 发布内容

```
# 图文笔记
publish_content(
  title="标题",
  content="正文",
  images=["/绝对路径/image.jpg"],
  tags=["话题"],
  visibility="仅自己可见"
)

# 视频笔记
publish_with_video(
  title="标题",
  content="正文",
  video="/绝对路径/video.mp4"
)
```

### 4. 多账号管理

```
# 添加新账号（自动切换为激活）
add_account(name="副号")

# 用新账号登录
creator_phone_login(phone="副号手机号")
creator_verify_otp(otp="验证码")

# 切换账号
switch_account(name="default")   # 切回主号
switch_account(name="副号")      # 切到副号

# 查看所有账号
list_accounts()
```

### 5. 广播发布（多账号同时发布）

```
broadcast_publish(
  accounts=["default", "副号"],
  title="标题",
  content="正文",
  images=["/path/image.jpg"]
)
```

---

## MCP 工具清单（20 个）

### 登录管理
| 工具 | 说明 |
|---|---|
| `creator_phone_login` | 发送手机验证码 |
| `creator_verify_otp` | 填写验证码完成登录 |
| `check_login_status` | 检查当前账号登录状态 |
| `get_login_qrcode` | QR 码登录（可选） |
| `delete_cookies` | 重置当前账号登录状态 |

### 账号管理
| 工具 | 说明 |
|---|---|
| `list_accounts` | 列出所有账号及激活状态 |
| `add_account` | 添加账号 |
| `switch_account` | 切换激活账号 |
| `remove_account` | 删除账号及其 profile 数据 |

### 内容发布
| 工具 | 说明 |
|---|---|
| `publish_content` | 发布图文笔记 |
| `publish_with_video` | 发布视频笔记 |
| `broadcast_publish` | 广播发布到多账号 |

### 内容浏览
| 工具 | 说明 |
|---|---|
| `list_feeds` | 获取首页推荐列表 |
| `search_feeds` | 搜索内容 |
| `get_feed_detail` | 获取笔记详情+评论 |
| `user_profile` | 获取用户主页信息 |

### 互动操作
| 工具 | 说明 |
|---|---|
| `post_comment_to_feed` | 发表评论 |
| `reply_comment_in_feed` | 回复评论 |
| `like_feed` | 点赞 |
| `favorite_feed` | 收藏 |

---

## 详细使用说明

参见 [USAGE.md](./USAGE.md)。

---

## 常驻运行

**systemd**

```ini
[Unit]
Description=Xiaohongshu MCP Pro Server
After=network.target

[Service]
ExecStart=/path/to/xiaohongshu-mcp-pro -bin /usr/bin/google-chrome
WorkingDirectory=/path/to/workdir
Environment=ACCOUNTS_PATH=/path/to/workdir/accounts.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**后台进程**

```bash
nohup ./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome > /tmp/xhs-mcp.log 2>&1 &
```

---

## 致谢

本项目基于 [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp) 二次开发。感谢原项目作者 [@xpzouying](https://github.com/xpzouying) 及全体贡献者构建了完整的小红书 MCP 基础能力，本项目在此基础上新增了手机号登录、Chrome profile 持久化 session 以及多账号管理等功能。
