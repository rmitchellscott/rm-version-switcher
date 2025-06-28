# reMarkable Version Switcher
[![rm1](https://img.shields.io/badge/rM1-supported-green)](https://remarkable.com/store/remarkable)
[![rm2](https://img.shields.io/badge/rM2-supported-green)](https://remarkable.com/store/remarkable-2)

A beginner-friendly application for switching between currently installed reMarkable OS versions with an interactive interface.

Supports reMarkable 1 and 2 at this time.

Supports reMarkable 1 and 2 at this time.

## Features

- **TUI interface** - Clean, boxed layout with color-coded partitions
- **Interactive version switching** - Select which version to boot next with arrow keys
- **Real-time status display** - Shows current active partition and next boot selection
- **Smart partition mapping** - Consistent A/B labeling (A=p2, B=p3)
- **Integrated reboot option** - Choose to reboot immediately or defer to next restart

<div align="center">
  <video src="https://github.com/user-attachments/assets/941202e1-67c5-45c2-8df2-b66b6a084a61"></video>
</div>

## Installation

### Download Latest Release

For **reMarkable 1 & 2** (ARMv7):
```bash
wget https://github.com/rmitchellscott/rm-version-switcher/releases/latest/download/rm-version-switcher.tar.gz
tar -xzf rm-version-switcher.tar.gz
```

Verify installation:
```bash
rm-version-switcher --help
```

### Manual Copy to Device

Alternatively, copy the binary directly to your reMarkable device:

```bash
# Copy to reMarkable 1/2 (replace with your device IP)
scp rm-version-switcher root@10.11.99.1:~/
```

## Usage

### Interactive Mode
```bash
./rm-version-switcher
```

Shows the overview, allows you to change the next boot partition, and optionally reboot immediately.

### View Only Mode
```bash
./rm-version-switcher --show-only
```

Display current partition status without any interactive options.

## Building

### Prerequisites
- Go 1.21 or later

### Install Dependencies
```bash
go mod tidy
```

### Build for reMarkable Devices

#### reMarkable 1 & reMarkable 2 (ARMv7)
```bash
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o rm-version-switcher .
```

### Build for Development/Testing
```bash
# Current platform (for testing with --dry-run)
go build -o rm-version-switcher .
```

## Interface

### Overview Display
```
┌──────────────────────────────────────────────────┐
│         reMarkable OS Version Switcher           │
└──────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────┐
│ Partition  A: 3.20.0.92 [NEXT BOOT]              │
│ Partition  B: 3.18.2.3  [ACTIVE]                 │
└──────────────────────────────────────────────────┘
```

### Color Coding
- **Green**: Active partition version numbers and [NEXT BOOT] when on active partition
- **Blue**: Fallback partition version numbers  
- **Yellow**: [NEXT BOOT] when on fallback partition

### Labels
- **[ACTIVE]**: Currently running partition
- **[NEXT BOOT]**: Partition that will boot after next reboot

## How It Works

The application follows the same proven logic as the reference [switch.sh](https://github.com/ddvk/remarkable-update/blob/main/switch.sh):

1. **Detects current state** using `rootdev` and `fw_printenv`
2. **Reads version information** from `/usr/share/remarkable/update.conf`
3. **Updates boot environment** using `fw_setenv` commands:
   - `upgrade_available=1`
   - `bootcount=0`
   - `fallback_partition={old_partition}`
   - `active_partition={new_partition}`

## Dependencies

- [charmbracelet/huh](https://github.com/charmbracelet/huh) - TUI forms and interactions
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
