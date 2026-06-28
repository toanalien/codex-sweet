# Codex Sweet 🍬

[![Build](https://github.com/toanalien/codex-sweet/actions/workflows/build.yml/badge.svg)](https://github.com/toanalien/codex-sweet/actions/workflows/build.yml)
[![Release](https://github.com/toanalien/codex-sweet/actions/workflows/release.yml/badge.svg)](https://github.com/toanalien/codex-sweet/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

CLI tool để quản lý nhiều tài khoản Codex authentication profiles.

![Demo](assets/demo.jpg)

## Tính năng

- ✅ Lưu nhiều Codex credentials từ `~/.codex/auth.json`
- ✅ Swap nhanh giữa các tài khoản bằng `switch` hoặc `use`
- ✅ Quản lý profiles: save, switch/use, list, info, delete
- ✅ Xem usage và limits (5h, weekly) qua ChatGPT backend API
- ✅ Chạy `codex-sweet` để xem current account và usage của tất cả profiles
- ✅ **Smart available check** - Tự động tìm profiles còn limit
- ✅ **Batch usage view** - Xem usage của tất cả profiles cùng lúc
- ✅ Bash completion qua `codex-sweet completion bash`
- ✅ Progress bar trực quan với % còn lại
- ✅ **Auto-switch daemon** - Tự động rotate profiles khi hết quota (10 phút/lần)
- ✅ **Notifications** - Gửi thông báo qua Telegram, Discord, Slack,... khi quota thấp, hết limit, auto-switch (via [shoutrrr](https://github.com/containrrr/shoutrrr))

## Cài đặt

### Download Pre-built Binaries (Recommended)

Download latest release từ [GitHub Releases](https://github.com/toanalien/codex-sweet/releases):

```bash
# Linux AMD64
wget https://github.com/toanalien/codex-sweet/releases/latest/download/codex-sweet-linux-amd64.tar.gz
tar xzf codex-sweet-linux-amd64.tar.gz
sudo mv codex-sweet-linux-amd64 /usr/local/bin/codex-sweet
chmod +x /usr/local/bin/codex-sweet

# Linux ARM64
wget https://github.com/toanalien/codex-sweet/releases/latest/download/codex-sweet-linux-arm64.tar.gz
tar xzf codex-sweet-linux-arm64.tar.gz
sudo mv codex-sweet-linux-arm64 /usr/local/bin/codex-sweet
chmod +x /usr/local/bin/codex-sweet

# macOS Intel (AMD64)
wget https://github.com/toanalien/codex-sweet/releases/latest/download/codex-sweet-darwin-amd64.tar.gz
tar xzf codex-sweet-darwin-amd64.tar.gz
sudo mv codex-sweet-darwin-amd64 /usr/local/bin/codex-sweet
chmod +x /usr/local/bin/codex-sweet

# macOS Apple Silicon (ARM64/M1/M2/M3)
wget https://github.com/toanalien/codex-sweet/releases/latest/download/codex-sweet-darwin-arm64.tar.gz
tar xzf codex-sweet-darwin-arm64.tar.gz
sudo mv codex-sweet-darwin-arm64 /usr/local/bin/codex-sweet
chmod +x /usr/local/bin/codex-sweet
```

**Verify checksum**:
```bash
sha256sum -c codex-sweet-linux-amd64.tar.gz.sha256
```

**macOS Users**: Remove quarantine attribute after download:
```bash
xattr -d com.apple.quarantine /usr/local/bin/codex-sweet
# Or use: xattr -c /usr/local/bin/codex-sweet
```

Hoặc allow qua **System Settings → Privacy & Security → Open Anyway**

### Build from Source

Requires Go 1.22+:

```bash
git clone https://github.com/toanalien/codex-sweet.git
cd codex-sweet
go build -o codex-sweet
sudo mv codex-sweet /usr/local/bin/
```

## Quick Start

### 1. Setup profiles

```bash
# Login tài khoản 1
codex auth login --device-auth
codex-sweet save
# ✓ Saved profile 'work@company.com'

# Login tài khoản 2
codex auth login --device-auth
codex-sweet save
# ✓ Saved profile 'personal@gmail.com'

# Login tài khoản 3
codex auth login --device-auth
codex-sweet save
# ✓ Saved profile 'hobby@outlook.com'

# Nếu login lại tài khoản đã có
codex auth login --device-auth
codex-sweet save
# ⚠️  Profile already exists: 'work@company.com'
# 💡 Tip: Use 'codex-sweet switch work@company.com' to activate it
```

### 2. Smart workflow

```bash
# Xem current account và usage tất cả profiles
codex-sweet

# Nếu chỉ muốn xem profiles còn limit
codex-sweet available

# Switch/use sang profile còn nhiều limit nhất
codex-sweet switch personal@gmail.com
# Hoặc:
codex-sweet use personal@gmail.com

# Bắt đầu code!
codex chat "help me implement..."
```

### 3. Monitor usage

```bash
# Xem usage của tất cả profiles
codex-sweet usage
```

## Sử dụng chi tiết

### 1. Lưu profile mới (Auto-named by email)

```bash
# Login với Codex CLI
codex auth login --device-auth

# Lưu profile (tự động dùng email làm tên)
codex-sweet save
# ✓ Saved profile 'your@email.com'

# Profile được tự động đặt tên theo email
# - Tránh trùng lặp
# - Dễ nhận biết
# - Tự động phát hiện email từ credentials
```

**Note**:
- ✅ Profiles được đặt tên tự động theo email
- ✅ Tự động check duplicate - nếu email đã tồn tại sẽ bỏ qua
- ✅ Không cần nhập tên profile thủ công

### 2. Switch/use profile

```bash
codex-sweet switch personal@gmail.com
# Alias dễ nhớ hơn:
codex-sweet use personal@gmail.com
```

File `~/.codex/auth.json` sẽ được cập nhật với credentials của profile "personal@gmail.com".

### 3. Liệt kê tất cả profiles

```bash
codex-sweet list
```

Output:
```
Saved profiles:
───────────────────────────────────────────────
● personal@gmail.com (created: 2026-03-19 10:30)
  work@company.com (created: 2026-03-19 09:15)
  hobby@outlook.com (created: 2026-03-19 08:00)
```

### 4. Xem current account và usage tất cả profiles (⭐ RECOMMENDED)

```bash
# Chạy không tham số để xem current account và usage
codex-sweet
```

Output:
```
Current account: personal@gmail.com

📊 work - work@email.com (plus)
───────────────────────────────────────────────────────────
5h limit:        [███████████████     ]  77% left (resets 16:30)
Weekly limit:    [█████████████       ]  68% left (resets 15:30 on 26 Mar)
```

### 5. Kiểm tra profiles có limit còn trống

```bash
# Dùng command available để xem profiles còn limit
codex-sweet available
```

Output:
```
🔍 Checking available profiles...

● work@company.com - 5h: 77% left, Weekly: 68% left
  personal@gmail.com - 5h: 95% left, Weekly: 88% left

✓ Found 2 profile(s) with available limits
```

### 6. Xem usage tất cả profiles

```bash
# Xem usage của tất cả profiles, không in dòng Current account
codex-sweet usage
```

Output:
```
📊 work - work@email.com (plus)
───────────────────────────────────────────────────────────
5h limit:        [███████████████     ]  77% left (resets 16:30)
Weekly limit:    [█████████████       ]  68% left (resets 15:30 on 26 Mar)

📊 personal - personal@email.com (plus)
───────────────────────────────────────────────────────────
5h limit:        [███████████████████ ]  95% left (resets 17:15)
Weekly limit:    [█████████████████   ]  88% left (resets 16:20 on 26 Mar)
```

### 7. Bash completion

```bash
# Load completion cho shell hiện tại
source <(codex-sweet completion bash)

# Cài lâu dài cho user hiện tại
mkdir -p ~/.local/share/bash-completion/completions
codex-sweet completion bash > ~/.local/share/bash-completion/completions/codex-sweet
```

Sau khi mở shell mới, Bash có thể autocomplete commands và profile names cho các command như `switch`, `use`, `usage`, `info`, và `delete`.

### 8. Xem thông tin chi tiết profile

```bash
codex-sweet info work
```

Output:
```
📋 Profile: work
───────────────────────────────────────────────
Created:     2026-03-19 10:30:00
Active:      true
Auth Mode:   chatgpt
Access Token: eyJhbGci...Bhic
Account ID:  f817b565-85c6-44db-9d54-f3f61d36c111
Last Refresh: 2026-03-19T09:37:00Z
```

### 9. Xem usage 1 profile cụ thể

```bash
codex-sweet usage work@company.com
```

### 10. Xem chi tiết profile

```bash
codex-sweet info work@company.com
```

### 11. Xóa profile

```bash
codex-sweet delete old@email.com
```

## 🎯 Workflow thông minh (Best Practice)

### Scenario 1: Bắt đầu ngày làm việc

```bash
# Morning check - Xem current account và usage
codex-sweet

# Output:
# Current account: work@company.com
# 📊 work@company.com - work@company.com (plus)
# 5h limit:        [████████████████████] 100% left (resets 16:30)

# Nếu chỉ muốn danh sách profiles còn limit
codex-sweet available

# Switch sang profile tốt nhất
codex-sweet switch work@company.com

# Start coding!
codex chat "review my code"
```

### Scenario 2: Đang code bị limit

```bash
# Khi gặp rate limit error
codex-sweet available  # Quick check available profiles

# Switch sang profile còn limit
codex-sweet use personal@gmail.com

# Continue coding immediately
codex chat "continue implementation"
```

### Scenario 3: Monitor toàn bộ accounts

```bash
# Xem usage tất cả profiles để plan
codex-sweet usage

# Output hiển thị:
# - work@company.com: 77% left 5h, 68% left weekly
# - personal@gmail.com: 95% left 5h, 88% left weekly
# - hobby@outlook.com: 45% left 5h, 52% left weekly

# Decision: Dùng personal cho task nặng, giữ work cho emergencies
```

### Scenario 4: Auto-switch daemon (⭐ NEW)

Run daemon để tự động rotate profiles khi hết quota:

```bash
# Start daemon (runs in foreground with status logs)
codex-sweet auto

# Output:
# 🤖 Starting auto-switch daemon...
# ⏰ Check interval: 10m0s
# 📊 Quota threshold: 20%
#
# [10:30:00] 🔍 Checking profiles...
# 📊 Current: work@company.com (5h: 45%, Weekly: 52%)
# ⚠️  Current profile quota low, finding alternative...
# 🔄 Switched: work@company.com → personal@gmail.com (5h: 95%, Weekly: 88%)
```

Daemon sẽ:
- ✅ Check quota mỗi 10 phút
- ✅ Cache quota trong 5 phút để giảm API calls
- ✅ Auto-switch khi quota < 20%
- ✅ Random check order để phân tán load
- ✅ Log tất cả switches vào `~/.codex-sweet/log.json`
- ✅ Graceful shutdown với Ctrl+C

**Chạy daemon dưới nền:**
```bash
# Run in background with nohup
nohup codex-sweet auto > /dev/null 2>&1 &

# Or with systemd (recommended)
# Create ~/.config/systemd/user/codex-sweet.service:
[Unit]
Description=Codex Sweet Auto-Switch Daemon
After=network.target

[Service]
ExecStart=/usr/local/bin/codex-sweet auto
Restart=always

[Install]
WantedBy=default.target

# Enable and start
systemctl --user enable codex-sweet
systemctl --user start codex-sweet
systemctl --user status codex-sweet
```

**Xem switch logs:**
```bash
cat ~/.codex-sweet/log.json
```

### Scenario 5: Notifications (⭐ NEW)

Nhận thông báo qua Telegram, Discord, Slack,... khi có sự kiện quota:

```bash
# Thêm Telegram notification
codex-sweet notify add "telegram://botToken@telegram?chats=chatID"

# Thêm Discord notification
codex-sweet notify add "discord://token@channelID"

# Thêm Slack notification
codex-sweet notify add "slack://token-a/token-b/token-c"

# Xem danh sách notification URLs
codex-sweet notify list

# Gửi test notification
codex-sweet notify test

# Xóa notification URL theo index
codex-sweet notify remove 0
```

**Các sự kiện được thông báo:**
- **quota_low** - Khi quota của profile hiện tại xuống thấp (< 20%)
- **limit_reached** - Khi profile bị rate limit
- **auto_switch** - Khi daemon tự động chuyển profile
- **all_exhausted** - Khi tất cả profiles đều hết quota

**Supported services** ([shoutrrr](https://github.com/containrrr/shoutrrr)):
Telegram, Discord, Slack, Email (SMTP), Ntfy, Pushover, Gotify, Teams, Google Chat, Matrix, Mattermost, Pushbullet, Rocket.Chat, và nhiều hơn nữa.

Config được lưu tại `~/.codex-sweet/notify.json` với permission `0600`.

## 💡 Tips & Tricks

1. **Alias cho workflow nhanh**:
   ```bash
   # Thêm vào ~/.bashrc hoặc ~/.zshrc
   alias cs='codex-sweet'
   alias csa='codex-sweet available'
   alias csu='codex-sweet usage'

   # Usage:
   cs                              # View current account + usage
   csa                             # Check available profiles
   csu                             # View all usage
   cs use personal@gmail.com       # Use/switch profile
   ```

2. **Pre-commit hook** - Check limit trước khi code:
   ```bash
   # .git/hooks/pre-commit
   #!/bin/bash
   codex-sweet available > /dev/null 2>&1
   if [ $? -ne 0 ]; then
       echo "⚠️  Warning: No Codex profiles with available limits"
   fi
   ```

3. **Status bar integration**:
   ```bash
   # Thêm vào tmux status bar
   set -g status-right '#(codex-sweet available | head -1 | cut -d" " -f2-)'
   ```

## 📊 Commands Summary

| Command | Description | Example |
|---------|-------------|---------|
| `codex-sweet` | Xem current account và usage tất cả profiles (default) | `codex-sweet` |
| `codex-sweet save` | Lưu profile mới (auto-named by email) | `codex-sweet save` |
| `codex-sweet switch <email>` | Switch sang profile khác | `codex-sweet switch work@company.com` |
| `codex-sweet use <email>` | Alias của `switch` để dùng profile khác | `codex-sweet use work@company.com` |
| `codex-sweet list` | List tất cả profiles | `codex-sweet list` |
| `codex-sweet available` | Xem profiles còn limit | `codex-sweet available` |
| `codex-sweet usage` | Xem usage tất cả profiles | `codex-sweet usage` |
| `codex-sweet usage <email>` | Xem usage 1 profile | `codex-sweet usage work@company.com` |
| `codex-sweet info <email>` | Xem chi tiết profile | `codex-sweet info work@company.com` |
| `codex-sweet delete <email>` | Xóa profile | `codex-sweet delete old@email.com` |
| `codex-sweet auto` | Auto-switch daemon (10min interval) | `codex-sweet auto` |
| `codex-sweet notify add <url>` | Thêm notification URL | `codex-sweet notify add "telegram://bot:token@telegram?chats=123"` |
| `codex-sweet notify remove <index>` | Xóa notification URL | `codex-sweet notify remove 0` |
| `codex-sweet notify list` | Xem notification URLs | `codex-sweet notify list` |
| `codex-sweet notify test` | Gửi test notification | `codex-sweet notify test` |
| `codex-sweet completion bash` | Xuất bash completion script | `source <(codex-sweet completion bash)` |

## 📋 Profile Structure Details

### Complete Profile Example

File: `~/.codex-sweet/profiles.json`

```json
{
  "profiles": {
    "work@company.com": {
      "name": "work@company.com",
      "auth": {
        "auth_mode": "chatgpt",
        "OPENAI_API_KEY": null,
        "tokens": {
          "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1LWJiYzktNDRkMS1hOWQwLWY5NTdiMDc5YmQwZSIsInR5cCI6IkpXVCJ9...",
          "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1LWJiYzktNDRkMS1hOWQwLWY5NTdiMDc5YmQwZSIsInR5cCI6IkpXVCJ9...",
          "refresh_token": "rt_wtLmg-eMMo1rfKicQGDL8ucum2ucPoSVz3BLYUxZAng...",
          "account_id": "f817b565-85c6-44db-9d54-f3f61d36c111"
        },
        "last_refresh": "2026-03-19T10:30:00Z"
      },
      "created_at": "2026-03-19T10:30:00Z",
      "active": true
    },
    "personal@gmail.com": {
      "name": "personal@gmail.com",
      "auth": {
        "auth_mode": "chatgpt",
        "OPENAI_API_KEY": null,
        "tokens": {
          "id_token": "eyJ...",
          "access_token": "eyJ...",
          "refresh_token": "rt_...",
          "account_id": "4f196405-c07b-47e6-9daa-52ae04ba6dcb"
        },
        "last_refresh": "2026-03-19T09:15:00Z"
      },
      "created_at": "2026-03-19T09:15:00Z",
      "active": false
    }
  },
  "current": "work@company.com"
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `profiles` | Object | Container cho tất cả profiles |
| `profiles.<email>` | Object | Profile của từng tài khoản (keyed by email) |
| `name` | String | Tên profile (email) |
| `auth.auth_mode` | String | Loại authentication (`chatgpt` hoặc `api_key`) |
| `auth.OPENAI_API_KEY` | String/null | API key nếu dùng API auth mode |
| `auth.tokens.id_token` | String | JWT ID token (chứa email, user info) |
| `auth.tokens.access_token` | String | Bearer token để gọi API |
| `auth.tokens.refresh_token` | String | Token để refresh access token |
| `auth.tokens.account_id` | String | ChatGPT account ID |
| `auth.last_refresh` | String | Timestamp lần refresh cuối |
| `created_at` | String | Timestamp khi tạo profile |
| `active` | Boolean | Profile đang được sử dụng? |
| `current` | String | Email của profile hiện tại |

## 🔌 API Endpoints

Tool này gọi ChatGPT backend API để lấy thông tin usage:

### Usage API

```
GET https://chatgpt.com/backend-api/wham/usage
```

**Headers**:
```
Authorization: Bearer {access_token}
chatgpt-account-id: {account_id}
User-Agent: codex-sweet/0.1.0
Accept: */*
Host: chatgpt.com
```

**Response**:
```json
{
  "user_id": "user-V1CeOCidog72akxH3v6LkQZk",
  "account_id": "user-V1CeOCidog72akxH3v6LkQZk",
  "email": "your@email.com",
  "plan_type": "plus",
  "rate_limit": {
    "allowed": true,
    "limit_reached": false,
    "primary_window": {
      "used_percent": 23,
      "limit_window_seconds": 18000,
      "reset_after_seconds": 12243,
      "reset_at": 1773928295
    },
    "secondary_window": {
      "used_percent": 32,
      "limit_window_seconds": 604800,
      "reset_after_seconds": 580944,
      "reset_at": 1774496996
    }
  },
  "credits": {
    "has_credits": false,
    "unlimited": false,
    "balance": "0"
  }
}
```

**Fields**:
- `primary_window`: 5 giờ limit
- `secondary_window`: 7 ngày (weekly) limit
- `used_percent`: % đã sử dụng (0-100)
- `reset_at`: Unix timestamp khi reset
- `reset_after_seconds`: Seconds còn lại đến khi reset

## 📂 File Locations

| File | Path | Permission | Description |
|------|------|------------|-------------|
| **Profiles** | `~/.codex-sweet/profiles.json` | `0600` | Lưu tất cả profiles và credentials |
| **Codex Auth** | `~/.codex/auth.json` | `0600` | Credentials hiện tại của Codex CLI |
| **Auto Logs** | `~/.codex-sweet/log.json` | `0600` | Auto-switch daemon log history (last 100 entries) |
| **Auto State** | `~/.codex-sweet/state.json` | `0600` | Daemon state with quota cache |
| **Notify Config** | `~/.codex-sweet/notify.json` | `0600` | Notification URLs and event settings |

### Profiles Structure

File `~/.codex-sweet/profiles.json`:
```json
{
  "profiles": {
    "work@company.com": { ... },
    "personal@gmail.com": { ... }
  },
  "current": "work@company.com"
}
```

### Codex Auth Structure

File `~/.codex/auth.json` (managed by Codex CLI):
```json
{
  "auth_mode": "chatgpt",
  "OPENAI_API_KEY": null,
  "tokens": {
    "id_token": "eyJ...",
    "access_token": "eyJ...",
    "refresh_token": "rt_...",
    "account_id": "user-..."
  },
  "last_refresh": "2026-03-19T10:30:00Z"
}
```

## 🔒 Lưu ý bảo mật

⚠️ **IMPORTANT**: Cả 2 files đều chứa credentials nhạy cảm:
- `~/.codex-sweet/profiles.json` - Chứa tất cả access tokens
- `~/.codex/auth.json` - Chứa current access token

**Security measures**:
- ✅ Auto-set permission `0600` (chỉ owner đọc/ghi)
- ✅ Không commit vào git (đã thêm vào `.gitignore`)
- ✅ Không share files này với ai
- ✅ Nên backup encrypted nếu cần

**Nếu bị leak**:
```bash
# 1. Revoke all sessions
codex auth logout

# 2. Xóa profiles cũ
rm -rf ~/.codex-sweet/

# 3. Login lại và save profiles mới
codex auth login --device-auth
codex-sweet save
```

## 🔧 Troubleshooting

### macOS: "Apple could not verify codex-sweet is free of malware"

Binary chưa được Apple sign. Fix bằng 1 trong 3 cách:

**Option 1: Remove quarantine (Khuyến nghị)**
```bash
xattr -d com.apple.quarantine /usr/local/bin/codex-sweet
```

**Option 2: Clear all attributes**
```bash
xattr -c /usr/local/bin/codex-sweet
```

**Option 3: Allow qua System Settings**
1. Try run `codex-sweet`
2. Click "Cancel" khi popup xuất hiện
3. Mở **System Settings → Privacy & Security**
4. Scroll xuống, click **"Open Anyway"**
5. Confirm với password

**Verify đã fix**:
```bash
codex-sweet --help
# Should work without popup
```

### Linux: Permission denied

```bash
chmod +x /usr/local/bin/codex-sweet
```

### Command not found

Check if binary in PATH:
```bash
echo $PATH | grep -o '/usr/local/bin'

# If empty, add to PATH
export PATH="/usr/local/bin:$PATH"

# Add to ~/.bashrc or ~/.zshrc để permanent
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
```

## Tham khảo

- [Codex Authentication](https://developers.openai.com/codex/auth)
- [Codex CLI Reference](https://developers.openai.com/codex/cli/reference)
- [OpenAI API Pricing](https://developers.openai.com/api/docs/pricing)
- [Codex Pricing Guide](https://developers.openai.com/codex/pricing)
