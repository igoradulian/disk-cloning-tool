package test

import (
	"forensic-duplicator/internal/forensic"
	"testing"
)

func TestDiskInfo(t *testing.T) {
	//diks := []string{"\\\\.\\PHYSICALDRIVE0", "\\\\.\\PHYSICALDRIVE1", "\\\\.\\PHYSICALDRIVE2"} // Adjust these paths as needed for your system
	// Adjust this path as needed for your system
	disks, err := forensic.GetDisksPhysicalPaths()
	t.Log(disks)
	if err != nil {
		t.Fatalf("Failed to get physical disk paths: %v", err)
	}

	for _, disk := range disks {
		info, err := forensic.GetWindowsDiskInfo(disk)

		if err != nil {
			t.Fatalf("Failed to get disk info: %v", err)
		}
		t.Logf("Disk Path: %s, Name: %s, Model: %s, Size: %d GB", info.Path, info.Name, info.Model, info.Size/1024/1024/1024)
	}

}

func TestGetDisksPhysicalPaths(t *testing.T) {
	physicalPaths, err := forensic.GetDisksPhysicalPaths()
	if err != nil {
		t.Fatalf("Failed to get physical disk paths: %v", err)
	}

	t.Logf("Physical Disk Paths: %v", physicalPaths)
	if len(physicalPaths) == 0 {
		t.Fatal("No physical disk paths found")
	}
}

func TestGetDisksEnumerationTest(t *testing.T) {
	disks, err := forensic.EnumerateWindowsDisks()

	if err != nil {
		t.Fatalf("Failed to enumerate disks: %v", err)
	}

	t.Logf("Found %d disks", len(disks))

	for _, disk := range disks {
		t.Logf("Disk Path: %s, Name: %s, Model: %s, Size: %d GB, FileSystem: %s",
			disk.Path, disk.Name, disk.Model, disk.Size/1024/1024/1024, disk.FileSystem)
	}
}

func TestGetDisksLetters(t *testing.T) {
	// Get all physical disk paths
	disk, e := forensic.GetMountedDrives()
	if e != nil {
		t.Fatalf("Failed to get mounted drives: %v", e)
	}

	t.Logf("Mounted Drives: %v", disk)
}
