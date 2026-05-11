package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	pflag "github.com/spf13/pflag"

	"github.com/rmitchellscott/remarkable-go/device"
	"github.com/rmitchellscott/remarkable-go/executor"
	"github.com/rmitchellscott/remarkable-go/filesystem"
	"github.com/rmitchellscott/remarkable-go/partition"
)

var version = "dev"

var (
	showVersion = pflag.BoolP("version", "v", false, "Print version information and exit")
	showOnly    = pflag.BoolP("show-only", "s", false, "Only display current partition info, don't show selector")
	debug       = pflag.BoolP("debug", "d", false, "Enable debug logging to debug.log file")

	// Styles
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fallbackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	nextBootStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("15"))
)

var mgr partition.Manager

func main() {
	pflag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	ctx := context.Background()

	fs := filesystem.NewLocal()
	exec := executor.NewLocal()

	deviceType, err := device.Detect(fs)
	if err != nil {
		log.Fatalf("Failed to detect device type: %v", err)
	}

	mgr = partition.NewManager(exec, fs, deviceType)

	info, err := mgr.GetSystemInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get system info: %v", err)
	}

	if *debug {
		logToFile(fmt.Sprintf("SystemInfo: Active=%d (%s), Fallback=%d (%s), DeviceType=%s",
			info.Active.Number, info.Active.Version,
			info.Fallback.Number, info.Fallback.Version,
			info.DeviceType))
	}

	if *showOnly {
		displaySystemInfo(info)
		return
	}

	displaySystemInfo(info)

	if err := runInteractiveTUI(info); err != nil {
		log.Fatalf("Failed to run TUI: %v", err)
	}
}

func displaySystemInfo(info *partition.SystemInfo) {
	width := 50

	title := titleStyle.Width(width - 2).Render("reMarkable OS Version Switcher")
	titleBox := boxStyle.Width(width).Render(title)

	activeIndicator := ""
	if info.Active.IsActive {
		activeIndicator = activeStyle.Render(" [ACTIVE]")
	}

	nextBootIndicator := ""
	if info.Active.IsNextBoot {
		nextBootIndicator = activeStyle.Render(" [NEXT BOOT]")
	}

	fallbackNextBootIndicator := ""
	if info.Fallback.IsNextBoot {
		fallbackNextBootIndicator = nextBootStyle.Render(" [NEXT BOOT]")
	}

	maxVersionLen := len("Partition  A: " + info.Active.Version)
	if len("Partition  B: "+info.Fallback.Version) > maxVersionLen {
		maxVersionLen = len("Partition  B: " + info.Fallback.Version)
	}

	partAPadding := maxVersionLen - len("Partition  A: "+info.Active.Version)
	partBPadding := maxVersionLen - len("Partition  B: "+info.Fallback.Version)

	if partAPadding < 0 {
		partAPadding = 0
	}
	if partBPadding < 0 {
		partBPadding = 0
	}

	var lineA, lineB string

	if info.Active.Number == 2 {
		lineA = fmt.Sprintf("Partition  A: %s", activeStyle.Render(info.Active.Version)) + strings.Repeat(" ", partAPadding) + activeIndicator + nextBootIndicator
		lineB = fmt.Sprintf("Partition  B: %s", fallbackStyle.Render(info.Fallback.Version)) + strings.Repeat(" ", partBPadding) + fallbackNextBootIndicator
	} else {
		lineA = fmt.Sprintf("Partition  A: %s", fallbackStyle.Render(info.Fallback.Version)) + strings.Repeat(" ", partBPadding) + fallbackNextBootIndicator
		lineB = fmt.Sprintf("Partition  B: %s", activeStyle.Render(info.Active.Version)) + strings.Repeat(" ", partAPadding) + activeIndicator + nextBootIndicator
	}

	partitionContent := lineA + "\n" + lineB
	partitionBox := boxStyle.Width(width).Render(partitionContent)

	fmt.Println(titleBox)
	fmt.Println(partitionBox)
}

