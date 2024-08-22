package packager

import "fmt"

type PackageError struct {
	Code    string
	Message string
}

func New(code string, message string) *PackageError {
	return &PackageError{code, message}
}

func (p *PackageError) Error() string {
	return fmt.Sprintf("%s: %s", p.Code, p.Message)
}

var (
	WritingIndexJSONError = New(
		"writingIndexJSONError",
		"error while creating `index.json` file",
	)
	PreparingRamlEntitiesError = New(
		"preparingRamlEntitiesError",
		"error while processing raml entities to archive",
	)
	WritingRamlEntitiesError = New(
		"writingRamlEntitiesError",
		"error while archiving entities data input",
	)
	PreparingDictionaryDataError = New(
		"preparingDictionaryDataError",
		"error while preparing archiving dictionaries data",
	)
	WritingDictionaryDataError = New(
		"writingDictionaryDataError",
		"error while adding dictionaries to archive",
	)
	ArchivingError = New(
		"archivingError",
		"error during archiving",
	)
	AddingPlatformPackageToArchiveError = New(
		"addingPlatformPackageToArchiveError",
		"error while adding platform package to archive",
	)
	AddingPlatformUiPackageToArchiveError = New(
		"addingPlatformUiPackageToArchiveError",
		"error while adding platform ui package to archive",
	)
)
