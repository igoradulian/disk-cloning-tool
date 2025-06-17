package forensic

import (
	"fmt"
	"forensic-duplicator/internal/models"
	"os"
	"syscall"
)

// getLinuxDiskInfo provides minimal disk info for a Linux block device
func getLinuxDiskInfo(diskPath string) (*models.DiskInfo, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open disk %s: %w", diskPath, err)
	}
	defer file.Close()

	// Get disk size using ioctl BLKGETSIZE64
	var size int64
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), 0x80081272, uintptr(unsafe.Pointer(&size)))
	if errno != 0 {
		return nil, fmt.Errorf("ioctl BLKGETSIZE64 failed: %v", errno)
	}

	return &models.DiskInfo{
		Path:         diskPath,
		Name:         fmt.Sprintf("Linux Block Device (%s)", diskPath),
		Size:         size,
		SectorSize:   512,
		SerialNumber: "Unknown",
		Model:        "Unknown",
		IsRemovable:  false,
		IsReadOnly:   false,
		FileSystem:   "Raw",
	}, nil
}

func validateSourceDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil // Additional validations can be added later
}

func validateTargetDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}

func enumerateLinuxDisks() ([]models.DiskInfo, error) {
	// Placeholder: Proper implementation would scan /sys/block or use lsblk
	return nil, fmt.Errorf("Linux disk enumeration not implemented yet")
}
