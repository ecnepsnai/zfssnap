package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type FilesystemSnapshotConfig struct {
	Name          string `yaml:"name"`
	NumberDaily   int    `yaml:"daily"`
	NumberWeekly  int    `yaml:"weekly"`
	NumberMonthly int    `yaml:"monthly"`
}

var snapshotNamePattern = regexp.MustCompile(`@auto_(daily|weekly|monthly)_[0-9]+$`)

func printHelpAndExit() {
	fmt.Printf("Usage: %s <path to config file>\n", os.Args[0])
	os.Exit(1)
}

func main() {
	args := os.Args
	if len(args) <= 1 {
		printHelpAndExit()
	}

	configPath := os.Args[1]
	filesystemsToSnapshot := []FilesystemSnapshotConfig{}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %s", configPath, err.Error())
		os.Exit(1)
	}
	if err := yaml.Unmarshal(configBytes, &filesystemsToSnapshot); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config syntax: %s", err.Error())
		os.Exit(1)
	}

	for _, filesystem := range filesystemsToSnapshot {
		if err := processFilesystem(filesystem); err != nil {
			log.Fatalf("Error processing snapshots for filesystem %s: %s", filesystem.Name, err.Error())
		}
	}
}

func processFilesystem(filesystemConfig FilesystemSnapshotConfig) error {
	filesystems, err := ListFilesystems()
	if err != nil {
		return err
	}
	found := false
	for _, filesystem := range filesystems {
		if filesystem.Name == filesystemConfig.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("no filesystem found with name %s", filesystemConfig.Name)
	}

	snapshots, err := ListSnapshotsForFilesystem(filesystemConfig.Name)
	if err != nil {
		return err
	}

	dailySnapshotName, weeklySnapshotName, monthlySnapshotName := getSnapshotNamesForFilesystem()

	numberOfDailySnapshots := 0
	createDailySnapshot := true
	numberOfWeeklySnapshots := 0
	createWeeklySnapshot := true
	numberOfMonthlySnapshots := 0
	createMonthlySnapshot := true

	for _, snapshot := range snapshots {
		if strings.Contains(snapshot.Name, "_daily_") {
			numberOfDailySnapshots++
			if snapshot.Name == filesystemConfig.Name+"@"+dailySnapshotName {
				createDailySnapshot = false
			}
		}
		if strings.Contains(snapshot.Name, "_weekly_") {
			numberOfWeeklySnapshots++
			if snapshot.Name == filesystemConfig.Name+"@"+weeklySnapshotName {
				createWeeklySnapshot = false
			}
		}
		if strings.Contains(snapshot.Name, "_monthly_") {
			numberOfMonthlySnapshots++
			if snapshot.Name == filesystemConfig.Name+"@"+monthlySnapshotName {
				createMonthlySnapshot = false
			}
		}
	}

	if createDailySnapshot && filesystemConfig.NumberDaily > 0 {
		if err := CreateSnapshot(filesystemConfig.Name, dailySnapshotName); err != nil {
			return err
		}
		numberOfDailySnapshots++
	}
	if createWeeklySnapshot && filesystemConfig.NumberWeekly > 0 {
		if err := CreateSnapshot(filesystemConfig.Name, weeklySnapshotName); err != nil {
			return err
		}
		numberOfWeeklySnapshots++
	}
	if createMonthlySnapshot && filesystemConfig.NumberMonthly > 0 {
		if err := CreateSnapshot(filesystemConfig.Name, monthlySnapshotName); err != nil {
			return err
		}
		numberOfMonthlySnapshots++
	}

	if err := cleanupDailySnapshotForFilesystem(filesystemConfig); err != nil {
		return fmt.Errorf("cleanup daily: %s", err.Error())
	}
	if err := cleanupWeeklySnapshotForFilesystem(filesystemConfig); err != nil {
		return fmt.Errorf("cleanup weekly: %s", err.Error())
	}
	if err := cleanupMonthlySnapshotForFilesystem(filesystemConfig); err != nil {
		return fmt.Errorf("cleanup monthly: %s", err.Error())
	}

	return nil
}

func getSnapshotNamesForFilesystem() (string, string, string) {
	dailySnapshotName := fmt.Sprintf("auto_daily_%s", time.Now().Format("20060102"))
	_, weekN := time.Now().ISOWeek()
	weeklySnapshotName := fmt.Sprintf("auto_weekly_%s%d", time.Now().Format("2006"), weekN)
	monthlySnapshotName := fmt.Sprintf("auto_monthly_%s", time.Now().Format("200601"))

	return dailySnapshotName, weeklySnapshotName, monthlySnapshotName
}

