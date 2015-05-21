package jsonv

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
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

/*
Holds the name and parser/validator for a single JSON object property

Note: Whether or not the value any non-slice, non-ptr field is required
*/
type ObjectProp struct {
	schema   SchemaType
	def      reflect.Value
	f        field
	required bool
}

func Prop(n string, s SchemaType) ObjectProp {
	return ObjectProp{
		schema:   s,
		f:        field{nameBytes: []byte(n)},
		required: true,
	}
}

func PropWithDefault(n string, s SchemaType, d interface{}) ObjectProp {
	return ObjectProp{
		schema:   s,
		def:      reflect.ValueOf(d),
		f:        field{nameBytes: []byte(n)},
		required: true,
	}
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

			if prop.def.IsValid() {
				// fix prop.def want leaf value, not ptr
				for prop.def.Kind() == reflect.Ptr {
					if prop.def.IsNil() {
						return fmt.Errorf(ERROR_NIL_DEFAULT, prop.f.name)
					}
				}

				// make sure default type is the same as the field type
				dtyp := prop.def.Type()
				if f.typ != dtyp {
					return fmt.Errorf(ERROR_WRONG_TYPE_DEFAULT, dtyp, f.typ)
				}
			}

			// determine if it's a required field (field.typ) is always the
			// concrete type
			ft := t.FieldByIndex(f.index)
			prop.required = ft.Type.Kind() != reflect.Ptr
			if ps, ok := prop.schema.(PreparedSchemaType); ok {
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
			missingFields = append(missingFields, pr.f.name)
		}
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("No field for props: %v on struct %v", missingFields, t)
	}

	return nil
}

func (p *ObjectParser) getProp(name []byte) (int, *ObjectProp) {
	// get the property
	var prop *ObjectProp
	var propi int
	for i := range p.props {
		pr := &p.props[i]

		if bytes.Equal(pr.f.nameBytes, name) {
			prop = pr
			propi = i
			break
		}
		if prop == nil && pr.f.equalFold(pr.f.nameBytes, name) {
			prop = pr
			propi = i
		}
	}

	return propi, prop
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
		return NewParseError("Expected '{' not " + tok.String())
	}

	// we'll accumulate validation errors into this
	var errs ValidationError
	// we'll track found properties into this
	gotProps := make([]bool, len(p.props))

	for {
		var key []byte

		// read the key, or '}'
		if tok, keyb, err := s.ReadToken(); tok == tokenError {
			return err
		} else if tok == tokenObjectEnd {
			break
		} else if tok != tokenString {
			return NewParseError("Expected object property name or '}' not " + tok.String())
		} else {
			key = keyb
		}

		// read the ':'
		if tok, _, err = s.ReadToken(); tok == tokenError {
			return err
		} else if tok != tokenPropSep {
			return NewParseError("Expected ':' not " + tok.String())
		}

		// get the appropriate prop
		propIndex, prop := p.getProp(key[1 : len(key)-1])
		if prop == nil {
			if err := s.SkipValue(); err != nil {
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

			// parse the value
			propPath := fmt.Sprintf("%s%s", path, key[1:len(key)-1])
			if err := prop.schema.Parse(propPath, s, propval.Addr().Interface()); err != nil {
				if verr, ok := err.(ValidationError); ok {
					errs = errs.AddMany(verr)
				} else {
					return err
				}
			}

			// we got it!!
			gotProps[propIndex] = true
		}

		// we want a , or a }
		if tok, _, err := s.ReadToken(); tok == tokenError {
			return err
		} else if tok == tokenObjectEnd {
			break
		} else if tok == tokenItemSep {
			// Note this a trailing ',' before the '}'
			continue
		} else {
			return NewParseError("Expected ',' or '}' not " + tok.String())
		}
	}

	// check we got all the required fields
	for i, prop := range p.props {
		if gotProps[i] {
			continue
		}

		// does it have a default??
		if prop.def.IsValid() {
			// get a value referencing the firld
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

			// now set it
			propval.Set(prop.def)
		} else if prop.required {
			errs = errs.Add(path+p.props[i].f.name, ERROR_PROP_REQUIRED)
		}
	}

	if len(errs) > 0 {
		return errs
	} else {
		return nil
	}
}

/*
Parses a JSON value into an array whos values are a single type.
*/
type SliceParser struct {
	elemType reflect.Type
	schema   SchemaType
	vs       []SliceValidator
}

func Slice(s SchemaType, vs ...SliceValidator) *SliceParser {
	return &SliceParser{schema: s, vs: vs}
}

func (p *SliceParser) Prepare(t reflect.Type) error {
	// make sure it's a struct
	if t.Kind() != reflect.Slice {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, t)
	}

	p.elemType = t.Elem()

	// prepare our sub-type if we need to
	if ps, ok := p.schema.(PreparedSchemaType); ok {
		return ps.Prepare(p.elemType)
	}

	return nil
}

