package models

import (
	"time"
)

// DiskInfo represents information about a physical disk
type DiskInfo struct {
	Path         string `json:"path"`         // Physical path (e.g., \\.\PhysicalDrive0)
	Name         string `json:"name"`         // Human-readable name
	Size         int64  `json:"size"`         // Size in bytes
	SectorSize   int    `json:"sectorSize"`   // Sector size in bytes
	SerialNumber string `json:"serialNumber"` // Drive serial number
	Model        string `json:"model"`        // Drive model
	IsRemovable  bool   `json:"isRemovable"`  // Whether disk is removable
	IsReadOnly   bool   `json:"isReadOnly"`   // Whether disk is read-only
	FileSystem   string `json:"fileSystem"`   // File system type (if applicable)
}

// CopyProgress represents the current state of a forensic copy operation
type CopyProgress struct {
	SourceDisk    string    `json:"sourceDisk"`    // Source disk path
	TargetDisks   []string  `json:"targetDisks"`   // Target disk paths
	BytesCopied   int64     `json:"bytesCopied"`   // Bytes copied so far
	TotalBytes    int64     `json:"totalBytes"`    // Total bytes to copy
	Progress      float64   `json:"progress"`      // Progress percentage (0-100)
	Speed         int64     `json:"speed"`         // Current speed in bytes/sec
	TimeRemaining int64     `json:"timeRemaining"` // Estimated time remaining in seconds
	Status        string    `json:"status"`        // Current status
	StartTime     time.Time `json:"startTime"`     // When copy started
	MD5Hash       string    `json:"md5Hash"`       // MD5 hash of copied data
	SHA256Hash    string    `json:"sha256Hash"`    // SHA-256 hash of copied data
	SHA1Hash      string    `json:"sha1Hash"`      // SHA-1 hash of copied data
	Error         string    `json:"error"`         // Error message if any
}

// CopyResult represents the final result of a forensic copy operation
type CopyResult struct {
	Success      bool         `json:"success"`      // Whether copy was successful
	BytesCopied  int64        `json:"bytesCopied"`  // Total bytes copied
	Duration     int64        `json:"duration"`     // Duration in seconds
	AverageSpeed int64        `json:"averageSpeed"` // Average speed in bytes/sec
	MD5Hash      string       `json:"md5Hash"`      // MD5 hash of source data
	SHA256Hash   string       `json:"sha256Hash"`   // SHA-256 hash of source data
	SHA1Hash     string       `json:"sha1Hash"`     // SHA-1 hash of source data
	TargetHashes []TargetHash `json:"targetHashes"` // Hash verification for each target
	StartTime    time.Time    `json:"startTime"`    // When copy started
	EndTime      time.Time    `json:"endTime"`      // When copy finished
	Error        error        `json:"error"`        // Error if copy failed
}

// TargetHash represents hash verification for a specific target
type TargetHash struct {
	TargetPath string `json:"targetPath"` // Path to target
	MD5Hash    string `json:"md5Hash"`    // MD5 hash of target
	SHA256Hash string `json:"sha256Hash"` // SHA-256 hash of target
	SHA1Hash   string `json:"sha1Hash"`   // SHA-1 hash of target
	Verified   bool   `json:"verified"`   // Whether hashes match source
}

// CopyOptions represents options for forensic copying
type CopyOptions struct {
	BufferSize      int  `json:"bufferSize"`      // Buffer size for copying (bytes)
	VerifyHashes    bool `json:"verifyHashes"`    // Whether to verify hashes after copy
	SkipBadSectors  bool `json:"skipBadSectors"`  // Whether to skip bad sectors
	CreateLogFile   bool `json:"createLogFile"`   // Whether to create detailed log file
	CompressTargets bool `json:"compressTargets"` // Whether to compress target images
}

// LogEntry represents a log entry for forensic operations
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"` // When log entry was created
	Level     string    `json:"level"`     // Log level (INFO, WARN, ERROR, DEBUG)
	Message   string    `json:"message"`   // Log message
	Source    string    `json:"source"`    // Source of log entry
}

// ForensicReport represents a comprehensive forensic report
type ForensicReport struct {
	CaseID           string       `json:"caseId"`           // Case identifier
	Examiner         string       `json:"examiner"`         // Examiner name
	Timestamp        time.Time    `json:"timestamp"`        // Report timestamp
	SourceInfo       DiskInfo     `json:"sourceInfo"`       // Source disk information
	TargetInfo       []DiskInfo   `json:"targetInfo"`       // Target disk information
	CopyResult       CopyResult   `json:"copyResult"`       // Copy operation results
	HashVerification []TargetHash `json:"hashVerification"` // Hash verification results
	Notes            string       `json:"notes"`            // Additional notes
}

// Status constants for copy operations
const (
	StatusIdle         = "Idle"
	StatusInitializing = "Initializing"
	StatusCopying      = "Copying"
	StatusVerifying    = "Verifying"
	StatusCompleted    = "Completed"
	StatusError        = "Error"
	StatusCancelled    = "Cancelled"
	StatusPaused       = "Paused"
)

// Log level constants
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)
