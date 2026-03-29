# Forensic Duplicator

Desktop disk imaging and cloning tool built with Go, Wails, and Svelte.

This project is designed as a portfolio-grade implementation of a forensic-style duplication workflow with real-time progress, multi-target copy, and hash reporting.

## Features

- Dual imaging modes:
  - Logical volume copy
  - Raw physical disk imaging
- Source and target disk selection from the desktop UI
- Optional target formatting in logical mode (with source/system safety checks)
- Parallel writes to multiple targets
- Live progress telemetry:
  - bytes copied
  - percent complete
  - throughput
  - estimated time remaining
- Hash reporting for raw imaging:
  - MD5
  - SHA-256
- Activity logging and user-triggered cancellation

## Interface Screenshots

Add screenshots to `docs/screenshots/` using these filenames:

![Main Dashboard](docs/screenshots/01-main-dashboard.png)
_Placeholder: main UI with source/target selection and copy mode._

![Disk Selection](docs/screenshots/02-disk-selection.png)
_Placeholder: selected source and multiple targets._

## Tech Stack

- Backend: Go
- Desktop runtime: Wails v2
- Frontend: Svelte + Vite
- Platform-specific disk integration: Windows APIs + PowerShell

## Project Structure

- `main.go` - Wails bootstrap and app bindings
- `app.go` - UI-exposed operations (enumeration, start/stop, formatting)
- `internal/forensic/copier.go` - copy logic, progress events, raw imaging, hash calculation
- `internal/forensic/disk.go` - cross-platform abstraction and validation entry points
- `internal/forensic/disk_windows.go` - Windows disk enumeration/info/format operations
- `frontend/src/App.svelte` - interface, telemetry display, and interaction flow

## Getting Started

### Prerequisites

- Go
- Node.js + npm
- Wails CLI

### Install Dependencies

```bash
cd /Users/igoradulyan/GolandProjects/icsiq/forensic-duplicator
go mod download
cd frontend
npm install
```

### Run in Development

```bash
cd /Users/igoradulyan/GolandProjects/icsiq/forensic-duplicator
wails dev
```

### Build

```bash
cd /Users/igoradulyan/GolandProjects/icsiq/forensic-duplicator
wails build
```

## Platform Status

- Windows workflow is currently the primary implementation path
- Linux and macOS disk backends are scaffolded but not fully implemented
- Raw imaging may require elevated privileges depending on device access

## Demo Workflow

1. Refresh available disks
2. Select copy mode (logical or raw)
3. Choose one source and one or more targets
4. Optionally format targets (logical mode)
5. Start imaging and monitor progress
6. Review hash output after completion

## Roadmap

- Complete Linux and macOS disk support
- Add post-copy verification for each target
- Export forensic report artifacts (metadata + hash evidence)
- Improve error classification and recovery behavior
- Add end-to-end test coverage for major workflows

## Safety and Legal Notice

This tool performs low-level disk operations and can permanently overwrite data.
Use only on systems/devices you are explicitly authorized to examine.

## Author

Igor Adulyan

- GitHub: `https://github.com/<your-username>`
- Portfolio: `https://<your-portfolio-url>`
- LinkedIn: `https://linkedin.com/in/<your-profile>`
