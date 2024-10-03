package storage

type Origin interface {
	Validate(Origin) error
	Download(string) (string, error)
}

type Storage interface {
	Origin() Origin
	Discover(string, string) (Origin, error)
}
