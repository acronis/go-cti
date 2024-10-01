package storage

type DownloadFn func(cacheDir string) (string, error)

type Origin interface {
	Validate(Origin) error
}

type Storage interface {
	Origin() Origin
	Discover(string, string) (DownloadFn, Origin, error)
}
