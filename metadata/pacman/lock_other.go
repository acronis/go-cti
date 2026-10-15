//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly || windows)

package pacman

import "os"

// No file locking on this platform: processes that share a cache directory
// are not serialised.

func tryLockFile(*os.File) (bool, error) { return true, nil }

func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) error { return nil }
