# xiaohongshu-mcp-pro

MCP Server for Xiaohongshu (RedNote). Publish content from a **headless server** (no desktop required), log in via **phone OTP** (no QR code scanning), and manage **multiple accounts** independently.

Built on top of [xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp). Thanks to the original author and all contributors for the excellent foundation.

---

## Key Features

| Feature | Description |
|---|---|
| Headless publishing | Runs fully headless on servers/Docker — no GUI required |
| Phone OTP login | Log in with SMS verification code — **no QR code scanning** |
| Persistent sessions | Chrome profile directory preserves full session across restarts |
| Multi-account | Each account has an isolated profile directory, switch with one command |
| Broadcast publish | Publish the same content to multiple accounts in one call |
| Image / Video | Supports both image posts and video posts |
| Browse | Home feed, search, note details, user profiles |
| Interact | Comment, reply, like, favorite |

---

## Quick Start

### 1. Start the server

```bash
# Headless mode (server/Docker)
./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome

# Debug mode (visible browser)
./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome -headless=false

# With proxy
XHS_PROXY=http://user:pass@host:port ./xiaohongshu-mcp-pro -bin /usr/bin/google-chrome
```

Listens on `http://localhost:18060/mcp` with **20 MCP tools**.

### Startup flags

| Flag | Description | Default |
|---|---|---|
| `-bin` | Chrome executable path | auto-detect |
| `-port` | Listen port | `:18060` |
| `-headless` | Headless mode | `true` |
| `XHS_PROXY` | HTTP proxy URL | none |
| `ACCOUNTS_PATH` | Account registry file | `accounts.json` |

### 2. Login (phone OTP, recommended)

```
# Step 1: Send OTP (11-digit number, no +86 prefix)
creator_phone_login(phone="13800000000")

# Step 2: Submit the 6-digit code
creator_verify_otp(otp="123456")
→ Writes both creator + www sessions into the Chrome profile. Survives restarts.

# Step 3: Verify login
check_login_status()
```

### 3. Publish content

```
# Image post
publish_content(
  title="Title",
  content="Body text",
  images=["/absolute/path/image.jpg"],
  tags=["topic"],
  visibility="仅自己可见"
)

# Video post
publish_with_video(
  title="Title",
  content="Body text",
  video="/absolute/path/video.mp4"
)
```

### 4. Multi-account management

```
# Add a new account (auto-activates)
add_account(name="account2")

# Log in with the new account
creator_phone_login(phone="phone_number")
creator_verify_otp(otp="code")

# Switch accounts
switch_account(name="default")    # back to main
switch_account(name="account2")   # to secondary

# List all accounts
list_accounts()
```

### 5. Broadcast publish (post to multiple accounts at once)

```
broadcast_publish(
  accounts=["default", "account2"],
  title="Title",
  content="Body",
  images=["/path/image.jpg"]
)
```

---

## All MCP Tools (20)

### Login
| Tool | Description |
|---|---|
| `creator_phone_login` | Send phone OTP |
| `creator_verify_otp` | Submit OTP to complete login |
| `check_login_status` | Check current account login status |
| `get_login_qrcode` | QR code login (alternative) |
| `delete_cookies` | Reset current account session |

### Account Management
| Tool | Description |
|---|---|
| `list_accounts` | List all accounts and active status |
| `add_account` | Add account |
| `switch_account` | Switch active account |
| `remove_account` | Delete account and its profile data |

### Publishing
| Tool | Description |
|---|---|
| `publish_content` | Publish image post |
| `publish_with_video` | Publish video post |
| `broadcast_publish` | Publish to multiple accounts at once |

### Browsing
| Tool | Description |
|---|---|
| `list_feeds` | Get home feed recommendations |
| `search_feeds` | Search content |
| `get_feed_detail` | Get note details + comments |
| `user_profile` | Get user profile page |

### Interaction
| Tool | Description |
|---|---|
| `post_comment_to_feed` | Post a comment |
| `reply_comment_in_feed` | Reply to a comment |
| `like_feed` | Like a note |
| `favorite_feed` | Favorite a note |

---

## Detailed Usage

See [USAGE.md](./USAGE.md).

---

## Acknowledgements

This project is a fork of [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp). Many thanks to [@xpzouying](https://github.com/xpzouying) and all contributors who built the solid MCP foundation this project builds upon. New additions include phone OTP login, Chrome profile-based session persistence, and multi-account management.