func cleanupDailySnapshotForFilesystem(filesystemConfig FilesystemSnapshotConfig) error {
	snapshots, err := ListSnapshotsForFilesystem(filesystemConfig.Name)
	if err != nil {
		return err
	}

	numberOfSnapshots := 0

	for _, snapshot := range snapshots {
		if strings.Contains(snapshot.Name, "_daily_") {
			numberOfSnapshots++
		}
	}

	for numberOfSnapshots > filesystemConfig.NumberDaily {
		oldestName := ""
		oldestDate := uint64(18446744073709551615)
		oldestIdx := -1

		for i, snapshot := range snapshots {
			if !strings.Contains(snapshot.Name, "_daily_") {
				continue
			}

			parts := strings.Split(snapshot.Name, "_")
			date, err := strconv.ParseUint(parts[len(parts)-1], 10, 64)
			if err != nil {
				continue
			}
			if oldestDate > date {
				oldestDate = date
				oldestName = snapshot.Name
				oldestIdx = i
			}
		}

		if oldestIdx == -1 {
			return nil
		}

		n := strings.Split(oldestName, "@")
		if err := DeleteSnapshot(n[0], n[1]); err != nil {
			return err
		}
		numberOfSnapshots--
		snapshots = append(snapshots[:oldestIdx], snapshots[oldestIdx+1:]...)
	}
	return nil
}

func cleanupWeeklySnapshotForFilesystem(filesystemConfig FilesystemSnapshotConfig) error {
	snapshots, err := ListSnapshotsForFilesystem(filesystemConfig.Name)
	if err != nil {
		return err
	}

	numberOfSnapshots := 0

	for _, snapshot := range snapshots {
		if strings.Contains(snapshot.Name, "_weekly_") {
			numberOfSnapshots++
		}
	}

	for numberOfSnapshots > filesystemConfig.NumberWeekly {
		oldestName := ""
		oldestWeek := uint64(18446744073709551615)
		oldestIdx := -1

		for i, snapshot := range snapshots {
			if !strings.Contains(snapshot.Name, "_weekly_") {
				continue
			}

			parts := strings.Split(snapshot.Name, "_")
			date, err := strconv.ParseUint(parts[len(parts)-1], 10, 64)
			if err != nil {
				continue
			}
			if oldestWeek > date {
				oldestWeek = date
				oldestName = snapshot.Name
				oldestIdx = i
			}
		}

		if oldestIdx == -1 {
			return nil
		}

		n := strings.Split(oldestName, "@")
		if err := DeleteSnapshot(n[0], n[1]); err != nil {
			return err
		}
		numberOfSnapshots--
		snapshots = append(snapshots[:oldestIdx], snapshots[oldestIdx+1:]...)
	}
	return nil
}

func cleanupMonthlySnapshotForFilesystem(filesystemConfig FilesystemSnapshotConfig) error {
	snapshots, err := ListSnapshotsForFilesystem(filesystemConfig.Name)
	if err != nil {
		return err
	}

	numberOfSnapshots := 0

	for _, snapshot := range snapshots {
		if strings.Contains(snapshot.Name, "_monthly_") {
			numberOfSnapshots++
		}
	}

	for numberOfSnapshots > filesystemConfig.NumberMonthly {
		oldestName := ""
		oldestDate := uint64(18446744073709551615)
		oldestIdx := -1

		for i, snapshot := range snapshots {
			if !strings.Contains(snapshot.Name, "_monthly_") {
				continue
			}

			parts := strings.Split(snapshot.Name, "_")
			date, err := strconv.ParseUint(parts[len(parts)-1], 10, 64)
			if err != nil {
				continue
			}
			if oldestDate > date {
				oldestDate = date
				oldestName = snapshot.Name
				oldestIdx = i
			}
		}

		if oldestIdx == -1 {
			return nil
		}

		n := strings.Split(oldestName, "@")
		if err := DeleteSnapshot(n[0], n[1]); err != nil {
			return err
		}
		numberOfSnapshots--
		snapshots = append(snapshots[:oldestIdx], snapshots[oldestIdx+1:]...)
	}
	return nil
}
