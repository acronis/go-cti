package ramlx

type RamlX struct {
}

func (x RamlX) ParseIndexFile(path string) ([]byte, error) {
	return nil, nil
}

func (x RamlX) ParseEntityFile(path string) ([]byte, error) {
	return nil, nil
}

func NewRamlX() (*RamlX, error) {
	return &RamlX{}, nil
}
