package copyschemas

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file contents: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func copyFile(src, dstDir string, hash string) (string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	baseName := filepath.Base(src)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	dstPath := filepath.Join(dstDir, fmt.Sprintf("%s_%s%s", nameWithoutExt, hash[:8], ext))
	if strings.Contains(nameWithoutExt, hash[:8]) {
		dstPath = filepath.Join(dstDir, baseName)
	}

	if _, err := os.Stat(dstPath); err == nil {
		return "", nil
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file contents: %w", err)
	}

	return dstPath, nil
}

func getLibraryGraphDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to determine current file path")
	}

	currentDir := filepath.Dir(file)
	parentDir := filepath.Dir(currentDir)
	graphDir := filepath.Join(parentDir, "graph")

	return graphDir, nil
}

func CopyGraphqlSchemas(dstDir string) ([]string, error) {
	mainDir, err := getLibraryGraphDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine library main directory: %w", err)
	}

	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating destination directory: %w", err)
	}

	var copiedFiles []string

	err = filepath.WalkDir(mainDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Dir(path) == dstDir {
			return nil
		}

		if strings.HasSuffix(d.Name(), ".graphqls") {
			hash, err := getFileHash(path)
			if err != nil {
				return err
			}

			copiedPath, err := copyFile(path, dstDir, hash)
			if err != nil {
				return err
			}

			if copiedPath != "" {
				copiedFiles = append(copiedFiles, copiedPath)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directories: %w", err)
	}

	return copiedFiles, nil
}
