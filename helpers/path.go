package helpers

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return fs.ErrPermission
}

// LookPath searches for an executable named file in the
// directories named by the PATH environment variable.
// The result will be an absolute path
//
// This function ensures we never return our own binary
func LookPathNoSelf(file string) (string, error) {
	selfPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", fmt.Errorf("unable to calculate absolute path for self: %w", err)
	}
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path, err := filepath.Abs(filepath.Join(dir, file))
		if err != nil {
			return "", fmt.Errorf("unable to calculate absolute path for path candidate: %w", err)
		}
		if path == selfPath {
			continue
		}
		err = findExecutable(path)
		if err == nil {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}
