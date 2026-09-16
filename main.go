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

	"gopkg.in/yaml.v3"
)

type Config struct {
	BackupInterval string `yaml:"backup_interval"`
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

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

func runBackup() bool {

	backupDir := "backup"

	// Create backup directory if it doesn't exist
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		fmt.Println("Failed to create backup directory:", err)
		return false
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

	// Handle pg_dump failure
	if err != nil {
		fmt.Println("Backup failed!")

		if len(output) > 0 {
			fmt.Println("Error:", string(output))
		}

		// Remove incomplete backup file if it was created
		if _, fileErr := os.Stat(backupFile); fileErr == nil {
			removeErr := os.Remove(backupFile)

			if removeErr != nil {
				fmt.Println(
					"Warning: failed to remove incomplete backup:",
					removeErr,
				)
			} else {
				fmt.Println(
					"Removed incomplete backup:",
					backupFile,
				)
			}
		}

		return false
	}

	// Verify that the backup file exists
	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		fmt.Println(
			"Backup command succeeded, but backup file was not found:",
			err,
		)
		return false
	}

	// Verify that the backup file is not empty
	if fileInfo.Size() == 0 {
		fmt.Println("Backup failed: backup file is empty")

		removeErr := os.Remove(backupFile)
		if removeErr != nil {
			fmt.Println(
				"Warning: failed to remove empty backup:",
				removeErr,
			)
		}

		return false
	}

	// Backup was successful
	fmt.Println("Backup completed successfully!")
	fmt.Println("Backup size:", fileInfo.Size(), "bytes")

	// Keep only the latest 3 backups
	cleanupOldBackups(backupDir, 3)

	return true
}

func backupWithRetry() {
	const maxAttempts = 3
	const retryDelay = 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {

		fmt.Printf("Backup attempt %d of %d\n", attempt, maxAttempts)

		if runBackup() {
			fmt.Println("Backup process completed successfully.")
			return
		}

		if attempt < maxAttempts {
			fmt.Printf("Backup failed. Retrying in %v...\n", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	fmt.Println("Backup failed after all retry attempts.")
}

func main() {

	log.Println("PostgreSQL backup service started.")

	// Load configuration
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Parse backup interval
	interval, err := time.ParseDuration(cfg.BackupInterval)
	if err != nil {
		log.Fatalf("Invalid backup_interval '%s': %v", cfg.BackupInterval, err)
	}

	fmt.Println("Backup interval:", interval)

	// Run one backup immediately when the service starts
	backupWithRetry()

	// Run the backup at the configured interval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Keep the service running
	for {
		<-ticker.C
		backupWithRetry()
	}
}