package main

import (
	"context"
	"fmt"
	"forensic-duplicator/internal/forensic"
	"forensic-duplicator/internal/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"sync"
)

// App struct
type App struct {
	ctx          context.Context
	activeCopier *forensic.Copier
	mu           sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// OnStartup is called when the app starts up
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Emit startup event
	runtime.EventsEmit(ctx, "app-ready")
}

// GetAvailableDisks returns list of available physical disks
func (a *App) GetAvailableDisks() ([]models.DiskInfo, error) {
	disks, err := forensic.EnumerateWindowsDisks()
	if err != nil {
		runtime.EventsEmit(a.ctx, "error", fmt.Sprintf("Failed to enumerate disks: %v", err))
		return nil, err
	}

	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Found %d available disks", len(disks)))
	return disks, nil
}

// StartForensicCopy begins the forensic copying process
func (a *App) StartForensicCopy(sourcePath string, targetPaths []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.activeCopier != nil {
		return fmt.Errorf("copy operation already in progress")
	}

	if len(targetPaths) == 0 {
		return fmt.Errorf("no target paths specified")
	}

	// Create new copier
	copier, err := forensic.NewCopier(sourcePath, targetPaths)
	if err != nil {
		runtime.EventsEmit(a.ctx, "error", fmt.Sprintf("Failed to create copier: %v", err))
		return err
	}

	a.activeCopier = copier

	// Start copying in goroutine
	go a.runCopyProcess()

	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf("Started forensic copy: %s -> %d targets", sourcePath, len(targetPaths)))
	return nil
}

// GetCopyProgress returns current copy progress
func (a *App) GetCopyProgress() *models.CopyProgress {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.activeCopier == nil {
		return &models.CopyProgress{
			Status: "Idle",
		}
	}

	return a.activeCopier.GetProgress()
}

// StopCopy cancels the current copy operation
func (a *App) StopCopy() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.activeCopier == nil {
		return fmt.Errorf("no active copy operation")
	}

	a.activeCopier.Cancel()
	runtime.EventsEmit(a.ctx, "log", "Copy operation cancelled by user")
	return nil
}

// ValidateSourceDisk checks if source disk is valid for forensic copying
func (a *App) ValidateSourceDisk(diskPath string) error {
	return forensic.ValidateSourceDisk(diskPath)
}

// ValidateTargetDisk checks if target disk is valid for forensic copying
func (a *App) ValidateTargetDisk(diskPath string) error {
	return forensic.ValidateTargetDisk(diskPath)
}

// GetDiskInfo returns detailed information about a specific disk
func (a *App) GetDiskInfo(diskPath string) (*models.DiskInfo, error) {
	return forensic.GetDiskInfo(diskPath)
}

// runCopyProcess handles the copy operation and emits progress events
func (a *App) runCopyProcess() {
	defer func() {
		a.mu.Lock()
		a.activeCopier = nil
		a.mu.Unlock()
	}()

	// Set up progress callback
	progressCallback := func(progress *models.CopyProgress) {
		runtime.EventsEmit(a.ctx, "copy-progress", progress)
	}

	// Set up log callback
	logCallback := func(message string) {
		runtime.EventsEmit(a.ctx, "log", message)
	}

	// Start the copy process
	result := a.activeCopier.Start(progressCallback, logCallback)

	if result.Error != nil {
		runtime.EventsEmit(a.ctx, "error", result.Error.Error())
		runtime.EventsEmit(a.ctx, "copy-error", result)
	} else {
		runtime.EventsEmit(a.ctx, "copy-complete", result)
	}
}
