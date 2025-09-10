package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type PartitionInfo struct {
	Number     int
	Version    string
	IsActive   bool
	IsNextBoot bool
}

type SystemInfo struct {
	Active     PartitionInfo
	Fallback   PartitionInfo
	NextBoot   int
	IsPaperPro bool
}

var (
	dryRun      = flag.Bool("dry-run", false, "Enable dry run mode for testing")
	showOnly    = flag.Bool("show-only", false, "Only display current partition info, don't show selector")
	resetDryRun = flag.Bool("reset-dry-run", false, "Reset dry run state to defaults")
	debug       = flag.Bool("debug", false, "Enable debug logging to debug.log file")

	// Styles
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fallbackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	nextBootStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("15"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

func main() {
	flag.Parse()

	if *resetDryRun {
		os.Remove("dry-run-boot.txt")
		fmt.Println("Reset dry run state to defaults")
		return
	}

	info, err := getSystemInfo()
	if err != nil {
		log.Fatalf("Failed to get system info: %v", err)
	}

	if *showOnly {
		displaySystemInfo(info)
		return
	}

	// Show overview first
	displaySystemInfo(info)

	if err := runInteractiveTUI(info); err != nil {
		log.Fatalf("Failed to run TUI: %v", err)
	}
}

func getSystemInfo() (*SystemInfo, error) {
	if *dryRun {
		return getDryRunSystemInfo()
	}

	// Check if this is a Paper Pro device
	isPaperPro := isPaperProDevice()

	var runningP, otherP, bootP int
	var err error

	if isPaperPro {
		// Paper Pro specific logic
		runningP, otherP, bootP, err = getPaperProPartitionInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to get Paper Pro partition info: %w", err)
		}
	} else {
		// Original logic for reMarkable 1 and 2
		runningDev, err := exec.Command("rootdev").Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get root device: %w", err)
		}

		runningDevStr := strings.TrimSpace(string(runningDev))
		re := regexp.MustCompile(`p(\d+)$`)
		matches := re.FindStringSubmatch(runningDevStr)
		if len(matches) < 2 {
			return nil, fmt.Errorf("could not parse partition number from %s", runningDevStr)
		}

		runningP, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid partition number: %w", err)
		}

		// Determine other partition
		otherP = 2
		if runningP == 2 {
			otherP = 3
		}

		// Get next boot partition
		bootPOut, err := exec.Command("fw_printenv", "active_partition").Output()
		bootP = runningP // default fallback
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(bootPOut)), "=")
			if len(parts) == 2 {
				if bp, err := strconv.Atoi(parts[1]); err == nil {
					bootP = bp
				}
			}
		}
	}

	// Get active version
	activeVersion, err := getVersionFromPartition(runningP, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get active version: %w", err)
	}

	// Get fallback version
	fallbackVersion, err := getVersionFromPartition(otherP, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get fallback version: %w", err)
	}

	info := &SystemInfo{
		Active: PartitionInfo{
			Number:     runningP,
			Version:    activeVersion,
			IsActive:   true,
			IsNextBoot: bootP == runningP,
		},
		Fallback: PartitionInfo{
			Number:     otherP,
			Version:    fallbackVersion,
			IsActive:   false,
			IsNextBoot: bootP == otherP,
		},
		NextBoot:   bootP,
		IsPaperPro: isPaperPro,
	}

	if *debug {
		logToFile(fmt.Sprintf("SystemInfo: runningP=%d, otherP=%d, bootP=%d", runningP, otherP, bootP))
		logToFile(fmt.Sprintf("Active: Number=%d, Version=%s, IsNextBoot=%v", info.Active.Number, info.Active.Version, info.Active.IsNextBoot))
		logToFile(fmt.Sprintf("Fallback: Number=%d, Version=%s, IsNextBoot=%v", info.Fallback.Number, info.Fallback.Version, info.Fallback.IsNextBoot))
	}

	return info, nil
}

