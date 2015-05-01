package jsonschema

import (
	"fmt"
	"io"
	"reflect"
)

/*
These are returned for any/all parsing errors.

E.g. If the root cannot be parsed, the path will be "/" and the Error string a,
hopefully, useful message for the client.
*/
type InvalidData struct {
	Path  string
	Error string
}

type ValidationError []InvalidData

func (v ValidationError) Error() string {
	// some handy way to write it out
	return "Not yet implemented"
}

func NewVErr(es []InvalidData) ValidationError {
	return ValidationError(es)
}

func NewSingleVErr(path, msg string) ValidationError {
	return []InvalidData{{path, msg}}
}

type ValidatingParser struct {
	targetType reflect.Type
	schema     SchemaType
}

/*
Build a parser, caching relevant metadata of the target type, t.

The first parameter, t, should be an instance of the type you will use. It can
be a pointer to or direct instance of the type, e.g. both Parser(&T{}) &
Parser(T{}) should work for most types.
*/
func Parser(t interface{}, s SchemaType) *ValidatingParser {
	// TODO: Check target type is valid, can take all fields, field types and
	// validators are all line up, etc
	return &ValidatingParser{reflect.Indirect(reflect.ValueOf(t)).Type(), s}
}

/*
Parses, and validates b into the v.

Will panic if b is not a pointer to the same type as was used to construct this
parser.
*/
func (p *ValidatingParser) Parse(r io.Reader, v interface{}) []InvalidData {
	// check the type is correct
	// we must get a Ptr to same type as was given on creation
	tPtr := reflect.TypeOf(v)
	if tPtr.Kind() != reflect.Ptr || tPtr.Elem() != p.targetType {
		panic(fmt.Errorf("Expected Ptr to \"%v\", got \"%v\"", p.targetType, tPtr))
	}

	s := NewScanner(r)

	p.schema.Parse("/", s, v)

	return nil
}
