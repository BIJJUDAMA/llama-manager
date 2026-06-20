# Llama Manager

A terminal-based manager for local large language models. It handles model discovery, hardware suitability estimation, launch profiles, model downloads, and llama.cpp lifecycle management.

## Requirements

* Go 1.26 or later
* llama.cpp binaries (downloaded and managed from within the application)
* Windows (Linux and macOS support is planned)

## Installation

Ensure Go is installed on your system. Run the following command to download and install the application:

```
go install github.com/BIJJUDAMA/llama-manager/cmd/llmgr@latest
```

The binary is placed in your Go bin directory (typically `$HOME/go/bin` or `%USERPROFILE%\go\bin`). Make sure this directory is in your system's `PATH`.

To temporarily add it to your path:

### Windows (PowerShell)
```powershell
$env:PATH += ";$env:USERPROFILE\go\bin"
```

### Linux / macOS
```bash
export PATH="$HOME/go/bin:$PATH"
```

## Usage

Start the application with:

```
llmgr
```

Navigation is keyboard-driven. Follow the key guides at the bottom of the screen.

### Flags
* `--version`: Print the installed version and exit.
* `--reset-onboarding`: Re-run the onboarding tour on the next launch.

## Data Directory

All application data (models, configuration, llama.cpp binaries, download queue) is stored in a fixed directory. On Windows, this is located at `%APPDATA%\llmgr`.

Place GGUF model files under the `models` subdirectory. The application discovers them recursively on startup.

## Configuration

Settings such as color themes and Hugging Face tokens are stored in `config.json` in the data directory and can be edited directly inside the application's Settings screen (`U` key).

## Updating

To update Llama Manager to the latest version, run:

```
go install github.com/BIJJUDAMA/llama-manager/cmd/llmgr@latest
```

## License

See [LICENSE](LICENSE).
