package pacman

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	lockHelperEnvDir  = "GO_CTI_LOCK_HELPER_DIR"
	lockHelperEnvMode = "GO_CTI_LOCK_HELPER_MODE"
)

// TestLockHelperProcess is not a test: the tests below run the test binary
// again to get a second process.
func TestLockHelperProcess(t *testing.T) {
	dir := os.Getenv(lockHelperEnvDir)
	if dir == "" {
		t.Skip("helper process")
	}
	pm := &packageManager{PackagesDir: dir}

	switch os.Getenv(lockHelperEnvMode) {
	case "try":
		// Reports whether the lock is free, without taking it for good.
		f, err := os.OpenFile(pm.getCacheLockPath(), os.O_CREATE|os.O_RDWR, 0o600)
		require.NoError(t, err)
		locked, err := tryLockFile(f)
		require.NoError(t, err)
		if locked {
			os.Stdout.WriteString("free\n")
		} else {
			os.Stdout.WriteString("busy\n")
		}
	case "hold":
		// Takes the lock and never releases it: the parent kills the process.
		_, err := pm.lockCache()
		require.NoError(t, err)
		os.Stdout.WriteString("locked\n")
		select {}
	}
}

func startLockHelper(t *testing.T, dir, mode string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$")
	cmd.Env = append(os.Environ(), lockHelperEnvDir+"="+dir, lockHelperEnvMode+"="+mode)
	return cmd
}

func lockStateInOtherProcess(t *testing.T, dir string) string {
	t.Helper()
	out, err := startLockHelper(t, dir, "try").Output()
	require.NoError(t, err)
	return strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
}

func Test_LockCacheExcludesOtherProcess(t *testing.T) {
	dir := t.TempDir()
	pm := &packageManager{PackagesDir: dir}

	unlock, err := pm.lockCache()
	require.NoError(t, err)
	require.Equal(t, "busy", lockStateInOtherProcess(t, dir))

	unlock()
	require.Equal(t, "free", lockStateInOtherProcess(t, dir))
}

// The operating system drops the lock of a process that dies, so a crashed
// cyberapp does not block the next build on the node.
func Test_LockCacheIsReleasedWhenHolderDies(t *testing.T) {
	dir := t.TempDir()
	pm := &packageManager{PackagesDir: dir}

	holder := startLockHelper(t, dir, "hold")
	stdout, err := holder.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, holder.Start())
	t.Cleanup(func() { _ = holder.Process.Kill(); _ = holder.Wait() })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "locked", strings.TrimSpace(line))

	f, err := os.OpenFile(pm.getCacheLockPath(), os.O_RDWR, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	locked, err := tryLockFile(f)
	require.NoError(t, err)
	require.False(t, locked, "the lock must be held by the helper process")

	require.NoError(t, holder.Process.Kill())
	_ = holder.Wait()

	require.Eventually(t, func() bool {
		ok, err := tryLockFile(f)
		return err == nil && ok
	}, 5*time.Second, 20*time.Millisecond, "the lock must be free after the holder died")
}

func Test_LockCacheWaitsForHolder(t *testing.T) {
	pm := &packageManager{PackagesDir: t.TempDir()}

	unlock, err := pm.lockCache()
	require.NoError(t, err)

	acquired := make(chan struct{})
	go func() {
		release, err := pm.lockCache()
		if err == nil {
			close(acquired)
			release()
		}
	}()

	select {
	case <-acquired:
		t.Fatal("the second lock was granted while the first was held")
	case <-time.After(200 * time.Millisecond):
	}

	unlock()
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the second lock was not granted after the first was released")
	}
}
