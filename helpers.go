package confgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func ptr[T any](val T) *T {
	return &val
}

func writeFile(filePath, content string) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		err = file.Close()
	}()
	_, err = file.WriteString(content)
	return
}

func setupFile(filePath, content string) (func(), error) {
	err := writeFile(filePath, content)
	if err != nil {
		return nil, fmt.Errorf("error writing test config file %q: %w", filePath, err)
	}

	return func() {
		err := os.Remove(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return
			}
			panic(fmt.Sprintf("error removing test config file %q: %s", filePath, err.Error()))
		}
	}, nil
}

func setupJSONConfig(filePath string, data map[string]any) (func(), error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %w", err)
	}
	return setupFile(filePath, string(bytes))
}

func updateJSONFile(filePath string, data map[string]any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}
	return writeFile(filePath, string(bytes))
}
