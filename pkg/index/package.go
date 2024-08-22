package index

type Package struct {
	Cti          string   `json:"cti"`
	Final        bool     `json:"final"`
	Values       []string `json:"value"`
	Dictionaries []string `json:"dictionaries"`
}
