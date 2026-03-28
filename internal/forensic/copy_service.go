package forensic

import (
	"context"
	"fmt"
	"forensic-duplicator/internal/models"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func AnalyzeDrive(sourceDrive string, overallStats *models.OverallStats) error {
	return filepath.Walk(sourceDrive, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			overallStats.Mutex.Lock()
			overallStats.TotalFiles++
			overallStats.TotalBytes += info.Size()

			// Update each destination drive's total counts
			for _, stats := range overallStats.DriveStats {
				stats.TotalFiles = overallStats.TotalFiles
				stats.TotalBytes = overallStats.TotalBytes
			}
			overallStats.Mutex.Unlock()
		}

		// Show progress every 1000 files
		if overallStats.TotalFiles%1000 == 0 {
			fmt.Printf("\rAnalyzing... %d files found", overallStats.TotalFiles)
		}

		return nil
	})
}

func copyDrive(ctx context.Context, sourceDrive, destDrive string, stats *models.CopyStats) {
	defer func() {
		stats.Mutex.Lock()
		stats.Completed = true
		stats.EndTime = time.Now()
		stats.Mutex.Unlock()
	}()

	lastEmit := time.Now()

	walkErr := filepath.Walk(sourceDrive, func(sourcePath string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			stats.Mutex.Lock()
			stats.Errors = append(stats.Errors, fmt.Sprintf("Access error: %s - %v", sourcePath, err))
			stats.Mutex.Unlock()
			return nil
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
				stats.Mutex.Lock()
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to create directory %s: %v", destPath, err))
				stats.Mutex.Unlock()
			}
			return nil
		}

		// Copy file with progress updates
		err = copyFileWithProgress(ctx, sourcePath, destPath, info, stats, &lastEmit)

		stats.Mutex.Lock()
		if err != nil {
			if err != context.Canceled {
				stats.Errors = append(stats.Errors, fmt.Sprintf("Failed to copy %s: %v", sourcePath, err))
			}
		} else {
			stats.CopiedFiles++
		}
		stats.Mutex.Unlock()

		return err
	})

	if walkErr != nil && walkErr != context.Canceled {
		stats.Mutex.Lock()
		stats.Errors = append(stats.Errors, fmt.Sprintf("Walk error: %v", walkErr))
		stats.Mutex.Unlock()
	}
}

func emitCopyProgress(ctx context.Context, stats *models.CopyStats) {
	stats.Mutex.RLock()
	copiedBytes := stats.CopiedBytes
	totalBytes := stats.TotalBytes
	copiedFiles := stats.CopiedFiles
	totalFiles := stats.TotalFiles
	startTime := stats.StartTime
	stats.Mutex.RUnlock()

	elapsed := time.Since(startTime)
	var progress float64
	var speed int64
	var eta int64
	if totalBytes > 0 && elapsed > 0 {
		progress = float64(copiedBytes) / float64(totalBytes) * 100
		speed = int64(float64(copiedBytes) / elapsed.Seconds())
		if speed > 0 {
			eta = int64(float64(totalBytes-copiedBytes) / float64(speed))
		}
	}

	runtime.EventsEmit(ctx, "copy-progress", map[string]interface{}{
		"bytesCopied":   copiedBytes,
		"totalBytes":    totalBytes,
		"progress":      progress,
		"speed":         speed,
		"timeRemaining": eta,
		"status":        "Copying",
		"filesCopied":   copiedFiles,
		"totalFiles":    totalFiles,
	})
}

func copyFileWithProgress(ctx context.Context, src, dest string, info os.FileInfo, stats *models.CopyStats, lastEmit *time.Time) error {
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

	buf := make([]byte, 1024*1024)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		readBytes, readErr := srcFile.Read(buf)
		if readBytes > 0 {
			if _, writeErr := destFile.Write(buf[:readBytes]); writeErr != nil {
				return writeErr
			}

			stats.Mutex.Lock()
			stats.CopiedBytes += int64(readBytes)
			stats.Mutex.Unlock()

			if time.Since(*lastEmit) >= 300*time.Millisecond {
				*lastEmit = time.Now()
				emitCopyProgress(ctx, stats)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	// Set file permissions and timestamps
	err = os.Chmod(dest, info.Mode())
	if err != nil {
		return err
	}

	err = os.Chtimes(dest, info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}

	emitCopyProgress(ctx, stats)
	return nil
}

func progressMonitor(overallStats *models.OverallStats, stop chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			//printProgressUpdate(overallStats)
		}
	}
}

func printProgressUpdate(overallStats *models.OverallStats) {
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println("Multi-Drive Copy Progress")
	fmt.Println("=========================")

	elapsed := time.Since(overallStats.StartTime)
	fmt.Printf("Overall Elapsed Time: %s\n\n", elapsed.Truncate(time.Second))

	overallStats.Mutex.RLock()
	for _, drive := range overallStats.DestDrives {
		stats := overallStats.DriveStats[drive]
		stats.Mutex.RLock()

		var progress float64
		if stats.TotalBytes > 0 {
			progress = float64(stats.CopiedBytes) / float64(stats.TotalBytes) * 100
		}

		status := "In Progress"
		if stats.Completed {
			status = "Completed"
		}

		var avgSpeed float64
		driveElapsed := time.Since(stats.StartTime)
		if driveElapsed.Seconds() > 0 && stats.CopiedBytes > 0 {
			avgSpeed = float64(stats.CopiedBytes) / driveElapsed.Seconds()
		}

		fmt.Printf("Drive %s [%s]\n", drive, status)
		fmt.Printf("  Progress: %.1f%% (%d/%d files)\n", progress, stats.CopiedFiles, stats.TotalFiles)
		fmt.Printf("  Data: %s / %s\n", formatBytes(stats.CopiedBytes), formatBytes(stats.TotalBytes))
		fmt.Printf("  Speed: %s/s\n", formatBytes(int64(avgSpeed)))
		fmt.Printf("  Errors: %d\n\n", len(stats.Errors))

		stats.Mutex.RUnlock()
	}
	overallStats.Mutex.RUnlock()
}