func getVersionFromPartition(partNum int, isActive bool) (string, error) {
	if isActive {
		// Use /etc/os-release for all devices
		if version, err := getVersionFromOSRelease(); err == nil {
			return version, nil
		}
		return "unknown", nil
	} else {
		// Mount the other partition temporarily
		runningDev, err := exec.Command("rootdev").Output()
		if err != nil {
			return "", fmt.Errorf("failed to get root device: %w", err)
		}

		runningDevStr := strings.TrimSpace(string(runningDev))
		baseDev := regexp.MustCompile(`p\d+$`).ReplaceAllString(runningDevStr, "")

		mountPoint := fmt.Sprintf("/tmp/mount_p%d", partNum)
		if err := os.MkdirAll(mountPoint, 0755); err != nil {
			return "", fmt.Errorf("failed to create mount point: %w", err)
		}

		defer func() {
			exec.Command("umount", mountPoint).Run()
			os.RemoveAll(mountPoint)
		}()

		if err := exec.Command("mount", "-o", "ro", fmt.Sprintf("%sp%d", baseDev, partNum), mountPoint).Run(); err != nil {
			return "", fmt.Errorf("failed to mount partition %d: %w", partNum, err)
		}

		// Try to get version from mounted partition
		if version, err := getVersionFromPartitionPath(mountPoint); err == nil {
			return version, nil
		}
		return "unknown", nil
	}
}

func displaySystemInfo(info *SystemInfo) {
	width := 50

	// Title
	title := titleStyle.Width(width - 2).Render("reMarkable OS Version Switcher")
	titleBox := boxStyle.Width(width).Render(title)

	// Partition info
	activeIndicator := ""
	if info.Active.IsActive {
		activeIndicator = activeStyle.Render(" [ACTIVE]")
	}

	nextBootIndicator := ""
	if info.Active.IsNextBoot {
		nextBootIndicator = activeStyle.Render(" [NEXT BOOT]") // Green when on active
	}

	fallbackNextBootIndicator := ""
	if info.Fallback.IsNextBoot {
		fallbackNextBootIndicator = nextBootStyle.Render(" [NEXT BOOT]") // Yellow when on fallback
	}

	// Build the base lines with versions
	partAVersionOnly := fmt.Sprintf("Partition  A: %s", activeStyle.Render(info.Active.Version))
	partBVersionOnly := fmt.Sprintf("Partition  B: %s", fallbackStyle.Render(info.Fallback.Version))

	// Calculate padding to align labels at the same column where [ACTIVE] appears
	// Find the longest version text to use as baseline
	maxVersionLen := len("Partition  A: " + info.Active.Version)
	if len("Partition  B: "+info.Fallback.Version) > maxVersionLen {
		maxVersionLen = len("Partition  B: " + info.Fallback.Version)
	}

	partAPadding := maxVersionLen - len("Partition  A: "+info.Active.Version)
	partBPadding := maxVersionLen - len("Partition  B: "+info.Fallback.Version)

	// Ensure padding is never negative
	if partAPadding < 0 {
		partAPadding = 0
	}
	if partBPadding < 0 {
		partBPadding = 0
	}

	// Build final lines with aligned labels
	// Map partitions correctly: A=p2, B=p3
	var lineA, lineB string

	if info.Active.Number == 2 {
		// Active is p2, so A=Active, B=Fallback
		lineA = partAVersionOnly + strings.Repeat(" ", partAPadding) + activeIndicator + nextBootIndicator
		lineB = partBVersionOnly + strings.Repeat(" ", partBPadding) + fallbackNextBootIndicator
	} else {
		// Active is p3, so A=Fallback, B=Active
		lineA = fmt.Sprintf("Partition  A: %s", fallbackStyle.Render(info.Fallback.Version)) + strings.Repeat(" ", partBPadding) + fallbackNextBootIndicator
		lineB = fmt.Sprintf("Partition  B: %s", activeStyle.Render(info.Active.Version)) + strings.Repeat(" ", partAPadding) + activeIndicator + nextBootIndicator
	}

	partALine := lineA
	partBLine := lineB

	partitionContent := partALine + "\n" + partBLine
	partitionBox := boxStyle.Width(width).Render(partitionContent)

	// // Actions
	// actionsContent := labelStyle.Render("Actions: [S]elect next boot    [Q]uit")
	// actionsBox := boxStyle.Width(width).Render(actionsContent)

	fmt.Println(titleBox)
	fmt.Println(partitionBox)
	// fmt.Println(actionsBox)
}

