package confgo

import "errors"

var (
	ErrConstructorIsNil                = errors.New("constructor is nil")
	ErrConstructorMustBePointer        = errors.New("constructor must be a pointer to a struct")
	ErrConstructorMustReturnZeroStruct = errors.New("constructor must return zero (empty) struct")

	ErrSourceIsNil      = errors.New("source is nil")
	ErrFormatterIsNil   = errors.New("formatter is nil")
	ErrNoLoadersDefined = errors.New("no loaders defined")
)
