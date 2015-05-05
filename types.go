package jsonv

import (
	"bytes"
	"encoding/json"
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
SchemaTypes can implement this to allow

Anything that embeds other types must implement this and call it on embedded
types to allow them to initialise once the type is known. E.g. Object types
use this to pre-cache all the fields they need and call it on each of their
ObjectProp.Types
*/
type PrecacheSchemaType interface {
	Prepare(reflect.Type) error
}

/*
Holds the name and parser/validator for a single JSON object property

Note: Whether or not the value is required determined as follows:
 - If a DefaultValue
*/
type ObjectProp struct {
	Name   string
	Schema SchemaType
	f      field
}

func Prop(n string, s SchemaType) ObjectProp {
	return ObjectProp{n, s, field{nameBytes: []byte(n)}}
}

/*
A simple mapping of a JSON object to a Golang Struct.

Any field that is a Pointer is an optional JSON property. Non-pointer
fields are mandatory JSON properties. Validators are only invoked on
properties that are present.

Unexpected fields will result in a ValidationError being pushed out.

Properties are mapped to struct fields in the same way the inbuilt
json.Unmarshall, i.e. via a depth-first mapping of, potentially overriden via
tags, field names into a flat namespace on a last in wins basis. So embedded
structs can have their fields hidden by the structs they're embedded within.

Note: Only exported fields are touched.

For example, the following struct will be considered to have 1 mandatory field
"Name":

	type Person struct {
		Name string
	}

As would this:

	type Person struct {
		Fullname string `json:"Name"`
	}

In this example, only Person.OuterName would be touched by the parser:

	type InnerPerson struct {
		InnerName string `json:"Name"`
	}

	type Person struct {
		InnerPerson
		OuterName string `json:"Name"`
	}

And if the OuterName field were just Name with no tag, it would still be used
over InnerPerson.InnerName, even though the latter is tagged. Tagged fields take
precedence only when competing at the same depth. So in this example the
Person.OtherName field would be used by the parser:

	type Person struct {
		OtherName string `json:"Name"`
		Name string
	}

*/
type ObjectParser struct {
	props []ObjectProp
}

func Object(props ...ObjectProp) *ObjectParser {
	// fill in the
	return &ObjectParser{props}
}

/*
We cache all the field lookup info here.
*/
func (p *ObjectParser) Prepare(t reflect.Type) error {
	// make sure it's a struct
	if t.Kind() != reflect.Struct {
		return fmt.Errorf(ERROR_BAD_OBJ_DEST, t)
	}

	// fill in the field for each prop
	fields := typeFields(t)
	for i := range fields {
		f := &fields[i]
		var prop *ObjectProp

		// find the prop for this field
		for j := range p.props {
			pr := &p.props[j]

			if bytes.Equal(f.nameBytes, pr.f.nameBytes) {
				prop = pr
				break
			}
			if prop == nil && f.equalFold(f.nameBytes, pr.f.nameBytes) {
				prop = pr
			}
		}

		// save info and Prepare the Schema if needed
		if prop != nil {
			prop.f = *f
			if ps, ok := prop.Schema.(PrecacheSchemaType); ok {
				if err := ps.Prepare(f.typ); err != nil {
					return err
				}
			}
		}
	}

	// check we found a field for each prop
	missingFields := make([]string, 0, 32)
	for i := range p.props {
		pr := &p.props[i]
		if pr.f.index == nil {
			missingFields = append(missingFields, pr.Name)
		}
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("No field for props: %v on struct %v", missingFields, t)
	}

	return nil
}

func (p *ObjectParser) getProp(name []byte) *ObjectProp {
	// get the property
	var prop *ObjectProp
	for i := range p.props {
		pr := &p.props[i]

		if bytes.Equal(pr.f.nameBytes, name) {
			prop = pr
			break
		}
		if prop == nil && pr.f.equalFold(pr.f.nameBytes, name) {
			prop = pr
		}
	}

	return prop
}

/*
Won't allocate the struct, but will allocate fields if needed.
*/
func (p *ObjectParser) Parse(path string, s *Scanner, v interface{}) error {
	// check we have a ptr to a struct
	ptrVal := reflect.ValueOf(v)
	ptrType := ptrVal.Type()
	if ptrType.Kind() != reflect.Ptr || ptrVal.IsNil() {
		return fmt.Errorf(ERROR_BAD_OBJ_DEST, ptrVal.Type())
	}
	val := ptrVal.Elem()
	valType := val.Type()
	if valType.Kind() != reflect.Struct {
		return fmt.Errorf(ERROR_BAD_OBJ_DEST, ptrVal.Type())
	}

	// read the '{'
	tok, _, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenObjectBegin {
		return NewParseError("'{' expected, not " + tok.String())
	}

	// read the first key, or '}'
	tok, key, err := s.ReadToken()
	for tok != tokenObjectEnd {
		if tok == tokenError {
			return err
		} else if tok != tokenString {
			return NewParseError("Object property name or '}' expected, not " + tok.String())
		}

		// read the ':'
		tok, _, err = s.ReadToken()
		if tok == tokenError {
			return err
		} else if tok != tokenPropSep {
			return NewParseError("':' expected, not " + tok.String())
		}

		// get the appropriate prop
		prop := p.getProp(key[1 : len(key)-1])
		if prop == nil {
			// skip the value
			_, _, err := s.ReadToken() // TODO: this handles simple values only for now (no arrays or objects)
			if err != nil {
				return err
			}
		} else {
			// walk to the actual value and allocate if needed
			propval := val
			for _, i := range prop.f.index {
				propval = propval.Field(i)
				if propval.Kind() == reflect.Ptr {
					if propval.IsNil() {
						propval.Set(reflect.New(propval.Type().Elem()))
					}
					propval = propval.Elem()
				}
			}
			if err := prop.Schema.Parse("/", s, propval.Addr().Interface()); err != nil {
				return err
			}
		}

		// we want a , or a }
		tok, _, err = s.ReadToken()
		if tok == tokenError {
			return err
		} else if tok == tokenObjectEnd {
			break
		} else if tok == tokenItemSep {
			// Note this + the loop conditional allows a trailing ',' before the '}'
			tok, key, err = s.ReadToken()
		} else {
			return NewParseError("Object , or } expected, not " + tok.String())
		}
	}

	// TODO: check for mandatory fields

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
		// TODO: Actually parse the thing, this is a fairly sub-optimal method
		json.Unmarshal(buf, t)
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
