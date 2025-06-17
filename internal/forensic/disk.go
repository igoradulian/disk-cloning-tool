package forensic

import (
	"fmt"
	"forensic-duplicator/internal/models"
	"os"
	"runtime"
	"strings"
)

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

// isPhysicalDrive checks if the given path represents a physical drive
func isPhysicalDrive(path string) bool {
	switch runtime.GOOS {
	case "windows":
		return strings.HasPrefix(strings.ToLower(path), `\\.\physicaldrive`)
	case "linux":
		return strings.HasPrefix(path, "/dev/sd") ||
			strings.HasPrefix(path, "/dev/hd") ||
			strings.HasPrefix(path, "/dev/nvme")
	case "darwin":
		return strings.HasPrefix(path, "/dev/disk")
	default:
		return false
	}
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

// Placeholder implementations - these would be in platform-specific files
func EnumerateWindowsDisks() ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	// Enumerate physical drives \\.\PhysicalDrive0, \\.\PhysicalDrive1, etc.
	for i := 0; i < 32; i++ {
		path := fmt.Sprintf(`\\.\PhysicalDrive%d`, i)

		// Try to open the drive
		file, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			continue // Drive doesn't exist or can't be accessed
		}
		file.Close()

		// Get disk information
		info, err := getWindowsDiskInfo(path)
		if err != nil {
			continue
		}

		disks = append(disks, *info)
	}

	return disks, nil
}

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

func validateSourceDiskPlatform(diskPath string, info *models.DiskInfo) error {
	// Platform-specific source validation
	return nil
}

func validateTargetDiskPlatform(diskPath string, info *models.DiskInfo) error {
	// Platform-specific target validation
	return nil
}