func runInteractiveTUI(info *SystemInfo) error {
	// Step 1: Overview + Change confirmation
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

	// Step 2: Partition selection
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
					// A=p2, B=p3 mapping
					func() []huh.Option[int] {
						if info.Active.Number == 2 {
							// Active is p2, so A=Active, B=Fallback
							return []huh.Option[int]{
								huh.NewOption(fmt.Sprintf("Partition A: %s", activeStyle.Render(info.Active.Version)), info.Active.Number),
								huh.NewOption(fmt.Sprintf("Partition B: %s", fallbackStyle.Render(info.Fallback.Version)), info.Fallback.Number),
							}
						} else {
							// Active is p3, so A=Fallback, B=Active
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

	if selectedBoot == info.NextBoot {
		fmt.Printf("No changes needed. Partition %d is already set to boot next.\n", selectedBoot)
		return nil
	}

	// Step 3: Switch boot partition
	if err := switchBootPartition(selectedBoot, info.NextBoot); err != nil {
		return err
	}

	// Step 4: Show updated overview + Reboot confirmation
	updatedInfo, err := getSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to refresh system info: %w", err)
	}

	var shouldReboot bool = false
	// Clear the old overview and show updated one
	// Different number of lines to clear based on dry run vs real mode
	if *dryRun {
		// fmt.Print("\033[1A") // Move up 1 line in dry run mode
	} else {
		fmt.Print("\033[10A") // Move up 10 lines for real fw_setenv output
	}
	fmt.Print("\033[J") // Clear from cursor to end of screen

	displaySystemInfo(updatedInfo)

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

func buildSystemInfoDisplay(info *SystemInfo) string {
	var lines []string

	lines = append(lines, "reMarkable OS Version Switcher")
	lines = append(lines, "")

	// Build partition lines with plain text (no lipgloss styling for huh)
	activeIndicator := ""
	if info.Active.IsActive {
		activeIndicator = " [ACTIVE]"
	}

	nextBootIndicator := ""
	if info.Active.IsNextBoot {
		nextBootIndicator = " [NEXT BOOT]"
	}

	fallbackNextBootIndicator := ""
	if info.Fallback.IsNextBoot {
		fallbackNextBootIndicator = " [NEXT BOOT]"
	}

	// Calculate padding for alignment
	baseVersionLen := len("Partition  A: " + info.Active.Version)
	partAPadding := baseVersionLen - len("Partition  A: "+info.Active.Version)
	partBPadding := baseVersionLen - len("Partition  B: "+info.Fallback.Version)

	partALine := fmt.Sprintf("Partition  A: %s%s%s%s",
		info.Active.Version,
		strings.Repeat(" ", partAPadding),
		activeIndicator,
		nextBootIndicator)
	partBLine := fmt.Sprintf("Partition  B: %s%s%s",
		info.Fallback.Version,
		strings.Repeat(" ", partBPadding),
		fallbackNextBootIndicator)

	lines = append(lines, partALine)
	lines = append(lines, partBLine)

	return strings.Join(lines, "\n")
}

func handleRebootDecision(shouldReboot bool, selectedBoot int, info *SystemInfo) error {
	// Get the version for the selected boot partition
	var selectedVersion string
	if selectedBoot == info.Active.Number {
		selectedVersion = info.Active.Version
	} else {
		selectedVersion = info.Fallback.Version
	}

	if shouldReboot {
		if *dryRun {
			fmt.Printf("[DRY RUN] Would reboot now to version %s\n", selectedVersion)
		} else {
			fmt.Printf("Rebooting now to version %s...\n", selectedVersion)
			if err := exec.CommandContext(context.Background(), "reboot").Run(); err != nil {
				return fmt.Errorf("failed to reboot: %w", err)
			}
		}
	} else {
		fmt.Printf("Version will switch to %s at the next reboot.\n", selectedVersion)
	}

	return nil
}

func runRebootConfirmation(selectedBoot int, info *SystemInfo) error {
	var shouldReboot bool = false // Default to No

	// Get the version for the selected boot partition
	var selectedVersion string
	if selectedBoot == info.Active.Number {
		selectedVersion = info.Active.Version
	} else {
		selectedVersion = info.Fallback.Version
	}

	// Ask if they want to reboot now
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

	if shouldReboot {
		if *dryRun {
			fmt.Printf("[DRY RUN] Would reboot now to version %s\n", selectedVersion)
		} else {
			fmt.Printf("Rebooting now to version %s...\n", selectedVersion)
			if err := exec.CommandContext(context.Background(), "reboot").Run(); err != nil {
				return fmt.Errorf("failed to reboot: %w", err)
			}
		}
	} else {
		fmt.Printf("Version will switch to %s at the next reboot.\n", selectedVersion)
	}

	return nil
}

func handleBootSelection(selectedBoot int, info *SystemInfo) error {
	if selectedBoot == info.NextBoot {
		fmt.Printf("No changes needed. Partition %d is already set to boot next.\n", selectedBoot)
		return nil
	}

	// Switch the boot partition first
	if err := switchBootPartition(selectedBoot, info.NextBoot); err != nil {
		return err
	}

	// Update system info to reflect the change
	updatedInfo, err := getSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to refresh system info: %w", err)
	}

	// Clear screen and show updated overview
	fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top
	displaySystemInfo(updatedInfo)

	// Then ask about reboot
	return runRebootConfirmation(selectedBoot, updatedInfo)
}

func switchBootPartition(newPart, oldPart int) error {
	if *dryRun {
		return saveDryRunBootPartition(newPart)
	}

	// Check if this is a Paper Pro device
	isPaperPro := isPaperProDevice()

	// Get the actual version from the target partition
	version, err := getVersionFromPartition(newPart, false)
	if err != nil {
		version = "unknown"
	}

	fmt.Printf("Setting next boot to version %s (partition %d)...\n", version, newPart)

	if isPaperPro {
		return switchPaperProBootPartition(newPart, version)
	}

	// Original logic for reMarkable 1 and 2
	commands := [][]string{
		{"fw_setenv", "upgrade_available", "1"},
		{"fw_setenv", "bootcount", "0"},
		{"fw_setenv", "fallback_partition", strconv.Itoa(oldPart)},
		{"fw_setenv", "active_partition", strconv.Itoa(newPart)},
	}

	for _, cmd := range commands {
		cmdStr := fmt.Sprintf("%s %s %s", cmd[0], cmd[1], cmd[2])
		if *debug {
			logToFile(fmt.Sprintf("Running: %s", cmdStr))
		}

		if err := exec.CommandContext(context.Background(), cmd[0], cmd[1:]...).Run(); err != nil {
			errMsg := fmt.Sprintf("ERROR: Command failed: %v", err)
			if *debug {
				logToFile(errMsg)
			}
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}

		if *debug {
			logToFile("✓ Success")
		}
	}

	fmt.Printf("Successfully set next boot to version %s (partition %d)\n", version, newPart)
	fmt.Println("Reboot to boot into the selected partition.")

	return nil
}

func getDryRunSystemInfo() (*SystemInfo, error) {
	// Default values - Active is always partition 3, Fallback is always partition 2
	activePartition := 3
	fallbackPartition := 2
	nextBootPartition := 3

	// Try to read stored boot partition
	if data, err := os.ReadFile("dry-run-boot.txt"); err == nil {
		if boot, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && (boot == 2 || boot == 3) {
			nextBootPartition = boot
		}
	}

	return &SystemInfo{
		Active: PartitionInfo{
			Number:     activePartition,
			Version:    "3.20.0.92",
			IsActive:   true,
			IsNextBoot: nextBootPartition == activePartition,
		},
		Fallback: PartitionInfo{
			Number:     fallbackPartition,
			Version:    "3.18.2.3",
			IsActive:   false,
			IsNextBoot: nextBootPartition == fallbackPartition,
		},
		NextBoot:   nextBootPartition,
		IsPaperPro: false,
	}, nil
}

func saveDryRunBootPartition(partition int) error {
	// Get version for the selected partition
	var version string
	if partition == 3 {
		version = "3.20.0.92"
	} else {
		version = "3.18.2.3"
	}

	fmt.Printf("[DRY RUN] Setting next boot to version %s (partition %d)\n", version, partition)

	if err := os.WriteFile("dry-run-boot.txt", []byte(strconv.Itoa(partition)), 0644); err != nil {
		return fmt.Errorf("failed to save dry run state: %w", err)
	}

	fmt.Printf("Saved boot partition %d to dry-run-boot.txt\n", partition)
	fmt.Println("Run again to see the updated boot configuration.")

	return nil
}

func isPaperProDevice() bool {
	// List of known Paper Pro model names
	paperProModels := []string{
		"Ferrari",
		"Chiappa",
	}

	// Check if this is a Paper Pro device by examining the device tree model
	for _, model := range paperProModels {
		pattern := fmt.Sprintf("reMarkable %s", model)
		if err := exec.Command("grep", "-q", pattern, "/proc/device-tree/model").Run(); err == nil {
			return true
		}
	}
	return false
}

// Helper function to get partition number from device path (e.g., /dev/mmcblk0p2 -> 2)
func getPartitionNumberFromDevice(device string) (int, error) {
	re := regexp.MustCompile(`p(\d+)$`)
	matches := re.FindStringSubmatch(device)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse partition number from %s", device)
	}
	return strconv.Atoi(matches[1])
}

