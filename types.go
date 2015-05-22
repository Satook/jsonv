package jsonv

import (
	"reflect"
)

/*
Used by Parser for parsing and validation of JSON types.

Can return either a ValidationError or a general error if encountered
This is used to allow the parser and it's clients to differentiate between
validation errors and IO errors.

If the error is just a vaidation error, but parsing can continue, the
implementation should return a ValidationError, otherwise any other error type
will be collected up with all errors accumulated so far and parsing stopped.
*/
type SchemaType interface {
	Parse(string, *Scanner, interface{}) error
}

/*
SchemaTypes can implement this to allow

Anything that embeds other types must implement this and call it on embedded
types to allow them to initialise once the type is known. E.g. Object types
use this to pre-cache all the fields they need and call it on each of their
ObjectProp.Types
*/
type PreparedSchemaType interface {
	Prepare(reflect.Type) error
}
