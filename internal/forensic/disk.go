package forensic

import (
	"bytes"
	"fmt"
	"forensic-duplicator/internal/models"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func GetDisksPhysicalPaths() ([]string, error) {
	cmd := exec.Command("powershell", "-Command", "Get-WmiObject Win32_DiskDrive | Select-Object -ExpandProperty DeviceID")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute PowerShell: %w", err)
	}
	lines := strings.Split(out.String(), "\n")
	var paths []string
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// EnumerateDisks returns a list of all available physical disks
func EnumerateDisks() ([]models.DiskInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return EnumerateWindowsDisks()
	case "linux":
		return EnumerateLinuxDisks()
	case "darwin":
		return enumerateDarwinDisks()
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// GetDiskInfo returns detailed information about a specific disk
func GetDiskInfo(diskPath string) (*models.DiskInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsDiskInfo(diskPath)
	case "linux":
		return getLinuxDiskInfo(diskPath)
	case "darwin":
		return getDarwinDiskInfo(diskPath)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// ValidateSourceDisk checks if a disk is suitable as a source for forensic copying
func ValidateSourceDisk(diskPath string) error {
	// Check if disk exists and is accessible
	file, err := os.OpenFile(diskPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot access source disk %s: %v", diskPath, err)
	}
	defer file.Close()

	// Get disk info
	info, err := GetDiskInfo(diskPath)
	if err != nil {
		return fmt.Errorf("cannot get disk info for %s: %v", diskPath, err)
	}

	// Check if disk has data
	if info.Size == 0 {
		return fmt.Errorf("source disk %s appears to be empty", diskPath)
	}

	// Additional platform-specific validations
	return validateSourceDiskPlatform(diskPath, info)
}

// ValidateTargetDisk checks if a disk is suitable as a target for forensic copying
func ValidateTargetDisk(diskPath string) error {
	// Check if disk exists and is writable
	file, err := os.OpenFile(diskPath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot access target disk %s for writing: %v", diskPath, err)
	}
	defer file.Close()

	// Get disk info
	info, err := GetDiskInfo(diskPath)
	if err != nil {
		return fmt.Errorf("cannot get disk info for %s: %v", diskPath, err)
	}

	// Check if disk is read-only
	if info.IsReadOnly {
		return fmt.Errorf("target disk %s is read-only", diskPath)
	}

	// Additional platform-specific validations
	return validateTargetDiskPlatform(diskPath, info)
}

// getDiskSize returns the size of a disk in bytes
func getDiskSize(diskPath string) (int64, error) {
	file, err := os.OpenFile(diskPath, os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Seek to end to get size
	size, err := file.Seek(0, 2) // SEEK_END
	if err != nil {
		return 0, err
	}

	return size, nil
}

// Platform-specific implementations would be in separate files:
// disk_windows.go, disk_linux.go, disk_darwin.go

func EnumerateLinuxDisks() ([]models.DiskInfo, error) {
	// Implementation would parse /proc/partitions, /sys/block/, etc.
	return []models.DiskInfo{}, fmt.Errorf("Linux disk enumeration not implemented")
}

func enumerateDarwinDisks() ([]models.DiskInfo, error) {
	// Implementation would use diskutil or similar
	return []models.DiskInfo{}, fmt.Errorf("macOS disk enumeration not implemented")
}

func getWindowsDiskInfo(diskPath string) (*models.DiskInfo, error) {
	size, err := getDiskSize(diskPath)
	if err != nil {
		return nil, err
	}

	return &models.DiskInfo{
		Path:         diskPath,
		Name:         fmt.Sprintf("Physical Drive (%s)", diskPath),
		Size:         size,
		SectorSize:   512, // Default sector size
		SerialNumber: "Unknown",
		Model:        "Unknown",
		IsRemovable:  false,
		IsReadOnly:   false,
		FileSystem:   "Raw",
	}, nil
}

func getLinuxDiskInfo(diskPath string) (*models.DiskInfo, error) {
	return nil, fmt.Errorf("Linux disk info not implemented")
}

func getDarwinDiskInfo(diskPath string) (*models.DiskInfo, error) {
	return nil, fmt.Errorf("macOS disk info not implemented")
}
