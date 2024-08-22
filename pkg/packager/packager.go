package packager

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type RamlPackager interface {
	Archive() (*bytes.Buffer, error)
}

type packagerImpl struct {
	entities           []byte
	dictionaries       []byte
	assets             []os.File
	sourcePlatformPath string
	additionalInfo     []byte
}

// Archive organize platform and entities folders and write entity buffer.
func (p packagerImpl) Archive() (*bytes.Buffer, error) {
	// Create a new buffer to store the zip file
	buffer := new(bytes.Buffer)

	// Create a new zip writer
	zipWriter := zip.NewWriter(buffer)

	// Add platform package to archive
	platformPackagePath := filepath.Join(p.sourcePlatformPath, "platform")
	err := addPlatformPackageToArchive(zipWriter, platformPackagePath, PlatformPackageFolder)
	if err != nil {
		return nil, AddingPlatformPackageToArchiveError
	}

	// Add platform ui package to archive
	platformUiPackagePath := filepath.Join(p.sourcePlatformPath, "ui")
	err = addPlatformPackageToArchive(zipWriter, platformUiPackagePath, PlatformUiPackageFolder)
	if err != nil {
		return nil, AddingPlatformUiPackageToArchiveError
	}

	// Add hidden platform package to ui folder as dependencies
	err = addPlatformPackageToArchive(zipWriter, platformPackagePath, UIPlatformPackageFolder)
	if err != nil {
		return nil, AddingPlatformPackageToArchiveError
	}

	// Create index.json
	var indexJSON = IndexJSON{Type: CtiType}

	// Add entities to archive and index.json
	if p.entities != nil {
		var errE error
		indexJSON.Entities, errE = addEntitiesToArchive(zipWriter, p.entities)
		if errE != nil {
			return nil, errE
		}
	}

	// Add dictionary to archive and index.json
	if p.dictionaries != nil {
		var errD error
		indexJSON.Dictionaries, errD = addDictionariesToArchive(zipWriter, p.dictionaries)
		if errD != nil {
			return nil, errD
		}
	}

	// Add assets to archive and index.json
	if p.assets != nil {
		var errA error
		indexJSON.Assets, errA = addAssetsFileToArchive(zipWriter, p.assets)
		if errA != nil {
			return nil, errA
		}
	}

	if p.additionalInfo != nil {
		var errI error
		indexJSON.AdditionalInfo, errI = p.addAdditionalInfoToArchive(p.additionalInfo)
		if errI != nil {
			return nil, errI
		}
	}

	// Add index.json to archive
	indexJSONContent, _ := json.MarshalIndent(&indexJSON, "", "  ")
	err = addFileToArchive(zipWriter, "index.json", indexJSONContent)
	if err != nil {
		return nil, WritingIndexJSONError
	}

	// Close writer
	err = zipWriter.Close()
	if err != nil {
		return nil, ArchivingError
	}

	return buffer, nil
}

func (p packagerImpl) addAdditionalInfoToArchive(additionalInfo []byte) (*AdditionalInfo, error) {
	var appAdditionalInfo AdditionalInfo
	err := json.Unmarshal(additionalInfo, &appAdditionalInfo)
	if err != nil {
		return nil, err
	}

	return &appAdditionalInfo, nil
}

func addEntitiesToArchive(zipWriter *zip.Writer, entities []byte) ([]string, error) {
	// Create entity raml files
	// entitiesPath := "entities/"
	// Add entity files to archive
	var jsonEntities []string
	vendorEntities, err := getRamlEntities(entities)
	if err != nil {
		return nil, PreparingRamlEntitiesError
	}
	for name, ramlContent := range vendorEntities {
		// Create entity raml file name
		file := filepath.Base(name)
		dirName := filepath.Dir(name)
		var fileName string
		if dirName == "." {
			fileName = filepath.Join("entities", file)
		} else {
			fileName = filepath.Join(dirName, file)
		}

		// Writer file names to index.json entities
		jsonEntities = append(jsonEntities, fileName)
		// Add entity files to archive
		errE := addFileToArchive(zipWriter, fileName, ramlContent)
		if errE != nil {
			return nil, WritingRamlEntitiesError
		}
	}

	return jsonEntities, nil
}