// Helper function to get the next boot partition based on OS version
func getPaperProNextBootPartition(currentVersion string) (int, error) {
	// Check if version is 3.22 or higher
	if compareVersions(currentVersion, "3.22") >= 0 {
		// For 3.22+, read from the new mmc boot_part location
		bootPartData, err := os.ReadFile("/sys/bus/mmc/devices/mmc0:0001/boot_part")
		if err != nil {
			// Fallback to old location if new one doesn't exist
			if os.IsNotExist(err) {
				return getPaperProNextBootPartitionLegacy()
			}
			return 0, fmt.Errorf("failed to read boot_part: %w", err)
		}
		
		bootPart := strings.TrimSpace(string(bootPartData))
		// In 3.22+: "1" means root_a (partition 2), "2" means root_b (partition 3)
		if bootPart == "1" {
			return 2, nil
		} else if bootPart == "2" {
			return 3, nil
		}
		return 0, fmt.Errorf("unexpected boot_part value: %s", bootPart)
	}
	
	// For versions < 3.22, use the legacy method
	return getPaperProNextBootPartitionLegacy()
}

// Helper function for legacy next boot partition detection
func getPaperProNextBootPartitionLegacy() (int, error) {
	nextBootPartData, err := os.ReadFile("/sys/devices/platform/lpgpr/root_part")
	if err != nil {
		return 0, fmt.Errorf("failed to read root_part: %w", err)
	}
	
	nextBootPart := strings.TrimSpace(string(nextBootPartData))
	if nextBootPart == "a" {
		return 2, nil
	} else if nextBootPart == "b" {
		return 3, nil
	}
	return 0, fmt.Errorf("unexpected root_part value: %s", nextBootPart)
}

