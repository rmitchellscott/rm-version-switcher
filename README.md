# reMarkable Version Switcher

A beginner-friendly application for switching between reMarkable OS versions with an interactive interface.

## Features

- **TUI interface** - Clean, boxed layout with color-coded partitions
- **Interactive version switching** - Select which version to boot next with arrow keys
- **Real-time status display** - Shows current active partition and next boot selection
- **Smart partition mapping** - Consistent A/B labeling (A=p2, B=p3)
- **Integrated reboot option** - Choose to reboot immediately or defer to next restart

## Installation

### Download Latest Release

For **ARMv7** systems (reMarkable 1 & 2):
```bash
wget https://github.com/rmitchellscott/rm-version-switcher/releases/latest/download/rm-version-switcher.tar.gz
tar -xzf rm-version-switcher.tar.gz
```

For **ARM64/AArch64** systems (reMarkable Paper Pro):
```bash
wget https://github.com/rmitchellscott/rm-version-switcher/releases/latest/download/rmpp-version-switcher.tar.gz
tar -xzf rmpp-version-switcher.tar.gz
```

Verify installation:
```bash
rm-version-switcher --help
# or for aarch64:
rmpp-version-switcher --help
```

### Manual Copy to Device

Alternatively, copy the appropriate binary directly to your reMarkable device:

```bash
# Copy to reMarkable 1/2 (replace with your device IP)
scp rm-version-switcher root@10.11.99.1:~/

# Copy to reMarkable Paper Pro (replace with your device IP)  
scp rmpp-version-switcher root@10.11.99.1:~/
```

## Usage

### Interactive Mode
```bash
# reMarkable 1 & 2
./rm-version-switcher

# reMarkable Paper Pro
./rmpp-version-switcher
```

Shows the overview, allows you to change the next boot partition, and optionally reboot immediately.

### View Only Mode
```bash
# reMarkable 1 & 2
./rm-version-switcher --show-only

# reMarkable Paper Pro
./rmpp-version-switcher --show-only
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

#### reMarkable Paper Pro (ARM64)
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o rmpp-version-switcher .
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
