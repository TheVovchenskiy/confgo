package confgo

import "errors"

var (
	ErrSourceIsNil                     = errors.New("source is nil")
	ErrFormatterIsNil                  = errors.New("formatter is nil")
	ErrConstructorIsNil                = errors.New("constructor is nil")
	ErrValidatorIsNil                  = errors.New("validator is nil")
	ErrConstructorMustBePointer        = errors.New("constructor must be a pointer to a struct")
	ErrConstructorMustReturnZeroStruct = errors.New("constructor must return zero (empty) struct")
	ErrNoLoadersDefined                = errors.New("no loaders defined")
)
