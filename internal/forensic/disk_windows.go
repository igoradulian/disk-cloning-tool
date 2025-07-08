package forensic

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"forensic-duplicator/internal/models"
	"golang.org/x/sys/windows"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"
)

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGetDriveTypeW     = kernel32.NewProc("GetDriveTypeW")
	procDeviceIoControl   = kernel32.NewProc("DeviceIoControl")
	ioctlDiskGetDriveGeom = uint32(0x70000) // IOCTL_DISK_GET_DRIVE_GEOMETRY
)

/*func EnumerateWindowsDisks() ([]models.DiskInfo, error) {
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
}*/

func EnumerateWindowsDisks() ([]models.DiskInfo, error) {

	//disks, err := GetDisksPhysicalPaths()
	disks, err := GetMountedDrives()
	if err != nil {
		return nil, fmt.Errorf("failed to get physical disk paths: %v", err)
	}

	var diskInfos []models.DiskInfo
	sectorSize := 512 // Default sector size, can be adjusted based on actual disk info

	for _, diskPath := range disks[1:] {
		//info, err := GetWindowsDiskInfo(diskPath)

		size, err := strconv.ParseInt(diskPath.Size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert disk size for %s: %v", size, err)
		}

		info := &models.DiskInfo{
			Path:         fmt.Sprintf("Physical Drive (%s)", diskPath.Caption),
			Name:         diskPath.Description,
			Size:         size,
			SectorSize:   sectorSize,
			SerialNumber: "Unknown",
			Model:        "Unknown",
			IsRemovable:  false,
			IsReadOnly:   false,
			FileSystem:   "Raw",
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get disk info for %s: %v", diskPath, err)
		}

		// Check if disk is read-only
		if info.IsReadOnly {
			continue // Skip read-only disks
		}

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
	isDriveLetter := len(diskPath) == 3 && diskPath[1] == ':' && diskPath[2] == '\\'
	pathUTF16, err := windows.UTF16PtrFromString(diskPath)
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
		return nil, fmt.Errorf("cannot open %s: %v", diskPath, err)
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
		return nil, fmt.Errorf("failed to get disk size for %s: %v", diskPath, err)
	}
	// Default sector size
	sectorSize := 512

	info := &models.DiskInfo{
		Path:         fmt.Sprintf("Physical Drive (%s)", diskPath),
		Name:         "",
		Size:         lengthInfo.Length,
		SectorSize:   sectorSize,
		SerialNumber: "Unknown",
		Model:        "Unknown",
		IsRemovable:  false,
		IsReadOnly:   false,
		FileSystem:   "Raw",
	}

	// Enumerate logical partitions (e.g., C:, D:	)

	if isDriveLetter {
		info.Path = diskPath
		info.Name = fmt.Sprintf("Partition (%s)", diskPath)
		info.FileSystem = "NTFS/FAT"
		info.Model = "Logical Partition"
	} else {
		info.Path = diskPath
		info.Name = fmt.Sprintf("Physical Drive (%s)", diskPath)
	}

	return info, nil
}

func GetMountedDrives() ([]models.DriveInfo, error) {

	cmd := exec.Command("wmic", "logicaldisk", "get", "caption,", "description,", "size,", "freespace")
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatalf("Error creating stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Step 2: Convert table output to CSV format
	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		// Basic parsing: split by whitespace. Adjust based on your command's output format.
		lines = append(lines, line)
	}

	// Remove empty lines
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}
	if len(cleanLines) < 2 {
		return nil, fmt.Errorf("unexpected output format")
	}

	header := strings.Fields(cleanLines[0])
	cleanLines = cleanLines[1:] // Header
	// Remaining lines are data
	var records [][]string
	records = append(records, header)

	for _, line := range lines {
		fields := strings.Fields(line)

		// Fix misaligned columns if Name has spaces
		if len(fields) > len(header) {
			diff := len(fields) - len(header) + 1
			name := strings.Join(fields[:diff], " ")
			rest := fields[diff:]
			fields = append([]string{name}, rest...)
		}

		if len(fields) == len(header) {
			records = append(records, fields)
		}
	}

	// Step 3: Convert to CSV format in memory
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	for _, rec := range records {
		_ = writer.Write(rec)
	}
	writer.Flush()

	// Step 4: Read CSV into structs
	reader := csv.NewReader(&buf)
	rawCSVdata, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	headers := rawCSVdata[0]
	var diskInfos []models.DriveInfo
	for _, row := range rawCSVdata[1:] {
		item := map[string]string{}
		for i, val := range row {
			item[headers[i]] = val
		}

		// Marshal to JSON then unmarshal into struct
		jsonData, _ := json.Marshal(item)
		var pi models.DriveInfo
		_ = json.Unmarshal(jsonData, &pi)
		diskInfos = append(diskInfos, pi)
	}

	return diskInfos, nil
}

func validateSourceDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}

func validateTargetDiskPlatform(diskPath string, info *models.DiskInfo) error {
	return nil
}
