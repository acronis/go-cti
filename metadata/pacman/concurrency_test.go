package pacman

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acronis/go-cti/metadata/ctipackage"
	"github.com/acronis/go-cti/metadata/storage"
	"github.com/stretchr/testify/require"
)

// overlapStorage counts the downloads that run at the same time.
type overlapStorage struct {
	mockStorage
	hold   time.Duration
	active atomic.Int32
	peak   atomic.Int32
}

type overlapInfo struct {
	mockInfo
	owner *overlapStorage
}

func (s *overlapStorage) Origin() storage.Origin {
	return &overlapInfo{owner: s}
}

func (s *overlapStorage) Discover(name, version string) (storage.Origin, error) {
	return &overlapInfo{mockInfo: mockInfo{Name: name, Version: version}, owner: s}, nil
}

func (i *overlapInfo) Validate(o storage.Origin) error {
	oi, ok := o.(*overlapInfo)
	if !ok {
		return fmt.Errorf("origin is not an overlapInfo")
	}
	return i.mockInfo.Validate(&oi.mockInfo)
}

func (i *overlapInfo) Download(dst string) (string, error) {
	n := i.owner.active.Add(1)
	defer i.owner.active.Add(-1)
	for {
		peak := i.owner.peak.Load()
		if n <= peak || i.owner.peak.CompareAndSwap(peak, n) {
			break
		}
	}
	time.Sleep(i.owner.hold)
	return i.mockInfo.Download(dst)
}

// The property behind the lock: downloads into one cache never overlap. The
// sleep makes the overlap certain without the lock, so the test does not depend
// on scheduling luck.
func Test_DownloadsIntoOneCacheDoNotOverlap(t *testing.T) {
	const workers = 6

	st := &overlapStorage{hold: 5 * time.Millisecond}
	cacheDir := filepath.Join(t.TempDir(), "_cache")

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			pm, err := New(WithStorage(st), WithPackagesCache(cacheDir))
			if err != nil {
				errs <- err
				return
			}
			pkg, err := ctipackage.New(filepath.Join(t.TempDir(), fmt.Sprintf("local-%d", w)),
				ctipackage.WithID("xyz.mock"))
			if err != nil {
				errs <- err
				return
			}
			if err := pkg.Initialize(); err != nil {
				errs <- err
				return
			}
			errs <- pm.Add(pkg, map[string]string{"mock@b1": "1.0.0"})
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), st.peak.Load(), "downloads into one cache ran at the same time")
}

// Several package managers share one cache directory, as several cyberapp
// processes do on one CI node. Every Add downloads and replaces the same
// cache entries, so without a lock a reader or a writer meets a half-replaced
// directory.
func Test_ConcurrentAddSharesCache(t *testing.T) {
	const (
		workers = 8
		rounds  = 10
	)

	cacheDir := filepath.Join(t.TempDir(), "_cache")
	depends := map[string]string{
		"mock@b1": "1.0.0",
		"mock@b3": "3.4.5",
	}

	var wg sync.WaitGroup
	errs := make(chan error, workers*rounds)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()

			pm, err := New(WithStorage(&mockStorage{}), WithPackagesCache(cacheDir))
			if err != nil {
				errs <- fmt.Errorf("worker %d: new: %w", w, err)
				return
			}

			for r := 0; r < rounds; r++ {
				pkg, err := ctipackage.New(filepath.Join(t.TempDir(), fmt.Sprintf("local-%d-%d", w, r)),
					ctipackage.WithID("xyz.mock"))
				if err != nil {
					errs <- fmt.Errorf("worker %d round %d: new package: %w", w, r, err)
					return
				}
				if err := pkg.Initialize(); err != nil {
					errs <- fmt.Errorf("worker %d round %d: initialize: %w", w, r, err)
					return
				}
				if err := pm.Add(pkg, depends); err != nil {
					errs <- fmt.Errorf("worker %d round %d: add: %w", w, r, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}
