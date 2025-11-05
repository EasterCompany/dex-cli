package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckFileExists checks if a file exists at the given path.
func CheckFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureDirectory ensures that a directory exists at the given path.
func EnsureDirectory(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// EnsureSymlink creates a symbolic link, ensuring the target exists.
func EnsureSymlink(target, linkName string) error {
	// Check if the target exists
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("cannot create symlink: target '%s' does not exist", target)
	}

	// Remove existing link if it exists
	if _, err := os.Lstat(linkName); err == nil {
		if err := os.Remove(linkName); err != nil {
			return fmt.Errorf("failed to remove existing symlink '%s': %w", linkName, err)
		}
	}

	// Create the new symlink
	return os.Symlink(target, linkName)
}

// FindProjectRoot finds the root of the project by looking for go.mod.
func FindProjectRoot(startPath string) (string, error) {
	currentPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentPath, nil
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			// Reached the filesystem root
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		currentPath = parentPath
	}
}
