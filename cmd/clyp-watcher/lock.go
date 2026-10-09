package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func acquireInstanceLock() (*os.File, bool, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	lockPath := filepath.Join(runtimeDir, fmt.Sprintf("clyp-watcher-%d.lock", os.Getuid()))
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false, fmt.Errorf("open watcher lock: %w", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("lock watcher instance: %w", err)
	}
	return lock, false, nil
}