func getPaperProPartitionInfo() (int, int, int, error) {
	// Get active partition using swupdate -g (like reference scripts)
	activePartOut, err := exec.Command("swupdate", "-g").Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get active partition: %w", err)
	}

	activeDevice := strings.TrimSpace(string(activePartOut))
	runningP, err := getPartitionNumberFromDevice(activeDevice)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse active partition: %w", err)
	}

	// Validate partition number
	if runningP != 2 && runningP != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected partition number: %d", runningP)
	}

	// Determine other partition
	otherP := 2
	if runningP == 2 {
		otherP = 3
	}

	// Get current OS version to determine which sysfs path to use
	currentVersion, err := getVersionFromOSRelease()
	if err != nil {
		// If we can't get version, assume legacy behavior
		currentVersion = "3.20"
	}

	// Get next boot partition using version-aware method
	bootP, err := getPaperProNextBootPartition(currentVersion)
	if err != nil {
		// Fallback to current partition if we can't determine next boot
		bootP = runningP
	}

	return runningP, otherP, bootP, nil
}

func switchPaperProBootPartition(newPart int, targetVersion string) error {
	// Validate partition number
	if newPart != 2 && newPart != 3 {
		return fmt.Errorf("invalid partition number: %d", newPart)
	}

	// Get current running version to determine which method we can use
	currentVersion, err := getVersionFromOSRelease()
	if err != nil {
		// If we can't determine current version, assume we need to use mmc for safety
		currentVersion = "3.22"
	}

	// Determine which switching method to use:
	// 1. If currently running 3.22+: MUST use mmc (permission denied on sysfs write)
	// 2. If target is 3.22+: SHOULD use mmc (to properly set up boot for 3.22+)
	// 3. Otherwise: Use legacy sysfs write
	useMmcMethod := compareVersions(currentVersion, "3.22") >= 0 || compareVersions(targetVersion, "3.22") >= 0

	if useMmcMethod {
		// Use mmc bootpart enable commands
		// Based on reference script: partition 2 (root_a) uses boot0, partition 3 (root_b) uses boot1
		var mmcCmd []string
		if newPart == 2 {
			// Switch to root_a: enable boot partition 1 on mmcblk0boot0
			mmcCmd = []string{"mmc", "bootpart", "enable", "1", "0", "/dev/mmcblk0boot0"}
		} else {
			// Switch to root_b: enable boot partition 2 on mmcblk0boot1
			mmcCmd = []string{"mmc", "bootpart", "enable", "2", "0", "/dev/mmcblk0boot1"}
		}

		if err := exec.Command(mmcCmd[0], mmcCmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run mmc bootpart enable: %w", err)
		}

		fmt.Printf("Successfully set Paper Pro next boot to version %s (partition %d) using mmc bootpart\n", targetVersion, newPart)
		fmt.Println("Reboot to boot into the selected partition.")
		return nil
	}

	// Use legacy Paper Pro method only when BOTH current and target are < 3.22
	var newPartLabel string
	if newPart == 2 {
		newPartLabel = "a"
	} else {
		newPartLabel = "b"
	}

	// Write the new partition label to the Paper Pro boot partition file
	if err := os.WriteFile("/sys/devices/platform/lpgpr/root_part", []byte(newPartLabel), 0644); err != nil {
		return fmt.Errorf("failed to set Paper Pro boot partition: %w", err)
	}

	fmt.Printf("Successfully set Paper Pro next boot to version %s (partition %d)\n", targetVersion, newPart)
	fmt.Println("Reboot to boot into the selected partition.")

	return nil
}

