package main

import (
	"context"
	"fmt"
	"forensic-duplicator/internal/forensic"
	"forensic-duplicator/internal/models"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewApp creates a new App
func NewApp() *App {
	return &App{}
}

// OnStartup is called at app start
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	runtime.EventsEmit(ctx, "app-ready")
}

// GetAvailableDisks lists physical and mounted disks
func (a *App) GetAvailableDisks() ([]models.DiskInfo, error) {
	disks, err := forensic.EnumerateWindowsDisks()
	if err != nil {
		runtime.EventsEmit(a.ctx, "error", fmt.Sprintf("Failed to enumerate disks: %v", err))
		return nil, err
	}

	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Found %d available disks", len(disks)))
	return disks, nil
}

// StartForensicCopy starts the imaging process
func (a *App) StartForensicCopy(sourcePath string, targetPaths []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel != nil {
		return fmt.Errorf("a copy is already in progress")
	}
	if len(targetPaths) == 0 {
		return fmt.Errorf("no target drives specified")
	}

	copyCtx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel

	// Start copy in background
	go func() {
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()

		runtime.EventsEmit(a.ctx, "log", "Starting forensic copy...")

		err := forensic.StartCopy(copyCtx, sourcePath, targetPaths)
		if err != nil {
			if err == context.Canceled {
				runtime.EventsEmit(a.ctx, "log", "Copy cancelled")
				return
			}
			runtime.EventsEmit(a.ctx, "error", err.Error())
			return
		}

		// Final emit with hashes
		/*runtime.EventsEmit(a.ctx, "copy-complete", map[string]string{
			"md5Hash":    md5,
			"sha256Hash": sha256,
		})*/
	}()

	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Started copying from %s to %d targets", sourcePath, len(targetPaths)))
	return nil
}

// StartRawCopy starts the raw disk imaging process
func (a *App) StartRawCopy(sourcePath string, targetPaths []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel != nil {
		return fmt.Errorf("a copy is already in progress")
	}
	if len(targetPaths) == 0 {
		return fmt.Errorf("no target drives specified")
	}

	copyCtx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel

	go func() {
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()

		runtime.EventsEmit(a.ctx, "log", "Starting raw disk imaging...")

		md5Hash, sha256Hash, err := forensic.CopyDiskRawWithProgress(copyCtx, sourcePath, targetPaths)
		if err != nil {
			if err == context.Canceled {
				runtime.EventsEmit(a.ctx, "log", "Copy cancelled")
				return
			}
			runtime.EventsEmit(a.ctx, "error", err.Error())
			return
		}

		runtime.EventsEmit(a.ctx, "copy-progress", map[string]interface{}{
			"progress":   100,
			"status":     "Completed",
			"md5Hash":    md5Hash,
			"sha256Hash": sha256Hash,
		})

		runtime.EventsEmit(a.ctx, "copy-complete", map[string]string{
			"md5Hash":    md5Hash,
			"sha256Hash": sha256Hash,
		})
	}()

	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Started raw imaging from %s to %d targets", sourcePath, len(targetPaths)))
	return nil
}

// StopCopy cancels the active copy
func (a *App) StopCopy() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel == nil {
		return fmt.Errorf("no active copy to cancel")
	}

	a.cancel()
	runtime.EventsEmit(a.ctx, "copy-progress", map[string]interface{}{
		"status":   models.StatusCancelled,
		"progress": 0,
	})
	runtime.EventsEmit(a.ctx, "log", "Copy operation cancelled by user")
	a.cancel = nil
	return nil
}

// Validate source/target disks
func (a *App) ValidateSourceDisk(path string) error {
	return forensic.ValidateSourceDisk(path)
}

func (a *App) ValidateTargetDisk(path string) error {
	return forensic.ValidateTargetDisk(path)
}

func (a *App) GetDiskInfo(path string) (*models.DiskInfo, error) {
	return forensic.GetDiskInfo(path)
}

func (a *App) FormatTargetDisks(targetPaths []string, sourcePath string) error {
	if len(targetPaths) == 0 {
		return fmt.Errorf("no target drives specified")
	}

	systemDrive := os.Getenv("SystemDrive")
	systemLetter := ""
	if systemDrive != "" {
		if letter, err := forensic.ExtractWindowsDriveLetter(systemDrive); err == nil {
			systemLetter = letter
		}
	}

	sourceLetter := ""
	if sourcePath != "" {
		if letter, err := forensic.ExtractWindowsDriveLetter(sourcePath); err == nil {
			sourceLetter = letter
		}
	}

	for _, target := range targetPaths {
		letter, err := forensic.ExtractWindowsDriveLetter(target)
		if err != nil {
			return err
		}
		if sourceLetter != "" && letter == sourceLetter {
			return fmt.Errorf("refusing to format source drive %s", target)
		}
		if systemLetter != "" && letter == systemLetter {
			return fmt.Errorf("refusing to format system drive %s", target)
		}

		runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Formatting %s...", letter))
		if err := forensic.FormatWindowsVolume(target); err != nil {
			return err
		}
	}

	runtime.EventsEmit(a.ctx, "log", "Formatting completed.")
	return nil
}
