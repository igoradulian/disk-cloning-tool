package test

import (
	"forensic-duplicator/internal/forensic"
	"testing"
)

func TestDiskList(t *testing.T) {

	disklist, err := forensic.EnumerateWindowsDisks()
	t.Log("Disk list test executed successfully")
	if err != nil {
		t.Fatalf("Failed to list disks: %v", err)
	}

	for _, disk := range disklist {
		t.Logf("Disk Name: %s, Model: %s, Size: %d GB", disk.Name, disk.Model, disk.Size)
	}

}
