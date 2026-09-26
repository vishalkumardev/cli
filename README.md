# BuildShare CLI

Official command-line tool for **BuildShare** — upload, manage, and distribute mobile app builds (APK / IPA) directly from your terminal or CI/CD pipelines.

---

### Web Download Portal

Visit the interactive download page to get direct binary downloads for all operating systems and architectures:
👉 **[https://vishalkumardev.github.io/cli/](https://vishalkumardev.github.io/cli/)**

---

## Installation

### macOS / Linux

Install via the official automated shell script:

```bash
curl -fsSL https://vishalkumardev.github.io/cli/install.sh | sh
```

Verify your installation:

```bash
buildshare --version
```

### Windows

Install via PowerShell:

```powershell
irm https://vishalkumardev.github.io/cli/install.ps1 | iex
```

Verify your installation:

```powershell
buildshare --version
```

---

### Manual Installation (GitHub Releases)

If you prefer to install manually or your system has restricted network access:

1. Go to the [GitHub Releases](https://github.com/vishalkumardev/cli/releases) page.
2. Download the appropriate archive for your operating system and architecture:
   - **macOS Apple Silicon:** `buildshare_<version>_darwin_arm64.tar.gz`
   - **macOS Intel:** `buildshare_<version>_darwin_amd64.tar.gz`
   - **Linux x64:** `buildshare_<version>_linux_amd64.tar.gz`
   - **Linux ARM64:** `buildshare_<version>_linux_arm64.tar.gz`
   - **Windows x64:** `buildshare_<version>_windows_amd64.zip`
3. Extract the archive:
   - **macOS / Linux:**
     ```bash
     tar -xzf buildshare_*.tar.gz
     sudo mv buildshare /usr/local/bin/
     sudo chmod +x /usr/local/bin/buildshare
     ```
   - **Windows:** Extract `buildshare.exe` to a directory in your `PATH` (such as `%LOCALAPPDATA%\Programs\buildshare\bin`).

---

## Quick Start

### 1. Authenticate

Log in with your BuildShare account credentials or personal access token:

```bash
buildshare login
```

Check your authenticated user details:

```bash
buildshare whoami
```

### 2. Upload a Build

Upload an Android APK or iOS IPA package:

```bash
buildshare upload ./app-release.apk
```

Provide additional release notes or specify an app:

```bash
buildshare upload ./app-release.ipa --app <app-id> --notes "Sprint 42 test build"
```

### 3. List Apps & Builds

View all registered applications:

```bash
buildshare app list
```

View recent builds:

```bash
buildshare build list --app <app-id>
```

---

## CLI Commands

| Command                    | Description                         |
| -------------------------- | ----------------------------------- |
| `buildshare login`         | Authenticate with BuildShare        |
| `buildshare logout`        | Log out and revoke credentials      |
| `buildshare whoami`        | Show current user and workspace     |
| `buildshare upload <path>` | Upload a new mobile build (APK/IPA) |
| `buildshare app list`      | List apps in your workspace         |
| `buildshare build list`    | List recent builds for an app       |
| `buildshare version`       | Print version information           |

### Global Flags

- `--json`: Output command results as structured JSON (ideal for automation and scripts).
- `--ci`: Run in non-interactive CI mode (disables prompts and colors).
- `-v, --verbose`: Enable detailed debug logging.

---

## CI / CD Integration

In continuous integration environments (GitHub Actions, GitLab CI, Bitbucket Pipelines, CircleCI), supply the API token via the `BUILDSHARE_TOKEN` environment variable:

```bash
export BUILDSHARE_TOKEN="your_personal_access_token"
buildshare upload ./build/app-release.apk --ci
```

---

## License

MIT
