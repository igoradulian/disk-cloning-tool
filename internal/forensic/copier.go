package forensic

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"forensic-duplicator/internal/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type CopyStats struct {
	DriveName   string
	TotalFiles  int64
	CopiedFiles int64
	TotalBytes  int64
	CopiedBytes int64
	StartTime   time.Time
	EndTime     time.Time
	Errors      []string
	Completed   bool
	mutex       sync.RWMutex
}

type OverallStats struct {
	SourceDrive string
	DestDrives  []string
	DriveStats  map[string]*CopyStats
	TotalFiles  int64
	TotalBytes  int64
	StartTime   time.Time
	mutex       sync.RWMutex
}

const DefaultBufferSize = 4 * 1024 * 1024 // 4MB

type Copier struct {
	sourcePath  string
	targetPaths []string
	cancelCtx   context.Context
	cancelFunc  context.CancelFunc
	progress    *models.CopyProgress
	mu          sync.RWMutex
}

func NewCopier(sourcePath string, targetPaths []string) (*Copier, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Copier{
		sourcePath:  sourcePath,
		targetPaths: targetPaths,
		cancelCtx:   ctx,
		cancelFunc:  cancel,
		progress: &models.CopyProgress{
			SourceDisk:  sourcePath,
			TargetDisks: targetPaths,
			Status:      models.StatusIdle,
			StartTime:   time.Now(),
		},
	}, nil
}

func (c *Copier) Cancel() {
	c.cancelFunc()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.progress.Status = models.StatusCancelled
}

func (c *Copier) GetProgress() *models.CopyProgress {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.progress
}

func analyzeDrive(sourceDrive string, stats *CopyStats) error {
	return filepath.Walk(sourceDrive, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access
			return nil
		}

		if !info.IsDir() {
			stats.TotalFiles++
			stats.TotalBytes += info.Size()
		}

		// Show progress every 1000 files
		if stats.TotalFiles%1000 == 0 {
			fmt.Printf("\rAnalyzing... %d files found", stats.TotalFiles)
		}

		return nil
	})
}

func StartCopy(ctx context.Context, sourceDrive string, targetDrives []string) error {

	fmt.Printf(sourceDrive)
	fmt.Printf(targetDrives[0])
	overAllStats := &models.OverallStats{
		TotalFiles: 0,
		TotalBytes: 0,
		DriveStats: make(map[string]*models.CopyStats),
	}

	// Initialize stats for source drive
	copyStat := &models.CopyStats{}

	fmt.Println("\nAnalyzing source drive...")
	err := AnalyzeDrive(sourceDrive, overAllStats)
	if err != nil {
		fmt.Printf("Error analyzing drive: %v\n", err)
		return nil
	}

	if ctx.Err() != nil {
		runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
			"status":   models.StatusCancelled,
			"progress": 0,
		})
		return ctx.Err()
	}

	copyStat.TotalFiles = overAllStats.TotalFiles * int64(len(targetDrives))
	copyStat.TotalBytes = overAllStats.TotalBytes * int64(len(targetDrives))
	copyStat.StartTime = time.Now()

	runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
		"bytesCopied":   int64(0),
		"totalBytes":    copyStat.TotalBytes,
		"progress":      0,
		"speed":         0,
		"timeRemaining": 0,
		"status":        "Copying",
	})

	// Start parallel copying
	var wg sync.WaitGroup

	// Start copy operations for each destination drive
	for _, destDrive := range targetDrives {
		wg.Add(1)
		go func(dest string) {
			defer wg.Done()
			copyDrive(ctx, sourceDrive, dest, copyStat)
		}(destDrive)
	}

	wg.Wait()

	if ctx.Err() != nil {
		runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
			"status":   models.StatusCancelled,
			"progress": 0,
		})
		return ctx.Err()
	}

	copyStat.Mutex.RLock()
	bytesCopied := copyStat.CopiedBytes
	totalBytes := copyStat.TotalBytes
	copyStat.Mutex.RUnlock()

	finalProgress := 0.0
	if totalBytes > 0 {
		finalProgress = 100
	}

	runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
		"bytesCopied":   bytesCopied,
		"totalBytes":    totalBytes,
		"progress":      finalProgress,
		"speed":         0,
		"timeRemaining": 0,
		"status":        "Completed",
	})

	runtime.EventsEmit(ctx, "copy-complete", nil)
	return nil
}

