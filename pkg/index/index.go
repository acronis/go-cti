package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/acronis/go-cti/pkg/filesys"
)

type IndexData struct {
	Type                 string      `json:"type"`
	AppCode              string      `json:"app_code"`
	Apis                 []string    `json:"apis,omitempty"`
	Entities             []string    `json:"entities,omitempty"`
	Assets               []string    `json:"assets,omitempty"`
	Dictionaries         []string    `json:"dictionaries,omitempty"`
	Depends              []string    `json:"depends,omitempty"`
	Examples             []string    `json:"examples,omitempty"`
	AdditionalProperties interface{} `json:"additional_properties,omitempty"`
}

type Index struct {
	Path string
	Data IndexData
}

func OpenIndexFile(path string) (Index, error) {
	file, err := os.Open(path)
	if err != nil {
		return Index{}, err
	}
	defer file.Close()

	idx, err := DecodeIndexFile(file)
	if err != nil {
		return Index{}, err
	}

	return Index{
		Path: filepath.Dir(path),
		Data: idx,
	}, nil
}

func DecodeIndexBytes(data []byte) (IndexData, error) {
	var idx IndexData
	err := json.Unmarshal(data, &idx)
	return idx, err
}

func DecodeIndexFile(file *os.File) (IndexData, error) {
	var idx IndexData
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&idx)
	if err != nil {
		return IndexData{}, fmt.Errorf("error decoding index file: %w", err)
	}

	return idx, nil
}

func (idx *Index) Save() error {
	data, err := json.MarshalIndent(idx.Data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(idx.Path, "index.json"), data, 0755)
}

func (idx *Index) GetEntities() ([]Entity, error) {
	var entities []Entity
	for _, entity := range idx.Data.Entities {
		name := filesys.GetBaseName(entity)
		entities = append(entities, Entity{
			Name: name,
			Path: entity,
		})
	}

	return entities, nil
}

func (idx *Index) GetAssets() (*[]Asset, error) {
	var assets []Asset
	for _, asset := range idx.Data.Assets {
		content, err := os.ReadFile(path.Join(idx.Path, asset))
		if err != nil {
			return nil, err
		}
		assets = append(assets, Asset{
			Name: asset,
			// TODO: Storing entire assets in memory is a REALLY BAD IDEA
			Value: content,
		})
	}

	return &assets, nil
}

func (idx *Index) GetDictionaries() (Dictionaries, error) {
	dictionaries := Dictionaries{
		Dictionaries: make(map[LangCode]Entry),
	}

	for _, dict := range idx.Data.Dictionaries {
		file, err := os.Open(path.Join(idx.Path, dict))
		if err != nil {
			return Dictionaries{}, err
		}
		defer file.Close()

		entry, err := ValidateDictionary(file)
		if err != nil {
			return Dictionaries{}, err
		}
		lang := filesys.GetBaseName(file.Name())
		dictionaries.Dictionaries[LangCode(lang)] = entry
	}

	return dictionaries, nil
}

func ValidateDictionary(file *os.File) (Entry, error) {
	var config Entry
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error decoding dictionary file: %w", err)
	}

	return config, nil
}
