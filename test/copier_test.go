package test

import (
	"context"
	"forensic-duplicator/internal/forensic"
	"golang.org/x/sys/windows"
	"os"
	"testing"
)

func TestOpenRawDisk_InvalidPath(t *testing.T) {
	_, err := forensic.OpenRawDisk("::invalid_path::", windows.GENERIC_READ)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestOpenRawDisk_ValidFile(t *testing.T) {

	f, err := forensic.OpenRawDisk("\\\\.\\PhysicalDrive1", windows.GENERIC_READ)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if f != nil {
		f.Close()
	}
}

func TestCopyDiskRawWithProgress(t *testing.T) {
	// Create a temp source file with some data
	srcFile, err := os.CreateTemp("", "srcdisk")
	if err != nil {
		t.Fatalf("failed to create temp source: %v", err)
	}
	defer os.Remove(srcFile.Name())
	srcData := []byte("testdata for raw disk copy")
	if _, err := srcFile.Write(srcData); err != nil {
		t.Fatalf("failed to write to source: %v", err)
	}
	srcFile.Close()

	// Create a temp target file
	tgtFile, err := os.CreateTemp("", "tgtdisk")
	if err != nil {
		t.Fatalf("failed to create temp target: %v", err)
	}
	defer os.Remove(tgtFile.Name())
	tgtFile.Close()

	ctx := context.Background()
	md5, sha256, err := forensic.CopyDiskRawWithProgress(ctx, "\\\\.\\PhysicalDrive1", []string{"\\\\.\\PhysicalDrive2"})
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	if md5 == "" || sha256 == "" {
		t.Error("expected non-empty hashes")
	}

	// Check that target file has the same content
	tgtData, err := os.ReadFile(tgtFile.Name())
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(tgtData) != string(srcData) {
		t.Errorf("target data mismatch: got %q, want %q", tgtData, srcData)
	}
}
