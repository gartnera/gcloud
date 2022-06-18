package helpers

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

var errPreambleMismatch = errors.New("preamble mismatch")

func findExecutable(file string, preamble string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		f, _ := os.Open(file)
		preambleBytes := make([]byte, len(preamble))
		_, err := f.Read(preambleBytes)
		if err != nil {
			return fmt.Errorf("unable to detect filetype: %w", err)
		}
		if preamble != string(preambleBytes) {
			return errPreambleMismatch
		}
		return nil
	}
	return fs.ErrPermission
}

// LookPath searches for an executable named file in the
// directories named by the PATH environment variable.
// The result will be an absolute path
//
// Ensure file contents have specific preamble (#!/bin/sh)
func LookPathPreamble(file string, preamble string) (string, error) {
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
		err = findExecutable(path, preamble)
		if err == nil {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}
