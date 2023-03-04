package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const zfsBin = "/usr/sbin/zfs"

type Filesystem struct {
	Name       string
	Creation   string
	Used       string
	Avail      string
	Refer      string
	Mountpoint string
}

func ListSnapshots() ([]Filesystem, error) {
	return zfsList("snapshot")
}

func ListSnapshotsForFilesystem(filesystemName string) ([]Filesystem, error) {
	allSnapshots, err := zfsList("snapshot")
	if err != nil {
		return nil, err
	}
	snapshots := []Filesystem{}
	for _, snapshot := range allSnapshots {
		if !strings.Contains(snapshot.Name, filesystemName+"@") {
			continue
		}
		if !snapshotNamePattern.MatchString(snapshot.Name) {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func zfsList(t string) ([]Filesystem, error) {
	output, err := exec.Command(zfsBin, "list", "-t", t, "-H", "-o", "name,creation,used,avail,refer,mountpoint").CombinedOutput()
	if err != nil {
		message := strings.ReplaceAll(string(output), "\n", " ")
		log.Printf("Error listing zfs %s filesystems: %s", t, message)
		return nil, fmt.Errorf("zfs: %s", message)
	}

	filesystems := []Filesystem{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) != 6 {
			continue
		}

		filesystems = append(filesystems, Filesystem{
			Name:       parts[0],
			Creation:   parts[1],
			Used:       parts[2],
			Avail:      parts[3],
			Refer:      parts[4],
			Mountpoint: parts[5],
		})
	}

	return filesystems, nil
}

func ListFilesystems() ([]Filesystem, error) {
	return zfsList("filesystem")
}

func CreateSnapshot(fsName, ssName string) error {
	filesystems, err := ListFilesystems()
	if err != nil {
		return err
	}
	snapshots, err := ListSnapshots()
	if err != nil {
		return err
	}

	fsMatch := false
	for _, filesystem := range filesystems {
		if filesystem.Name == fsName {
			fsMatch = true
			break
		}
	}
	if !fsMatch {
		return fmt.Errorf("zfs: no filesystem with name %s", fsName)
	}
	for _, snapshot := range snapshots {
		if snapshot.Name == fmt.Sprintf("%s@%s", fsName, ssName) {
			return fmt.Errorf("zfs: snapshot already exists %s@%s", fsName, ssName)
		}
	}

	output, err := exec.Command(zfsBin, "snapshot", fmt.Sprintf("%s@%s", fsName, ssName)).CombinedOutput()
	if err != nil {
		message := strings.ReplaceAll(string(output), "\n", " ")
		log.Printf("Error creating snapshot %s@%s: %s", fsName, ssName, message)
		return fmt.Errorf("zfs: %s", message)
	}
	log.Printf("Created snapshot %s@%s", fsName, ssName)

	return nil
}

func DeleteSnapshot(fsName, ssName string) error {
	output, err := exec.Command(zfsBin, "destroy", fmt.Sprintf("%s@%s", fsName, ssName)).CombinedOutput()
	if err != nil {
		message := strings.ReplaceAll(string(output), "\n", " ")
		log.Printf("Error destroying snapshot %s@%s: %s", fsName, ssName, message)
		return fmt.Errorf("zfs: %s", message)
	}
	log.Printf("Destroyed snapshot %s@%s", fsName, ssName)

	return nil
}