func runInteractiveTUI(info *partition.SystemInfo) error {
	ctx := context.Background()

	var showSelector bool = false

	overviewForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Change next boot partition?").
				Value(&showSelector),
		),
	).WithTheme(huh.ThemeBase())

	if err := overviewForm.Run(); err != nil {
		return fmt.Errorf("overview form error: %w", err)
	}

	if !showSelector {
		return nil
	}

	var selectedBoot int
	if info.Active.IsNextBoot {
		selectedBoot = info.Active.Number
	} else {
		selectedBoot = info.Fallback.Number
	}

	selectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select Next Boot Partition").
				Options(
					func() []huh.Option[int] {
						if info.Active.Number == 2 {
							return []huh.Option[int]{
								huh.NewOption(fmt.Sprintf("Partition A: %s", activeStyle.Render(info.Active.Version)), info.Active.Number),
								huh.NewOption(fmt.Sprintf("Partition B: %s", fallbackStyle.Render(info.Fallback.Version)), info.Fallback.Number),
							}
						} else {
							return []huh.Option[int]{
								huh.NewOption(fmt.Sprintf("Partition A: %s", fallbackStyle.Render(info.Fallback.Version)), info.Fallback.Number),
								huh.NewOption(fmt.Sprintf("Partition B: %s", activeStyle.Render(info.Active.Version)), info.Active.Number),
							}
						}
					}()...,
				).
				Value(&selectedBoot),
		),
	).WithTheme(huh.ThemeBase())

	if err := selectForm.Run(); err != nil {
		return fmt.Errorf("select form error: %w", err)
	}

	nextBoot := info.Active.Number
	if info.Fallback.IsNextBoot {
		nextBoot = info.Fallback.Number
	}

	if selectedBoot == nextBoot {
		fmt.Printf("No changes needed. Partition %d is already set to boot next.\n", selectedBoot)
		return nil
	}

	if err := mgr.CanSwitchTo(info, selectedBoot); err != nil {
		if errors.Is(err, partition.ErrEncryptionBlocked) {
			var targetVersion string
			if selectedBoot == info.Active.Number {
				targetVersion = info.Active.Version
			} else {
				targetVersion = info.Fallback.Version
			}

			warningTitle := titleStyle.Width(46).Render("Cannot Switch to Pre-3.18 Firmware")

			warningMsg := warningStyle.Width(46).Align(lipgloss.Center).Render(fmt.Sprintf(`This device has encryption enabled, which
was introduced in firmware 3.18.

Switching to version %s would make your
device unbootable.

Please disable encryption before
downgrading to pre-3.18 firmware.`, targetVersion))

			var abort bool = true

			warningForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(warningTitle).
						Description(warningMsg).
						Affirmative("CANCEL").
						Negative("").
						Value(&abort),
				),
			).WithTheme(huh.ThemeBase())

			if err := warningForm.Run(); err != nil {
				return fmt.Errorf("warning form error: %w", err)
			}

			return nil
		}
		return fmt.Errorf("cannot switch partition: %w", err)
	}

	result, err := mgr.SwitchBoot(ctx, selectedBoot)
	if err != nil {
		return fmt.Errorf("failed to switch boot partition: %w", err)
	}

	if *debug {
		logToFile(fmt.Sprintf("SwitchBoot result: %s (method: %s)", result.Message, result.Method))
	}

	fmt.Println(result.Message)

	updatedInfo, err := mgr.GetSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh system info: %w", err)
	}

	fmt.Print("\033[10A")
	fmt.Print("\033[J")

	displaySystemInfo(updatedInfo)

	var shouldReboot bool = false

	rebootForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Reboot now?").
				Value(&shouldReboot),
		),
	).WithTheme(huh.ThemeBase())

	if err := rebootForm.Run(); err != nil {
		return fmt.Errorf("reboot form error: %w", err)
	}

	return handleRebootDecision(shouldReboot, selectedBoot, updatedInfo)
}

func handleRebootDecision(shouldReboot bool, selectedBoot int, info *partition.SystemInfo) error {
	var selectedVersion string
	if selectedBoot == info.Active.Number {
		selectedVersion = info.Active.Version
	} else {
		selectedVersion = info.Fallback.Version
	}

	if shouldReboot {
		fmt.Printf("Rebooting now to version %s...\n", selectedVersion)
		if err := mgr.Reboot(context.Background()); err != nil {
			return fmt.Errorf("failed to reboot: %w", err)
		}
	} else {
		fmt.Printf("Version will switch to %s at the next reboot.\n", selectedVersion)
	}

	return nil
}

func logToFile(message string) {
	f, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := fmt.Sprintf("[%v] ", time.Now().Format("2006-01-02 15:04:05"))
	f.WriteString(fmt.Sprintf("%s%s\n", timestamp, message))
}
