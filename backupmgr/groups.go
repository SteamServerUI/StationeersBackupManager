package backupmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getBackupGroups collects and groups backup files
func (m *BackupManager) getBackupGroups() ([]BackupGroup, error) {
	var files []os.DirEntry
	err := filepath.WalkDir(m.config.SafeBackupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, d)
		}
		return nil
	})
	if err != nil {
		// if the error contains no such file or directory, return nil but return a custom string intsted 	of the error
		if strings.Contains(err.Error(), "no such file or directory") || strings.Contains(err.Error(), "The system cannot find the file specified") {
			return nil, fmt.Errorf("save dir doesn't seem to exist (yet). Try starting the gameserver and click ↻ once it's up. If the Save folder exists and you still get this error, verify the 'Use New Terrain and Save System' setting. Detailed Error: %w", err)
		}
		return nil, fmt.Errorf("failed to walk safe backup dir: %w", err)
	}

	groups := make(map[int]BackupGroup)

	for _, file := range files {
		filename := file.Name()
		if !isValidBackupFile(filename) {
			continue
		}

		fullPath := filepath.Join(m.config.SafeBackupDir, filename)
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Parse index or assign synthetic index for .save files
		index := parseBackupIndex(filename, info.ModTime(), files)
		if index == -1 {
			continue
		}

		group := groups[index]
		group.Index = index
		group.ModTime = info.ModTime()

		if strings.HasSuffix(filename, ".save") {
			group.BinFile = fullPath
		} else {
			switch {
			case strings.HasSuffix(filename, ".bin"):
				group.BinFile = fullPath
			case strings.Contains(filename, "world(") && strings.HasSuffix(filename, ".xml"):
				group.XMLFile = fullPath
			case strings.Contains(filename, "world_meta(") && strings.HasSuffix(filename, ".xml"):
				group.MetaFile = fullPath
			}
		}

		groups[index] = group
	}

	var result []BackupGroup
	for _, group := range groups {
		// Include both old-style groups (all three files) and .save-based groups (just BinFile)
		if (group.BinFile != "" && group.XMLFile != "" && group.MetaFile != "") || (group.BinFile != "" && strings.HasSuffix(group.BinFile, ".save")) {
			result = append(result, group)
		}
	}

	return result, nil
}
