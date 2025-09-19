package confgo

import (
	"os"
	"strings"
	"time"
)

func stringsToBytes(s []string) []byte {
	return []byte(strings.Join(s, "\n"))
}

var _ Source = (*EnvSource)(nil)

// EnvSource is a configuration source that reads environment variables.
type EnvSource struct{}

func NewEnvSource() *EnvSource {
	return &EnvSource{}
}

func (es *EnvSource) Read() ([]byte, error) {
	return stringsToBytes(os.Environ()), nil
}

var _ Source = (*FileSource)(nil)
var _ ModTimer = (*FileSource)(nil)

// FileSource is a configuration source that reads from a file.
type FileSource struct {
	path string
}

func NewFileSource(path string) *FileSource {
	return &FileSource{path: path}
}

func (fs *FileSource) Read() ([]byte, error) {
	return os.ReadFile(fs.path)
}

func (fs *FileSource) ModTime() (time.Time, error) {
	info, err := os.Stat(fs.path)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
}