/*func CopyDrive(ctx context.Context, sourceDrive, destDrive string, stats *CopyStats) error {
	return filepath.Walk(sourceDrive, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Access error: %s - %v", sourcePath, err))
			return nil // Continue with other files
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDrive, sourcePath)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDrive, relPath)

		if info.IsDir() {
			// Create directory
			err := os.MkdirAll(destPath, info.Mode())
			if err != nil {
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory %s: %v", destPath, err))
			}
			return nil
		}

		// Copy file
		//err = copyFile(sourcePath, destPath, info)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to copy %s: %v", sourcePath, err))
		} else {
			stats.CopiedFiles++
			stats.CopiedBytes += info.Size()
		}

		// Show progress
		if stats.CopiedFiles%100 == 0 {
			progress := float64(stats.CopiedBytes) / float64(stats.TotalBytes) * 100
			elapsed := time.Since(stats.StartTime)
			fmt.Printf("\rProgress: %.1f%% (%d/%d files) - %s elapsed",
				progress, stats.CopiedFiles, stats.TotalFiles, elapsed.Truncate(time.Second))
			speed := stats.CopiedBytes / int64(elapsed.Seconds())

			runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
				"bytesCopied":   stats.CopiedBytes,
				"totalBytes":    stats.TotalBytes,
				"progress":      progress,
				"speed":         speed,
				"timeRemaining": int(elapsed),
				"status":        "Copying",
			})
		}

		runtime.EventsEmit(ctx, "copy-complete", nil)

		return nil
	})
}*/

/*func copyFile(src, dest string, info os.FileInfo) error {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Set file permissions and timestamps
	err = os.Chmod(dest, info.Mode())
	if err != nil {
		return err
	}

	return os.Chtimes(dest, info.ModTime(), info.ModTime())
}*/

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func CopyDiskRawWithProgress(ctx context.Context, sourcePath string, targetPaths []string) (string, string, error) {
	const bufferSize = 4096 // 1MB blocks

	srcHandle, err := OpenRawDisk(sourcePath, windows.GENERIC_READ)
	if err != nil {
		return "", "", fmt.Errorf("failed to open source: %w", err)
	}
	defer srcHandle.Close()

	totalSize, err := GetWindowsDiskSize(srcHandle.Fd())
	if err != nil {
		return "", "", fmt.Errorf("failed to get source size: %w", err)
	}

	targetFiles := make([]*os.File, 0, len(targetPaths))
	for _, p := range targetPaths {
		t, err := OpenRawDisk(p, windows.GENERIC_WRITE)
		if err != nil {
			return "", "", fmt.Errorf("failed to open target: %w", err)
		}
		defer t.Close()
		targetFiles = append(targetFiles, t)
	}

	buf := make([]byte, bufferSize)
	hasherMD5 := md5.New()
	hasherSHA := sha256.New()
	var copied int64 = 0
	start := time.Now()

	for copied < totalSize {
		select {
		case <-ctx.Done():
			runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
				"status":   models.StatusCancelled,
				"progress": 0,
			})
			return "", "", ctx.Err()
		default:
		}

		toRead := bufferSize
		if remaining := totalSize - copied; remaining < int64(bufferSize) {
			toRead = int(remaining)
		}

		n, err := srcHandle.Read(buf[:toRead])
		if err != nil {
			return "", "", fmt.Errorf("read error: %w", err)
		}

		// Hash update
		hasherMD5.Write(buf[:n])
		hasherSHA.Write(buf[:n])

		// Write in parallel to all targets
		var wg sync.WaitGroup
		for _, target := range targetFiles {
			wg.Add(1)
			go func(t *os.File) {
				defer wg.Done()
				_, err := t.Write(buf[:n])
				if err != nil {
					runtime.EventsEmit(ctx, "error", fmt.Sprintf("Write error: %v", err))
				}
			}(target)
		}
		wg.Wait()

		copied += int64(n)
		elapsed := time.Since(start).Seconds()
		speed := float64(copied) / elapsed
		progress := (float64(copied) / float64(totalSize)) * 100
		eta := float64(totalSize-copied) / speed

		runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
			"bytesCopied":   copied,
			"totalBytes":    totalSize,
			"progress":      progress,
			"speed":         speed,
			"timeRemaining": int(eta),
			"status":        "Copying",
		})
	}

	runtime.EventsEmit(ctx, "copy-complete", nil)

	return hex.EncodeToString(hasherMD5.Sum(nil)), hex.EncodeToString(hasherSHA.Sum(nil)), nil
}

func OpenRawDisk(path string, access uint32) (*os.File, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %v", path, err)
	}

	handle, err := windows.CreateFile(
		pathPtr,
		access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		//windows.FILE_FLAG_NO_BUFFERING|windows.FILE_FLAG_WRITE_THROUGH,
		0,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open disk %s: %v", path, err)
	}
	return os.NewFile(uintptr(handle), path), nil
}

func GetWindowsDiskSize(handle uintptr) (int64, error) {
	const ioctlDiskGetLengthInfo = 0x7405C // IOCTL_DISK_GET_LENGTH_INFO
	var outLen uint32
	buf := make([]byte, 8)

	err := windows.DeviceIoControl(
		windows.Handle(handle),
		ioctlDiskGetLengthInfo,
		nil,
		0,
		&buf[0],
		uint32(len(buf)),
		&outLen,
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("DeviceIoControl failed: %w", err)
	}

	length := *(*int64)(unsafe.Pointer(&buf[0]))
	return length, nil
}
