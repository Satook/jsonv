package jsonschema

import (
	"fmt"
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

/*
A simple mapping of a JSON object to a Golang Struct. This is quite strict
*/
type ObjectParser struct {
	props []ObjectProp
}

func Object(props ...ObjectProp) *ObjectParser {
	return &ObjectParser{props}
}

func (p *ObjectParser) Parse(path string, s *Scanner, v interface{}) error {
	// want a tokenObjectBegin
	// want string
	// want a tokenObjectBegin

	return nil
}

/*
A simple parser that accepts only a JSON string value and stores the result in
a *string.
*/
type StringParser struct {
	vs []StringValidator
}

func String(vs ...StringValidator) *StringParser {
	return &StringParser{vs}
}

func (p StringParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return fmt.Errorf(ERROR_BAD_STRING_DEST, reflect.TypeOf(v), path)
	case *string:
		*t = string(buf)
	}

	return nil
}

type BooleanParser struct {
}

func Boolean() *BooleanParser {
	return &BooleanParser{}
}

func (p *BooleanParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	// wasn't the correct type
	if tok == tokenError {
		return err
	} else if tok != tokenTrue && tok != tokenFalse {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_BOOL, string(buf)))
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return fmt.Errorf(ERROR_BAD_BOOL_DEST, reflect.TypeOf(v), path)
	case *string:
		*t = string(buf)
	case *bool:
		*t = buf[0] == 't'
	}

	return nil
}

/*
Accepts any whole-integer JSON number value and stores it in any Go integer
primative type, e.g. int8, int16, uint8, etc.
*/
type IntegerParser struct {
	vs []IntegerValidator
}

func Integer(vs ...IntegerValidator) *IntegerParser {
	return &IntegerParser{vs}
}

func (p *IntegerParser) Parse(path string, s *Scanner, v interface{}) error {
	tv, err := s.ReadInteger()
	if err != nil {
		return err
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
	case *int8:
		*t = int8(tv)
	case *uint8:
		*t = uint8(tv)
	}

	return nil
}