// addDictionariesToArchive add dictionary files to archive.
func addDictionariesToArchive(zipWriter *zip.Writer, dictionaries []byte) ([]string, error) {
	dictionariesPath := "dictionaries/"

	// Add dictionary files to archive
	dictionaryMap, errE := getDictionaries(dictionaries)
	if errE != nil {
		return nil, PreparingDictionaryDataError
	}

	var jsonDictionaries []string
	for domain, dictionary := range dictionaryMap {
		// Create entity raml file name
		fileName := fmt.Sprintf("%s%s.json", dictionariesPath, domain)
		// Writer file names to index.json entities
		jsonDictionaries = append(jsonDictionaries, fileName)
		// Add entity files to archive
		err := addFileToArchive(zipWriter, fileName, dictionary)
		if err != nil {
			return nil, WritingDictionaryDataError
		}
	}

	return jsonDictionaries, nil
}

func addFileToArchive(zipWriter *zip.Writer, fileName string, content []byte) error {
	fileWriter, err := zipWriter.Create(fileName)
	if err != nil {
		return err
	}

	_, err = fileWriter.Write(content)
	return err
}

// addAssetsFileToArchive add assets file to archive and add list of them to index.json file.
func addAssetsFileToArchive(zipWriter *zip.Writer, assetFiles []os.File) ([]string, error) {
	var jsonAssets []string
	for i, asset := range assetFiles {
		// Get the file information
		fileInfo, err := asset.Stat()
		if err != nil {
			return nil, err
		}

		// Create a new file header
		header, errZh := zip.FileInfoHeader(fileInfo)
		if errZh != nil {
			return nil, errZh
		}

		// Create a new file in the archive
		header.Name = filepath.Join("assets/", header.Name)
		jsonAssets = append(jsonAssets, header.Name)
		archiveFile, errH := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, errH
		}

		_, err = io.Copy(archiveFile, &assetFiles[i])
		if err != nil {
			return nil, err
		}
	}

	return jsonAssets, nil
}

// getRamlEntities extract dictionaries string.
func getDictionaries(data []byte) (map[string][]byte, error) {
	var dataMap map[string][]string

	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return nil, err
	}

	// Create entities buffer mapping
	mapDictionaryByte := make(map[string][]byte)

	// Read dictionary contents
	for name, dictionary := range dataMap {
		// Create new string builder
		var dictionaryBuilder strings.Builder
		// Read dictionary contents
		for _, ramlString := range dictionary {
			// Concat dictionary content
			dictionaryBuilder.WriteString(ramlString)
		}
		// Write contents to mapped object by domain
		mapDictionaryByte[name] = []byte(dictionaryBuilder.String())
	}

	return mapDictionaryByte, nil
}

// getRamlEntities extract RAML string.
func getRamlEntities(data []byte) (map[string][]byte, error) {
	var dataMap map[string][]string

	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return nil, err
	}

	// Create entities buffer mapping
	mapEntitiesByte := make(map[string][]byte)

	// Read raml contents
	for name, ramlDoc := range dataMap {
		// Add raml header
		var ramlBuilder strings.Builder
		ramlBuilder.WriteString(RamlLibraryHeader)
		// Read raml contents
		for _, ramlString := range ramlDoc {
			// Concat raml content
			ramlBuilder.WriteString(ramlString)
		}
		// Write contents to mapped object by domain
		var fileName = name
		if filepath.Ext(fileName) == "" {
			fileName = name + RamlExtension
		}
		mapEntitiesByte[fileName] = []byte(ramlBuilder.String())
	}

	return mapEntitiesByte, nil
}

func addPlatformPackageToArchive(zipWriter *zip.Writer, sourcePath string, basePath string) error {
	err := filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && filepath.Base(filePath)[0] == '.' {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(sourcePath, filePath)
		if err != nil {
			return err
		}

		// Create a new file header based on the file path
		header, err := getZipFileHeader(info, basePath+relPath)
		if err != nil {
			return err
		}

		// Create a new file in the zip archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Reading and add file to archive
		// And exclude ramlx.js (if any)
		if info.Name() == "ramlx.js" {
			return nil
		}

		if !info.IsDir() {
			file, _ := os.Open(filePath)
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func getZipFileHeader(info os.FileInfo, headerName string) (*zip.FileHeader, error) {
	// Read file info header and set file path based on basePath
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, err
	}
	header.Name = headerName

	// If the file is a directory,
	// add a trailing slash to the file name
	if info.IsDir() {
		header.Name += "/"
	}

	return header, nil
}

func NewRamlPackager(entities []byte, dictionaries []byte, assets []os.File, sourcePlatformPath string, additionalInfo []byte) RamlPackager {
	return &packagerImpl{
		entities:           entities,
		dictionaries:       dictionaries,
		assets:             assets,
		sourcePlatformPath: sourcePlatformPath,
		additionalInfo:     additionalInfo,
	}
}
