package forensic

import (
	"encoding/json"
	"fmt"
	"forensic-duplicator/internal/models"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func EnumerateWindowsDisks() ([]models.DiskInfo, error) {

	var diskInfos []models.DiskInfo
	pathSet := map[string]struct{}{}

	physicalPaths, err := GetDisksPhysicalPaths()
	if err == nil {
		for _, diskPath := range physicalPaths {
			infoOfPhysicalDisk, infoErr := GetWindowsDiskInfo(diskPath)
			if infoErr != nil {
				continue
			}
			if _, exists := pathSet[infoOfPhysicalDisk.Path]; exists {
				continue
			}
			pathSet[infoOfPhysicalDisk.Path] = struct{}{}
			diskInfos = append(diskInfos, *infoOfPhysicalDisk)
		}
	}

	mountedDrives, mountErr := GetMountedDrives()
	if mountErr != nil && len(diskInfos) == 0 {
		return nil, fmt.Errorf("failed to enumerate disks: %v", mountErr)
	}

	sectorSize := 512 // Default sector size, can be adjusted based on actual disk info
	for _, diskPath := range mountedDrives {
		if strings.TrimSpace(diskPath.Size) == "" {
			continue
		}

		size, sizeErr := strconv.ParseInt(diskPath.Size, 10, 64)
		if sizeErr != nil {
			return nil, fmt.Errorf("failed to convert disk size for %s: %v", diskPath.Caption, sizeErr)
		}

		driveType := 0
		if strings.TrimSpace(diskPath.DriveType) != "" {
			driveType, _ = strconv.Atoi(diskPath.DriveType)
		}
		if driveType == 5 { // CD-ROM
			continue
		}

		rootDiskPath, _, pathErr := normalizeWindowsDrivePath(diskPath.Caption)
		if pathErr != nil {
			continue
		}
		if _, exists := pathSet[rootDiskPath]; exists {
			continue
		}

		name := rootDiskPath
		if strings.TrimSpace(diskPath.VolumeName) != "" {
			name = fmt.Sprintf("%s (%s)", strings.TrimSpace(diskPath.VolumeName), rootDiskPath)
		}

		info := &models.DiskInfo{
			Path:         rootDiskPath,
			Name:         name,
			Size:         size,
			SectorSize:   sectorSize,
			SerialNumber: "Unknown",
			Model:        "Logical Volume",
			IsRemovable:  driveType == 2,
			IsReadOnly:   false,
			FileSystem:   "NTFS/FAT",
		}

		// Check if disk is read-only
		if info.IsReadOnly {
			continue // Skip read-only disks
		}

		pathSet[rootDiskPath] = struct{}{}
		diskInfos = append(diskInfos, *info)
	}

	return diskInfos, nil
}

func getMountedVolumes(physicalDrive string) []string {
	// This is a placeholder. To implement correctly, query Windows WMI or use SetupAPI
	// For simplicity, return empty for now
	return []string{}
}

func GetWindowsDiskInfo(diskPath string) (*models.DiskInfo, error) {
	trimmedPath := strings.TrimSpace(diskPath)
	if trimmedPath == "" {
		return nil, fmt.Errorf("disk path is empty")
	}

	isDriveLetter := len(trimmedPath) >= 2 && trimmedPath[1] == ':'
	openPath := trimmedPath
	infoPath := trimmedPath
	name := trimmedPath
	fileSystem := "Raw"
	model := "Unknown"

	if isDriveLetter {
		rootPath, letter, err := normalizeWindowsDrivePath(trimmedPath)
		if err != nil {
			return nil, err
		}
		openPath = fmt.Sprintf("\\\\.\\%s:", letter)
		infoPath = rootPath
		name = rootPath
		fileSystem = "NTFS/FAT"
		model = "Logical Volume"
	}

	pathUTF16, err := windows.UTF16PtrFromString(openPath)
	if err != nil {
		return nil, fmt.Errorf("invalid disk path: %v", err)
	}

	handle, err := windows.CreateFile(
		pathUTF16,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %v", openPath, err)
	}
	defer windows.CloseHandle(handle)

	// Define structure to receive disk length
	type GET_LENGTH_INFORMATION struct {
		Length int64
	}

	const IOCTL_DISK_GET_LENGTH_INFO = 0x7405c
	var lengthInfo GET_LENGTH_INFORMATION
	var bytesReturned uint32

	err = windows.DeviceIoControl(
		handle,
		IOCTL_DISK_GET_LENGTH_INFO,
		nil,
		0,
		(*byte)(unsafe.Pointer(&lengthInfo)),
		uint32(unsafe.Sizeof(lengthInfo)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk size for %s: %v", openPath, err)
	}
	// Default sector size
	sectorSize := 512

	size, err := strconv.ParseInt(strconv.FormatInt(lengthInfo.Length, 10), 10, 64)

	info := &models.DiskInfo{
		Path:         infoPath,
		Name:         name,
		Size:         size,
		SectorSize:   sectorSize,
		SerialNumber: "Unknown",
		Model:        model,
		IsRemovable:  false,
		IsReadOnly:   false,
		FileSystem:   fileSystem,
	}

	return info, nil
}

func GetMountedDrives() ([]models.DriveInfo, error) {

	data, err := runPowerShell("Get-CimInstance Win32_LogicalDisk | Select-Object Caption, Description, DeviceID, DriveType, FreeSpace, Size, VolumeName | ConvertTo-Json -Compress")
	if err != nil {
		return nil, err
	}

	type logicalDiskRow struct {
		Caption     string      `json:"Caption"`
		Description string      `json:"Description"`
		DeviceID    string      `json:"DeviceID"`
		DriveType   json.Number `json:"DriveType"`
		Size        json.Number `json:"Size"`
		FreeSpace   json.Number `json:"FreeSpace"`
		VolumeName  string      `json:"VolumeName"`
	}

	rows, err := parsePowerShellJSON[logicalDiskRow](data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse logical disks: %w", err)
	}

	var diskInfos []models.DriveInfo
	for _, row := range rows {
		caption := strings.TrimSpace(row.Caption)
		if caption == "" {
			continue
		}
		size := strings.TrimSpace(row.Size.String())
		freeSpace := strings.TrimSpace(row.FreeSpace.String())

		diskInfos = append(diskInfos, models.DriveInfo{
			Caption:     caption,
			Description: strings.TrimSpace(row.Description),
			DeviceID:    strings.TrimSpace(row.DeviceID),
			DriveType:   strings.TrimSpace(row.DriveType.String()),
			Size:        size,
			FreeSpace:   freeSpace,
			VolumeName:  strings.TrimSpace(row.VolumeName),
		})
	}
	return diskInfos, nil
}

func ExtractWindowsDriveLetter(path string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", fmt.Errorf("drive path is empty")
	}
	if len(trimmedPath) >= 2 && trimmedPath[1] == ':' {
		letter := strings.ToUpper(trimmedPath[:1])
		if letter[0] < 'A' || letter[0] > 'Z' {
			return "", fmt.Errorf("invalid drive letter in %s", path)
		}
		return letter, nil
	}
	return "", fmt.Errorf("path %s is not a drive-letter volume", path)
}

func FormatWindowsVolume(drivePath string) error {
	letter, err := ExtractWindowsDriveLetter(drivePath)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("Format-Volume -DriveLetter %s -FileSystem NTFS -Force -Confirm:$false", letter)
	if _, err := runPowerShell(cmd); err != nil {
		return fmt.Errorf("failed to format %s: %w", letter, err)
	}
	return nil
}

func normalizeWindowsDrivePath(path string) (string, string, error) {
	letter, err := ExtractWindowsDriveLetter(path)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s:\\", letter), letter, nil
}

func validateSourceDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}

func validateTargetDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}
