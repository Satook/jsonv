package jsonschema

import (
	"fmt"
	"reflect"
	"strconv"
)

/*
Used by Parser for parsing and validation of JSON types.

Can return either a ValidationError or a general error if encountered
This is used to allow the parser and it's clients to differentiate between
validation errors and IO errors.
*/
type SchemaType interface {
	Parse(string, *Scanner, interface{}) error
}

/*
Holds the name and parser/validator for a single JSON object property

Note: Whether or not the value is required determined as follows:
 - If a DefaultValue
*/
type ObjectProp struct {
	Name string
	Type SchemaType
}

func Prop(n string, t SchemaType) ObjectProp {
	return ObjectProp{n, t}
}

type ObjectParser struct {
	props []ObjectProp
}

func Object(props ...ObjectProp) *ObjectParser {
	return &ObjectParser{props}
}

func (p *ObjectParser) Parse(path string, s *Scanner, v interface{}) error {
	return nil
}

func Array() {
}

func String() {
}

func Number() {
}

type BooleanParser struct {
}

func Boolean() {
}

func (p *BooleanParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	// wasn't the correct type
	if tok == tokenError {
		return err
	} else if tok == tokenParseError {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_PARSE_BOOL, err.Error()))
	} else if tok != tokenTrue && tok != tokenFalse {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_BOOL, string(buf)))
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		// TODO: Give "wrong destination type" error
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_BOOL, string(buf)))
	case *[]byte:
		*t = make([]byte, len(buf))
		copy(*t, buf)
	case *string:
		*t = string(buf)
	case *bool:
		*t = buf[0] == 't'
	}

	return nil
}

type IntegerParser struct {
	vs []IntegerValidator
}

func Integer(vs ...IntegerValidator) *IntegerParser {
	return &IntegerParser{vs}
}

func (p *IntegerParser) Parse(path string, s *Scanner, v interface{}) error {
	tv, err := s.ReadInteger()
	if err != nil {
		if v, ok := err.(*strconv.NumError); ok {
			return NewSingleVErr(path, fmt.Sprintf(ERROR_PARSE_INT, v.Error()))
		} else {
			return err
		}
	}

	// check the value
	var errs ValidationError
	for _, v := range p.vs {
		if err := v.ValidateInteger(tv); err != nil {
			if errs == nil {
				errs = make(ValidationError, 0, 5)
			}
			errs = append(errs, InvalidData{path, err.Error()})
		}
	}
	if errs != nil {
		return errs
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return fmt.Errorf(ERROR_BAD_INT_DEST, reflect.TypeOf(v))
	case *int64:
		*t = tv
	case *uint64:
		*t = uint64(tv)
	case *int:
		*t = int(tv)
	case *uint:
		*t = uint(tv)
	case *int16:
		*t = int16(tv)
	case *uint16:
		*t = uint16(tv)
	}

	return nil
}
