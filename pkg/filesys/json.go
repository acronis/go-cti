package filesys

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadJSON(fName string, v interface{}) error {
	f, err := os.Open(fName)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("open file for read: %w", err)
		}
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func WriteJSON(fName string, v interface{}) error {
	f, err := os.OpenFile(fName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("open file for write %s: %w", fName, err)
		}
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}
