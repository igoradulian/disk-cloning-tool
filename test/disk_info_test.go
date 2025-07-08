package test

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestDiskInfoWithCommandLine(t *testing.T) {
	// This test is a placeholder for command line disk info retrieval
	// You can implement command line disk info retrieval logic here
	cmd := exec.Command("wmic", "logicaldisk", "get", "caption,", "description,", "deviceid,", "volumename,", "drivetype,", "size,", "freespace")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}

func TestDiskInfoWithCommandLineAndFilter(t *testing.T) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	// Get a handle to the GetLogicalDrives function
	getLogicalDrivesProc := kernel32.NewProc("GetLogicalDrives")

	// Get a handle to the GetDriveTypeW function
	getDriveTypeWProc := kernel32.NewProc("GetDriveTypeW")

	// Call GetLogicalDrives to get a bitmask of available drives
	ret, _, err := getLogicalDrivesProc.Call()
	if ret == 0 {
		log.Fatalf("Error calling GetLogicalDrives: %v", err)
	}

	driveMask := uint32(ret)

	// Iterate through the bitmask to identify available drives
	for i := 0; i < 26; i++ {
		if (driveMask>>uint(i))&1 == 1 {
			driveLetter := byte('A' + i)
			drivePath := fmt.Sprintf("%c:\\", driveLetter)

			// Convert drive path to UTF-16 for GetDriveTypeW
			drivePathUTF16, err := syscall.UTF16PtrFromString(drivePath)
			if err != nil {
				log.Printf("Error converting string to UTF16: %v", err)
				continue
			}

			// Call GetDriveTypeW to determine the drive type
			driveType, _, _ := getDriveTypeWProc.Call(uintptr(unsafe.Pointer(drivePathUTF16)))

			// Check if it's a fixed drive (e.g., local hard drive)
			if driveType == 3 { // DRIVE_FIXED
				fmt.Printf("Mounted Drive: %s\n", drivePath)
			}
		}
	}
}

func TestConvertDriveInfoIntoCSV(t *testing.T) {
	// This function is a placeholder for converting drive info into CSV format
	// You can implement the logic to retrieve drive info and convert it to CSV here
	cmd := exec.Command("wmic", "logicaldisk", "get", "caption,", "description,", "deviceid,", "volumename,", "drivetype,", "size,", "freespace")
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatalf("Error creating stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	var records [][]string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Basic parsing: split by whitespace. Adjust based on your command's output format.
		fields := strings.Fields(line)
		if len(fields) > 0 { // Avoid empty lines
			records = append(records, fields)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading stdout: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatalf("Command finished with error: %v", err)
	}

	// 2. Write the captured data to a CSV file
	file, err := os.Create("output.csv")
	if err != nil {
		log.Fatalf("Error creating CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush() // Ensure all buffered data is written

	if err := writer.WriteAll(records); err != nil {
		log.Fatalf("Error writing to CSV: %v", err)
	}

	fmt.Println("Data successfully written to output.csv")

}