func (p *SliceParser) Parse(path string, s *Scanner, v interface{}) error {
	// check we have a ptr to a struct
	ptrVal := reflect.ValueOf(v)
	ptrType := ptrVal.Type()
	if ptrType.Kind() != reflect.Ptr || ptrVal.IsNil() {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, ptrVal.Type())
	}
	val := ptrVal.Elem()
	valType := val.Type()
	if valType.Kind() != reflect.Slice {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, ptrVal.Type())
	}

	// read the '['
	tok, _, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenArrayBegin {
		return NewParseError("Expected '[' not " + tok.String())
	}

	finished := false

	// see if we have at least 1 value
	if tok, err := s.PeekToken(); err != nil {
		return err
	} else if tok == tokenArrayEnd {
		// actually consume it
		s.ReadToken()
		finished = true
	}

	// this is where we'll store all the validation errors
	var errs ValidationError

	// now read val then ','|']'
	i := 0
	for !finished {
		// next up must be a value
		// Grow the slice if necessary
		if i >= val.Cap() {
			newcap := val.Cap() + val.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(val.Type(), val.Len(), newcap)
			reflect.Copy(newv, val)
			val.Set(newv)
		}
		if i >= val.Len() {
			val.SetLen(i + 1)
		}

		// read in the value
		itemPtr := val.Index(i).Addr().Interface()
		itemPath := fmt.Sprintf("%s%d/", path, i)
		if err := p.schema.Parse(itemPath, s, itemPtr); err != nil {
			if verr, ok := err.(ValidationError); ok {
				errs = errs.AddMany(verr)
			} else {
				return err
			}
		}

		i++

		// we want either a ',' or a ']'
		if tok, _, err := s.ReadToken(); tok == tokenError {
			return err
		} else if tok == tokenArrayEnd {
			finished = true
		} else if tok == tokenItemSep {
			continue
		} else {
			return NewParseError("Expected ',' or '[' not " + tok.String())
		}
	}

	// validate the contents
	for _, v := range p.vs {
		if err := v.ValidateSlice(val); err != nil {
			errs = errs.Add(path, err.Error())
		}
	}
	if len(errs) > 0 {
		return errs
	} else {
		return nil
	}
}

/*
A simple parser that accepts only a JSON string value and stores the result in
a *string or string field on a struct.

The value will be parsed (i.e. escaped chars and unicode chars parsed). Invalid
unicode code points will be replaced with unicode.ReplacementChar.
*/
type StringParser struct {
	vs []StringValidator
}

func String(vs ...StringValidator) *StringParser {
	return &StringParser{vs}
}

func (p *StringParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.String {
		return fmt.Errorf("Want string not %v", t)
	}

	return nil
}

func (p *StringParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if ss, ok := v.(*string); !ok {
		return fmt.Errorf(ERROR_BAD_STRING_DEST, reflect.TypeOf(v), path)
	} else {
		// now check for validation errors
		var errs ValidationError

		s, ok := Unquote(buf)
		if !ok {
			return errs.Add(path, "Invalid string")
		}

		*ss = s

		// validate the contents
		for _, v := range p.vs {
			if err := v.ValidateString(*ss); err != nil {
				errs = errs.Add(path, err.Error())
			}
		}

		if len(errs) > 0 {
			return errs
		} else {
			return nil
		}
	}
}

/*
Parses true/false JSON values into a *bool/*string or bool/string struct field.

For strings, the literal text "false"/"true", without quotes, is assigned to the
string.
*/
type BooleanParser struct {
}

func Boolean() *BooleanParser {
	return &BooleanParser{}
}

func (p *BooleanParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Bool && t.Kind() != reflect.String {
		return fmt.Errorf("Want bool not %v", t)
	}

	return nil
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
Parses strings into byte slices, i.e []byte, This still decodes the string value
but avoids making additional copies of the data when a byte slice is required
(e.g. decoding/decryption/etc).
*/
type ByteSliceParser struct {
}

func Bytes() *ByteSliceParser {
	return &ByteSliceParser{}
}

func (p *ByteSliceParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("Want []byte not %v", t)
	}

	return nil
}

func (p *ByteSliceParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if bdest, ok := v.(*[]byte); !ok {
		return fmt.Errorf(ERROR_BAD_BYTE_DEST, reflect.TypeOf(v), path)
	} else {
		buff, ok := UnquoteBytes(buf)
		if !ok {
			var errs ValidationError
			return errs.Add(path, "Invalid string")
		}

		*bdest = buff
	}

	return nil
}

/*
Parses strings into byte slices, i.e []byte. This does not decode the string
value, so escape sequences (e.g. "\n", "\u0020") will be left as-is.

This is useful if the value is only ever meant to be non-escaped chars, e.g. a
base64 encoded string.
*/
type RawByteSliceParser struct {
}

func RawBytes() *RawByteSliceParser {
	return &RawByteSliceParser{}
}

func (p *RawByteSliceParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("Want []byte not %v", t)
	}

	return nil
}

func (p *RawByteSliceParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if bdest, ok := v.(*[]byte); !ok {
		return fmt.Errorf(ERROR_BAD_BYTE_DEST, reflect.TypeOf(v), path)
	} else {
		// scanner owns buf, so we need to make a copy
		*bdest = make([]byte, len(buf)-2)
		copy(*bdest, buf[1:len(buf)-1])
	}

	return nil
}

/*
Parses any whole-integer JSON number value and stores it in any Go integer
primitive type, e.g. int8, int16, uint8, etc.
*/
type IntegerParser struct {
	vs []IntegerValidator
}

func Integer(vs ...IntegerValidator) *IntegerParser {
	return &IntegerParser{vs}
}

func (p *IntegerParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Int64 {
		return fmt.Errorf("Want int64 not %v", t)
	}

	return nil
}

func (p *IntegerParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenNumber {
		return NewParseError(fmt.Sprintf(ERROR_INVALID_INT, string(buf)))
	}

	var errs ValidationError

	tv, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		errs = errs.Add(path, err.Error())
		return errs
	}

	// check the value
	for _, v := range p.vs {
		if err := v.ValidateInteger(tv); err != nil {
			errs = errs.Add(path, err.Error())
		}
	}
	if len(errs) > 0 {
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