func getVersionFromOSRelease() (string, error) {
	return getVersionFromPartitionPath("")
}

func getVersionFromPartitionPath(basePath string) (string, error) {
	// Try update.conf first (RELEASE_VERSION)
	updateConfPath := basePath + "/usr/share/remarkable/update.conf"
	if basePath == "" {
		updateConfPath = "/usr/share/remarkable/update.conf"
	}
	
	if version, err := readVersionFromFile(updateConfPath, "RELEASE_VERSION="); err == nil {
		return version, nil
	}

	// Fall back to os-release (IMG_VERSION)
	osReleasePath := basePath + "/etc/os-release"
	if basePath == "" {
		osReleasePath = "/etc/os-release"
	}
	
	if version, err := readVersionFromFile(osReleasePath, "IMG_VERSION="); err == nil {
		return version, nil
	}

	return "", fmt.Errorf("version not found in update.conf or os-release")
}

func readVersionFromFile(path, prefix string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if line contains the prefix (supports wildcard before RELEASE_VERSION)
		if strings.Contains(line, prefix) {
			// Find the position of the prefix and extract everything after it
			idx := strings.Index(line, prefix)
			if idx != -1 {
				version := line[idx+len(prefix):]
				// Remove quotes if present
				version = strings.Trim(version, `"`)
				return version, nil
			}
		}
	}

	return "", fmt.Errorf("%s not found in file %s", prefix, path)
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}
	return 0
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
