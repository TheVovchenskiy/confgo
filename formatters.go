package confgo

import (
	"bytes"
	"encoding/json"

	"github.com/caarlos0/env/v11"
)

const pairLen = 2

var _ Formatter = (*EnvFormatter)(nil)

// EnvFormatter is a formatter that parses environment variable-style key-value pairs
// and converts them into structured data. It supports the standard format of KEY=VALUE
// pairs, one per line, and handles parsing of such data into Go structs via the env package.
type EnvFormatter struct{}

func NewEnvFormatter() *EnvFormatter {
	return &EnvFormatter{}
}

func (ef *EnvFormatter) parseRawIntoMap(raw []byte) map[string]string {
	res := make(map[string]string)
	lines := bytes.Split(raw, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		pair := bytes.SplitN(line, []byte("="), pairLen)
		if len(pair) != pairLen {
			// This can only happen if the string does not have an equal sign. In this case,
			// the slice length will be equal to 1. The case where the slice length is 2 is impossible.
			continue
		}
		res[string(pair[0])] = string(pair[1])
	}
	return res
}

func (ef *EnvFormatter) Unmarshal(data []byte, v any) error {
	// At some point we may want to make our own implementation of env parser
	// in order to reduce dependencies count
	return env.ParseWithOptions(v, env.Options{
		Environment: ef.parseRawIntoMap(data),
		// Prefix: // Do we need to support this?
	})
}

var _ Formatter = (*JSONFormatter)(nil)

// JSONFormatter is a simple json formatter used to parse raw json data via the standard json package.
type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (jf *JSONFormatter) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
