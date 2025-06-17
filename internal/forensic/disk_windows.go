package forensic

import (
	"fmt"
	"forensic-duplicator/internal/models"
	"golang.org/x/sys/windows"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGetDriveTypeW     = kernel32.NewProc("GetDriveTypeW")
	procDeviceIoControl   = kernel32.NewProc("DeviceIoControl")
	ioctlDiskGetDriveGeom = uint32(0x70000) // IOCTL_DISK_GET_DRIVE_GEOMETRY
)

func enumerateWindowsDisks() ([]models.DiskInfo, error) {
	var disks []models.DiskInfo

	// Enumerate Physical Drives
	for i := 0; i < 32; i++ {
		path := fmt.Sprintf(`\\.\\PhysicalDrive%d`, i)
		file, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			continue
		}
		file.Close()

		info, err := getWindowsDiskInfo(path)
		if err != nil {
			continue
		}

		// Attempt to exclude system drive by checking for C:\ in mounted volumes
		vols := getMountedVolumes(fmt.Sprintf("PhysicalDrive%d", i))
		isSystem := false
		for _, v := range vols {
			if strings.ToUpper(v) == "C:\\" {
				isSystem = true
				break
			}
		}
		if isSystem {
			continue // Skip system disk
		}

		disks = append(disks, *info)
	}

	// Enumerate logical partitions (e.g., C:, D:)
	for letter := 'A'; letter <= 'Z'; letter++ {
		drive := fmt.Sprintf("%c:\\", letter)
		driveType, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(drive))))
		if driveType == 3 { // DRIVE_FIXED
			info := models.DiskInfo{
				Path:         drive,
				Name:         fmt.Sprintf("Partition (%s)", drive),
				Size:         0,
				SectorSize:   512,
				FileSystem:   "NTFS/FAT",
				IsReadOnly:   false,
				IsRemovable:  false,
				SerialNumber: "Unknown",
				Model:        "Logical Partition",
			}
			disks = append(disks, info)
		}
	}

	return disks, nil
}

func getMountedVolumes(physicalDrive string) []string {
	// This is a placeholder. To implement correctly, query Windows WMI or use SetupAPI
	// For simplicity, return empty for now
	return []string{}
}

func getWindowsDiskInfo(diskPath string) (*models.DiskInfo, error) {
	file, err := syscall.CreateFile(
		syscall.StringToUTF16Ptr(diskPath),
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFile failed: %w", err)
	}
	defer syscall.CloseHandle(file)

	var geometry [24]byte
	var bytesReturned uint32
	r1, _, err := procDeviceIoControl.Call(
		uintptr(file),
		uintptr(ioctlDiskGetDriveGeom),
		0,
		0,
		uintptr(unsafe.Pointer(&geometry[0])),
		uintptr(len(geometry)),
		uintptr(unsafe.Pointer(&bytesReturned)),
		0,
	)
	if r1 == 0 {
		return nil, fmt.Errorf("DeviceIoControl failed: %w", err)
	}

	size := int64(*(*int64)(unsafe.Pointer(&geometry[8]))) * int64(*(*int32)(unsafe.Pointer(&geometry[0])))

	return &models.DiskInfo{
		Path:         diskPath,
		Name:         fmt.Sprintf("Physical Drive (%s)", diskPath),
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
	return nil
}

func validateTargetDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}
