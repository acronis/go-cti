package pacman

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

func (pm *packageManager) getCacheLockPath() string {
	return filepath.Join(pm.PackagesDir, ".cache", "lock")
}

// lockCache takes the exclusive lock of the packages cache and returns the
// function that releases it. Processes that share one cache directory download
// into it and copy out of it. Both replace and read the same directories, so
// they must not overlap. The operating system drops the lock when the process
// ends, so a crashed process does not block the others.
func (pm *packageManager) lockCache() (func(), error) {
	lockPath := pm.getCacheLockPath()
	if err := os.MkdirAll(filepath.Dir(lockPath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", lockPath, err)
	}
	if !locked {
		slog.Info("Waiting for the packages cache lock", slog.String("path", lockPath))
		if err := fl.Lock(); err != nil {
			return nil, fmt.Errorf("lock %s: %w", lockPath, err)
		}
	}

	return func() {
		if err := fl.Unlock(); err != nil {
			slog.Error("Failed to unlock the packages cache", slog.String("path", lockPath), slog.String("error", err.Error()))
		}
	}, nil
}
