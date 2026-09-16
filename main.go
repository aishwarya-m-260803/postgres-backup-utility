package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func cleanupOldBackups(backupDir string, keep int) {
	files, err := os.ReadDir(backupDir)
	if err != nil {
		fmt.Println("Failed to read backup directory:", err)
		return
	}

	var backups []os.DirEntry

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		// Only consider timestamped FinPay backup files
		if strings.HasPrefix(name, "finpay_") &&
			strings.HasSuffix(name, ".sql") &&
			name != "finpay_backup.sql" {

			backups = append(backups, file)
		}
	}

	// Sort newest first based on the timestamp in the filename
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	// Delete backups beyond the retention limit
	if len(backups) > keep {
		for _, file := range backups[keep:] {
			path := filepath.Join(backupDir, file.Name())

			err := os.Remove(path)
			if err != nil {
				fmt.Println("Failed to delete:", path, err)
				continue
			}

			fmt.Println("Deleted old backup:", file.Name())
		}
	}
}

func runBackup() {

	backupDir := "backup"

	// Create backup directory if it doesn't exist
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		fmt.Println("Failed to create backup directory:", err)
		return
	}

	// Generate timestamp
	timestamp := time.Now().Format("20060102_150405")

	// Create timestamped backup filename
	backupFile := filepath.Join(
		backupDir,
		fmt.Sprintf("finpay_%s.sql", timestamp),
	)

	fmt.Println("Starting PostgreSQL backup...")
	fmt.Println("Backup file:", backupFile)

	// Execute pg_dump
	cmd := exec.Command(
		"pg_dump",
		"-h", "localhost",
		"-p", "5432",
		"-U", "postgres",
		"finpay",
		"-f", backupFile,
	)

	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Backup failed!")
		fmt.Println(string(output))
		return
	}

	fmt.Println("Backup completed successfully!")

	// Keep only the latest 3 backups
	cleanupOldBackups(backupDir, 3)
}

func main() {

	// Log the service start time only once
	log.Println("PostgreSQL backup service started.")

	fmt.Println("Backup interval: 8 hours")

	// Run one backup immediately when the service starts
	runBackup()

	// Run the backup every 8 hours
	ticker := time.NewTicker(8 * time.Hour)
	defer ticker.Stop()

	// Keep the service running
	for {
		<-ticker.C
		runBackup()
	}
}