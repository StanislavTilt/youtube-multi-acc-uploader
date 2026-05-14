./youtube-uploader.exe serve# YouTube Shorts Uploader

Multi-account YouTube Shorts uploader with web admin panel, scheduled uploads, parallel publishing, and automatic video uniquification.

---

## Requirements

| Software | Version | Required |
|----------|---------|----------|
| **Go** | 1.22+ | Yes |
| **FFmpeg** | 6.0+ | Yes (for video uniquification) |
| **Google Cloud Project** | - | Yes (YouTube Data API v3) |

---

## Installation & Setup

### Step 1: Install Go

Download from [go.dev/dl](https://go.dev/dl/) and install. Verify:

```bash
go version
# go version go1.22+ ...
```

### Step 2: Install FFmpeg

FFmpeg is required for automatic video uniquification before upload.

**Windows (winget):**
```powershell
winget install Gyan.FFmpeg
```

**Windows (scoop):**
```powershell
scoop install ffmpeg
```

**Windows (manual):**
1. Download from [ffmpeg.org/download.html](https://ffmpeg.org/download.html)
2. Extract to `C:\ffmpeg\`
3. Add `C:\ffmpeg\bin` to system PATH

**macOS:**
```bash
brew install ffmpeg
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt install ffmpeg
```

After installation, **restart your terminal** and verify:
```bash
ffmpeg -version
```

> If `ffmpeg` is not found after install via winget, add it to PATH manually:
> ```powershell
> # PowerShell:
> $env:PATH += ";C:\ProgramData\winget\Links"
>
> # CMD:
> set PATH=%PATH%;C:\ProgramData\winget\Links
>
> # Git Bash:
> export PATH="$PATH:/c/ProgramData/winget/Links"
> ```

### Step 3: Google Cloud Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or use existing)
3. **Enable YouTube Data API v3:**
   - APIs & Services → Library → search "YouTube Data API v3" → **Enable**
4. **Create OAuth credentials:**
   - APIs & Services → Credentials → Create Credentials → OAuth client ID
   - Application type: **Web application**
   - Authorized redirect URIs: `http://localhost:3000/callback`
   - Copy **Client ID** and **Client Secret**
5. **Configure OAuth consent screen:**
   - APIs & Services → OAuth consent screen
   - Add test users (your Gmail addresses that have YouTube channels)

### Step 4: Configure the Project

```bash
cd youtube-uploader

# Install Go dependencies
go mod tidy

# Copy example config
cp .env.example .env

# Edit .env — add your Google OAuth credentials
```

`.env` file:
```env
# Server
PORT=3000
HOST=0.0.0.0

# Database
DB_PATH=./data.db

# Google OAuth2 (single app for all accounts)
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret

# Directories
VIDEOS_PENDING=./videos/pending
VIDEOS_UPLOADED=./videos/uploaded

# Upload concurrency
MAX_CONCURRENCY=3

# Logging
LOG_LEVEL=info
```

### Step 5: Build & Run

```bash
# Build the binary
go build -o youtube-uploader.exe .

# Start the web panel
./youtube-uploader.exe serve
```

Open **http://localhost:3000** in your browser.

### Restarting After Changes

```bash
# If you changed Go code (backend):
taskkill /F /IM youtube-uploader.exe
go build -o youtube-uploader.exe .
./youtube-uploader.exe serve

# If you changed only HTML/CSS/JS (frontend):
# Just refresh the browser (Ctrl+F5), no restart needed
```

---

## Web Admin Panel

The admin panel provides a full GUI for managing accounts, uploads, and settings.

### Pages

| Page | Description |
|------|-------------|
| **Dashboard** | Overview stats: accounts, uploads, failures, schedules |
| **Accounts** | Add, authorize, delete, restore YouTube accounts |
| **New Upload** | Upload videos with title, description, tags, privacy, scheduling |
| **Uploads** | Upload history with status, YouTube links, delete option |
| **Presets** | Video metadata templates |
| **Schedules** | Cron-based automated upload schedules |
| **Logs** | Real-time JSON logs (split view: list + detail panel) |

### Adding Accounts (Web)

**Option A: One-click Google OAuth (recommended):**
1. Click **"+ Add via Google"**
2. Sign into your Google account
3. Grant YouTube permissions
4. Account is created and authorized automatically

**Option B: Manual credentials:**
1. Click **"+ Manual (ID/Secret)"**
2. Enter per-account Client ID and Client Secret
3. Click **"Authorize"** button next to the account

**Authorize All:**
- Click **"Authorize All"** to open OAuth windows for all unauthorized accounts

### Uploading Videos (Web)

1. Go to **New Upload** page
2. Drag & drop a video or select from `videos/pending/`
3. Fill in: **Title**, **Description**, **Tags** (click quick-tag buttons or type comma-separated)
4. Select **Privacy** (public / unlisted / private)
5. Choose **Upload Now** or **Schedule for Later** (pick date & time)
6. Select target accounts (Select All / Deselect All)
7. Click **"Upload to YouTube"**

After successful upload, the video moves from `videos/pending/` to `videos/uploaded/`.

### Quick Tags

- Pre-loaded tags appear as clickable buttons under the Tags field
- Click to add/remove tags instantly
- Add custom tags with the input + "Add" button
- Tags track usage count (most used appear first)
- Delete tags with the x button on hover

---

## Video Uniquification

Each video is automatically processed with FFmpeg before upload to make every copy visually unique per account. This helps avoid duplicate content detection.

### How It Works

Before uploading to each YouTube account, a unique copy is created with random subtle adjustments:

| Parameter | Range | Effect |
|-----------|-------|--------|
| Brightness | -0.04 .. +0.04 | Slight brightness shift |
| Contrast | 0.95 .. 1.05 | Subtle contrast adjustment |
| Saturation | 0.92 .. 1.08 | Minor color saturation change |
| Gamma | 0.94 .. 1.06 | Gamma correction |
| Hue | -3 .. +3 degrees | Imperceptible hue shift |
| Noise | 2-6 | Very faint random noise |

**What stays the same:**
- Video duration (unchanged)
- Audio (copied without modification)
- Resolution and aspect ratio (9:16)

**What changes:**
- Each account gets its own unique visual fingerprint
- Changes are subtle enough to be invisible to viewers
- Temporary files are auto-deleted after upload

**Fallback:** If FFmpeg is not installed, videos upload without uniquification (original file).

---

## CLI Commands

### Account Management

```bash
# Import accounts from file (format: client_id:client_secret per line)
./youtube-uploader.exe import --file accounts.txt

# List all accounts with auth status
./youtube-uploader.exe account list

# Soft delete (can be restored)
./youtube-uploader.exe account delete <id>

# Restore soft-deleted account
./youtube-uploader.exe account restore <id>
```

### Authorization

```bash
# Authorize a specific account (opens browser)
./youtube-uploader.exe auth --account <id>

# Authorize all unauthorized accounts
./youtube-uploader.exe auth --account all
```

This opens a browser window for Google OAuth2. After granting access, the token is saved to the database.

### Uploading Videos

```bash
# Upload all videos from videos/pending/ to all accounts
./youtube-uploader.exe upload --title "My Short" --description "Check this out" --tags "shorts,viral,funny"

# Upload a specific file
./youtube-uploader.exe upload --video ./videos/pending/my-video.mp4 --title "My Short"

# Upload using a preset
./youtube-uploader.exe upload --preset gaming
```

### Presets

```bash
# Create a preset
./youtube-uploader.exe preset create \
  --name gaming \
  --title "{filename}" \
  --description "Gaming content" \
  --tags "gaming,shorts,viral" \
  --category 20 \
  --privacy public

# List all presets
./youtube-uploader.exe preset list
```

**YouTube Category IDs:** 1=Film, 2=Cars, 10=Music, 15=Pets, 17=Sports, 20=Gaming, 22=People, 23=Comedy, 24=Entertainment, 25=News, 26=Howto, 28=Science

### Scheduling

```bash
# Upload every day at 9:00 AM
./youtube-uploader.exe schedule add --cron "0 9 * * *" --preset gaming

# Upload twice a day (12:00 and 18:00)
./youtube-uploader.exe schedule add --cron "0 12,18 * * *" --title "Daily Short"

# Upload weekdays only at 10:00 AM
./youtube-uploader.exe schedule add --cron "0 10 * * 1-5" --title "Weekday Short"

# List schedules
./youtube-uploader.exe schedule list

# Disable/enable a schedule
./youtube-uploader.exe schedule disable <id>
./youtube-uploader.exe schedule enable <id>
```

### Scheduler Daemon

```bash
# Start the background scheduler (checks every 60 seconds)
./youtube-uploader.exe daemon
```

The daemon picks the next video from `videos/pending/`, uniquifies and uploads it to all accounts, then moves it to `videos/uploaded/`.

### Web Panel Server

```bash
# Start on default port (3000)
./youtube-uploader.exe serve

# Start on custom port
./youtube-uploader.exe serve --port 8080
```

---

## Logs

Logs are stored in the `logs/` directory with daily rotation and JSON format. Two separate streams:

### Log Files

```
logs/
├── admin-2026-04-04.json     # HTTP requests to admin panel
├── admin-2026-04-05.json
├── youtube-2026-04-04.json   # YouTube API: uploads, auth, errors
└── youtube-2026-04-05.json
```

### Admin Log Entry

```json
{
  "timestamp": "2026-04-04T20:14:03+02:00",
  "level": "info",
  "category": "admin",
  "method": "POST",
  "path": "/api/upload",
  "status": 202,
  "duration": "1.2ms",
  "ip": "127.0.0.1",
  "userAgent": "Mozilla/5.0..."
}
```

### YouTube Log Entry

```json
{
  "timestamp": "2026-04-04T20:14:06+02:00",
  "level": "info",
  "category": "youtube",
  "message": "Upload success",
  "accountId": 1,
  "videoFile": "my-video.mp4",
  "youtubeId": "gCAFOcxGDsU"
}
```

### Viewing Logs

| Method | Command |
|--------|---------|
| **Web panel** | Logs tab → select Admin/YouTube → filter by level → click entry for full detail |
| **CLI (today)** | `cat logs/youtube-$(date +%Y-%m-%d).json` |
| **CLI (real-time)** | `tail -f logs/youtube-$(date +%Y-%m-%d).json` |
| **CLI (search errors)** | `grep '"error"' logs/youtube-*.json` |

---

## Project Structure

```
youtube-uploader/
├── .env                          # Configuration (not in git)
├── .env.example                  # Config template
├── go.mod / go.sum               # Go dependencies
├── main.go                       # CLI entry point
├── accounts.txt                  # Import file (client_id:client_secret)
├── videos/
│   ├── pending/                  # Place videos here for upload
│   └── uploaded/                 # Auto-moved after successful upload
├── logs/                         # JSON logs (daily rotation)
├── migrations/                   # SQLite schema migrations
│   ├── 001_init.sql              # Core tables
│   ├── 002_quick_tags.sql        # Quick tags with defaults
│   └── 003_app_oauth.sql         # App-level OAuth support
├── cmd/
│   ├── serve.go                  # Web server + API handlers
│   ├── auth.go                   # OAuth2 CLI flow
│   ├── upload.go                 # CLI upload command
│   ├── import.go                 # Account import from file
│   ├── account.go                # Account CRUD CLI
│   ├── preset.go                 # Preset management CLI
│   ├── schedule.go               # Schedule management CLI
│   └── daemon.go                 # Background scheduler
├── internal/
│   ├── config/config.go          # Environment config loader
│   ├── db/
│   │   ├── db.go                 # SQLite connection + migrations
│   │   └── models.go             # Data models + CRUD operations
│   ├── youtube/
│   │   ├── auth.go               # OAuth2 (app-level + per-account)
│   │   └── upload.go             # YouTube upload + file management
│   ├── uniquify/uniquify.go      # FFmpeg video uniquification
│   ├── logger/logger.go          # Dual-stream JSON logger
│   ├── scheduler/scheduler.go    # Cron expression parser
│   └── web/static/index.html     # Admin panel (dark theme)
```

---

## Troubleshooting

### `YouTube Data API v3 has not been used in project`
Enable the API: [Google Cloud Console -> APIs -> YouTube Data API v3 -> Enable](https://console.cloud.google.com/apis/library/youtube.googleapis.com)

### `access_denied` during OAuth
Add your email to test users: Google Cloud Console -> OAuth consent screen -> Audience -> Add users

### `redirect_uri_mismatch`
Add `http://localhost:3000/callback` to: Google Cloud Console -> Credentials -> your OAuth client -> Authorized redirect URIs

### `youtubeSignupRequired`
The Google account doesn't have a YouTube channel. Create one at [YouTube Studio](https://studio.youtube.com).

### `ffmpeg not found`
Install FFmpeg (see Installation step 2). After install, restart your terminal. Verify with `ffmpeg -version`. If still not found, add to PATH manually.

### Only 3 hashtags visible on Shorts
YouTube displays **max 3 hashtags** as clickable links under the title. All tags are in the description (visible when expanded) and in video metadata for search/SEO. Put the most important tags first.

### Video not moving to `uploaded/`
The video only moves when upload succeeds on **all** selected accounts. If any account fails, the file stays in `pending/`.

### Upload works but video not unique
Ensure FFmpeg is installed and in PATH. Check logs for "uniquify skipped" messages. The app falls back to uploading the original if FFmpeg is unavailable.
