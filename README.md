# llama-manager

A terminal-based manager for local large language models. It handles model discovery, hardware suitability estimation, launch profiles, and llama.cpp lifecycle management — all from the keyboard.

## Requirements

- [Go 1.21 or later](https://go.dev/dl/)
- llama.cpp binaries (managed from within the application)
- Windows (Linux and macOS support is planned)

## Data Directory

All application data — models, configuration, llama.cpp binaries, download cache — is stored in a fixed directory determined by the operating system. It is created automatically on first launch.

| Platform | Path |
|---|---|
| Windows | `%APPDATA%\llmgr` |
| Linux | planned |
| macOS | planned |

## Installation

```
go install github.com/BIJJUDAMA/llama-manager/cmd/llmgr@latest
```

The binary is placed in `$GOPATH/bin` (default: `$HOME/go/bin`). Ensure that directory is on your `PATH`. After a standard Go installation it usually is, but if not, add it manually:

**Linux / macOS:**
```
export PATH="$HOME/go/bin:$PATH"
```

**Windows (PowerShell):**
```
$env:PATH += ";$env:USERPROFILE\go\bin"
```

Add the line to your shell profile to make it permanent.

## Usage

```
llmgr
```

Navigation is keyboard-driven. Press `?` or follow the footer hints shown at the bottom of each screen.

**Flags:**

| Flag | Description |
|---|---|
| `--version` | Print the installed version and exit |
| `--reset-onboarding` | Re-run the onboarding tour on next launch |

## Models

Place GGUF model files anywhere under the `models` subdirectory of the data directory. The application discovers them recursively on startup and re-scans as needed.

## Configuration

Configuration is stored automatically on first launch. Settings such as the color theme and Hugging Face token can be changed from within the Settings screen (`U` key).

## Updating

```
go install github.com/BIJJUDAMA/llama-manager/cmd/llmgr@latest
```

Running the same install command pulls the latest tagged release.

## License

See [LICENSE](LICENSE).
