package ctipackage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func getIndexFilePath() string {
	wd := os.Getenv("TestDir")
	path := filepath.Join(wd, ".platform", "index.json")
	return path
}

func TestConfig_IsValid_OK(t *testing.T) {
	// Create a temporary file for testing
	tempFilePath, _ := getTempIndexJson()
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	_, err = DecodeIndex(file)
	require.NoError(t, err)
}

func TestConfig_IsValid_Err(t *testing.T) {
	// Create a temporary file for testing
	tempFilePath, _ := getTempInvalidIndexJson()
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	_, err = DecodeIndex(file)
	require.Errorf(t, err, "error decoding index file: json: cannot unmarshal object into Go struct field Config.entities of type []string")
}

func TestOpenIndexFile_OK(t *testing.T) {
	tempFilePath, _ := getTempIndexJson()
	defer os.Remove(tempFilePath)

	config, err := ReadIndexFile("json")
	require.NoError(t, err)
	require.NotNil(t, config)
}

func TestOpenIndexFile_err(t *testing.T) {
	tempFilePath, _ := getTmpJsonFile("nonIndex.json")
	defer os.Remove(tempFilePath)

	_, err := ReadIndexFile(tempFilePath)
	require.Errorf(t, err, "error decoding index file: json: cannot unmarshal string into Go value of type index.PackageIndex")
}

func TestOpenIndexFile_err2(t *testing.T) {
	nonExistenceFile := "nonExistenceFile.json"
	_, err := ReadIndexFile(nonExistenceFile)
	require.Errorf(t, err, "error decoding index file: json: cannot unmarshal string into Go value of type index.PackageIndex")
}

func TestValidateIndex_Err(t *testing.T) {
	tempFilePath, _ := getTmpJsonFile("nonIndex.json")
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	_, err = DecodeIndex(file)
	require.Errorf(t, err, "error decoding index file: json: cannot unmarshal string into Go value of type index.PackageIndex")
}

func getTmpJsonFile(fileName string) (string, []byte) {
	// Create a temporary test file
	fileContent := []byte(`"assets": [
		"tmp/tmpFile1.json",
		"tmp/tmpFile1.json"
	  ]`)

	_ = createTestFile(fileName, fileContent)

	return fileName, fileContent
}

func TestValidateIndex_OK(t *testing.T) {
	tempFilePath, _ := getTempIndexJson()
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	config, err := DecodeIndex(file)
	require.NoError(t, err)
	require.NotNil(t, config)
}

func getTempIndexJson() (string, []byte) {
	// Create a temporary test file
	fileName := "index.json"
	fileContent := []byte(`{
	  "entities": [
		"entities/backup.raml",
		"entities/basic_alerts.raml",
		"entities/basic_alerts_objects.raml",
		"entities/email_security_1.raml",
		"entities/email_security_2.raml",
		"entities/security.raml"
	  ]
	}`)

	_ = createTestFile(fileName, fileContent)

	return fileName, fileContent
}

func getTempInvalidIndexJson() (string, []byte) {
	// Create a temporary test file
	fileName := "index.json"
	fileContent := []byte(`{
	  "entities": {
		"test": "test"
	  }
	}`)

	_ = createTestFile(fileName, fileContent)

	return fileName, fileContent
}

func createTestFile(filePath string, content []byte) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(string(content))
	return err
}

func TestGetEntities(t *testing.T) {
	// TODO rework
	path := getIndexFilePath()
	idx, _ := ReadIndexFile(path)
	entities, err := idx.GetEntities()
	require.NoError(t, err)

	require.NotEmpty(t, entities)

	jsonOutput, err := json.Marshal(entities)
	require.NoError(t, err)
	require.NotEmpty(t, jsonOutput)
}

func TestGetAssets(t *testing.T) {
	// TODO rework
	path := getIndexFilePath()
	idx, _ := ReadIndexFile(path)
	assets := idx.GetAssets()

	require.Empty(t, assets)
}

func TestGetDictionaries(t *testing.T) {
	// TODO rework
	pkg := New(getIndexFilePath())
	require.NoError(t, pkg.Read())

	dictionaries, err := pkg.GetDictionaries()
	require.NoError(t, err)
	require.NotEmpty(t, dictionaries)

	jsonOutput, err := json.Marshal(dictionaries)
	require.NoError(t, err)
	require.NotEmpty(t, jsonOutput)
}

func getTmpDictionaryFile(fileName string) (string, []byte) {
	// Create a temporary test file
	fileContent := []byte(`{ "name": "test", "description": "test desc" }`)

	_ = createTestFile(fileName, fileContent)

	return fileName, fileContent
}

func TestValidateDictionary_OK(t *testing.T) {
	tempFilePath, _ := getTmpDictionaryFile("en.json")
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	config, err := validateDictionary(file)
	require.NoError(t, err)
	require.NotNil(t, config)
}

func TestValidateDictionary_Err(t *testing.T) {
	tempFilePath, _ := getTmpJsonFile("any.json")
	defer os.Remove(tempFilePath)

	file, err := os.Open(tempFilePath)
	require.NoError(t, err)
	defer file.Close()

	_, err = validateDictionary(file)
	require.Errorf(t, err, "error decoding dictionary file: json: cannot unmarshal string into Go value of type models.Entry")
}
