package forensic

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"forensic-duplicator/internal/models"
	"io"
	"os"
	"sync"
	"time"
)

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

func (c *Copier) Start(progressCb func(*models.CopyProgress), logCb func(string)) *models.CopyResult {
	result := &models.CopyResult{
		StartTime: time.Now(),
	}

	source, err := os.OpenFile(c.sourcePath, os.O_RDONLY, 0)
	if err != nil {
		result.Error = fmt.Errorf("failed to open source: %v", err)
		return result
	}
	defer source.Close()

	sourceInfo, _ := source.Stat()
	c.progress.TotalBytes = sourceInfo.Size()
	c.progress.Status = models.StatusCopying

	targets := make([]*os.File, 0, len(c.targetPaths))
	for _, path := range c.targetPaths {
		target, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			result.Error = fmt.Errorf("failed to open target %s: %v", path, err)
			return result
		}
		targets = append(targets, target)
		defer target.Close()
	}

	// Initialize hashers
	hMd5 := md5.New()
	hSha1 := sha1.New()
	hSha256 := sha256.New()

	buffer := make([]byte, DefaultBufferSize)
	var totalCopied int64
	start := time.Now()

	for {
		if c.cancelCtx.Err() != nil {
			result.Error = fmt.Errorf("copy cancelled")
			return result
		}

		n, err := source.Read(buffer)
		if n > 0 {
			slice := buffer[:n]

			// Update hashes
			hMd5.Write(slice)
			hSha1.Write(slice)
			hSha256.Write(slice)

			// Write in parallel
			wg := sync.WaitGroup{}
			for _, target := range targets {
				t := target
				data := make([]byte, n)
				copy(data, slice)
				wg.Add(1)
				go func() {
					defer wg.Done()
					t.Write(data)
				}()
			}
			wg.Wait()

			totalCopied += int64(n)

			c.mu.Lock()
			c.progress.BytesCopied = totalCopied
			c.progress.Progress = float64(totalCopied) / float64(c.progress.TotalBytes) * 100
			c.progress.Speed = int64(float64(totalCopied) / time.Since(start).Seconds())
			c.progress.TimeRemaining = int64(float64(c.progress.TotalBytes-totalCopied) / float64(c.progress.Speed))
			c.mu.Unlock()

			progressCb(c.GetProgress())
			logCb(fmt.Sprintf("Copied %d bytes...", totalCopied))
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			result.Error = fmt.Errorf("copy failed: %v", err)
			return result
		}
	}

	// Finalize result
	duration := time.Since(start).Seconds()
	result.BytesCopied = totalCopied
	result.Duration = int64(duration)
	result.AverageSpeed = int64(float64(totalCopied) / duration)
	result.MD5Hash = fmt.Sprintf("%x", hMd5.Sum(nil))
	result.SHA1Hash = fmt.Sprintf("%x", hSha1.Sum(nil))
	result.SHA256Hash = fmt.Sprintf("%x", hSha256.Sum(nil))
	result.Success = true
	result.EndTime = time.Now()

	c.mu.Lock()
	c.progress.Status = models.StatusCompleted
	c.progress.MD5Hash = result.MD5Hash
	c.progress.SHA1Hash = result.SHA1Hash
	c.progress.SHA256Hash = result.SHA256Hash
	c.mu.Unlock()

	progressCb(c.GetProgress())
	logCb("Copy completed successfully")
	return result
}
