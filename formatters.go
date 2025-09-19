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

// JSONFormatterOption option that configures json decoder.
type JSONFormatterOption func(jf *JSONFormatter)

// DisallowUnknownFields causes the json.Decoder to return an error when the
// destination is a struct and the input contains object keys which do not match
// any non-ignored, exported fields in the destination.
func DisallowUnknownFields(jf *JSONFormatter) {
	jf.decoderTweaks = append(jf.decoderTweaks, func(decoder *json.Decoder) { decoder.DisallowUnknownFields() })
}

// UseNumber causes the json.Decoder to unmarshal a number into an interface
// value as a json.Number instead of as a float64.
func UseNumber(jf *JSONFormatter) {
	jf.decoderTweaks = append(jf.decoderTweaks, func(decoder *json.Decoder) { decoder.UseNumber() })
}

var _ Formatter = (*JSONFormatter)(nil)

// JSONFormatter is a simple json formatter used to parse raw json data via the standard json package.
type JSONFormatter struct {
	decoderTweaks []func(*json.Decoder)
}

func NewJSONFormatter(opts ...JSONFormatterOption) *JSONFormatter {
	jsonF := &JSONFormatter{}
	for _, opt := range opts {
		if opt != nil {
			opt(jsonF)
		}
	}
	return jsonF
}

func (jf *JSONFormatter) Unmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	for _, tweak := range jf.decoderTweaks {
		tweak(dec)
	}
	return dec.Decode(v)
}
