package jsonv

import (
	"fmt"
	"io"
	"reflect"
)

/*
These are returned for parsing errors that don't render the input un-parsable.

E.g. 1: The failure of a validation requirement means that parsing can continue
whereas a malformed number, e.g. "e22", would require parsing to stop.

E.g. 2: If the root cannot be parsed, the path will be "/" and the Error string
a, hopefully, useful message for the client.
*/
type InvalidData struct {
	Path  string
	Error string
}

type ValidationError []InvalidData

func (v ValidationError) Error() string {
	// some handy way to write it out
	return fmt.Sprint([]InvalidData(v))
}

func (v ValidationError) Len() int {
	return len([]InvalidData(v))
}

func (v ValidationError) Add(path, message string) ValidationError {
	if len(v)+1 > cap(v) {
		newCap := cap(v) + cap(v)/2
		if newCap < 4 {
			newCap = 4
		}
		newv := make([]InvalidData, len(v), newCap)
		copy(newv, v)
		v = newv
	}
	// capacity is there, so just resize
	v = v[:len(v)+1]
	v[len(v)-1] = InvalidData{path, message}

	return v
}

func (v ValidationError) AddMany(o ValidationError) ValidationError {
	off := len(v)
	if off+len(o) > cap(v) {
		newCap := cap(v) + len(o) + cap(v)/2
		if newCap < 4 {
			newCap = 4
		}
		newv := make([]InvalidData, off, newCap)
		copy(newv, v)
		v = newv
	}
	v = v[:off+len(o)]
	copy(v[off:], o)
	return v
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
	if p, err := ParserError(t, s); err != nil {
		panic(err)
	} else {
		return p
	}
}

/*
Same as Parser, but returns an error instead of panicing
*/
func ParserError(t interface{}, s SchemaType) (*ValidatingParser, error) {
	targetType := reflect.Indirect(reflect.ValueOf(t)).Type()
	if ps, ok := s.(PreparedSchemaType); ok {
		if err := ps.Prepare(targetType); err != nil {
			return nil, err
		}
	}
	return &ValidatingParser{targetType, s}, nil
}

/*
Parses, and validates b into the v.

Will panic if b is not a pointer to the same type as was used to construct this
parser.
*/
func (p *ValidatingParser) Parse(r io.Reader, v interface{}) error {
	// check the type is correct
	// we must get a Ptr to same type as was given on creation
	tPtr := reflect.TypeOf(v)
	if tPtr.Kind() != reflect.Ptr || tPtr.Elem() != p.targetType {
		panic(fmt.Errorf("Expected Ptr to \"%v\", got \"%v\"", p.targetType, tPtr))
	}

	s := NewScanner(r)

	// the base pather
	path := func() string {
		return "/"
	}

	if err := p.schema.Parse(path, s, v); err != nil {
		if verr, ok := err.(ValidationError); ok {
			return verr
		} else if perr, ok := err.(*ParseError); ok {
			return NewSingleVErr("/", perr.Error())
		} else if err == io.EOF {
			return NewSingleVErr("/", "Unexpected end of input during parsing")
		} else {
			return err
		}
	}

	return nil
}
