package jsonv

import (
	"bytes"
	"fmt"
	"reflect"
)

/*
Holds infomation to map a JSON object property to a struct field.

Note: Whether or not the value any non-slice, non-ptr field is required
*/
type StructPropInfo struct {
	schema   SchemaType
	def      reflect.Value
	f        field
	required bool
}

func Prop(n string, s SchemaType) StructPropInfo {
	return StructPropInfo{
		schema:   s,
		f:        field{nameBytes: []byte(n)},
		required: true,
	}
}

func PropWithDefault(n string, s SchemaType, d interface{}) StructPropInfo {
	return StructPropInfo{
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

Unexpected/unknown fields will be ignored, and their value skipped over.

Properties are mapped to struct fields in the same way the inbuilt
json.Unmarshall, i.e. via a depth-first mapping of, potentially overriden via
tags, field names into a flat namespace on a last in wins basis. So embedded
structs can have their fields hidden by the structs they're embedded within.

Note: Only exported fields are considered.

For example, the following struct will be considered to have 1 field, "Name":

	type Person struct {
		Name string
	}

As would this:

	type Person struct {
		Fullname string `json:"Name"`
	}

In this example, only Person.OuterName would be considered by the parser:

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

Note: The above field visibility/naming rules decide the list of available
fields for a given struct type. The mapping of parser type->Field is established
via the list of StructPropInfo objects that are handed to the Struct() function.

This list of props defines the parsing and validation rules that will be used by
the JSON parser for this type. Any field that is present on the struct but not
present in the props list, will be ignored and left untouched by the Parser.
*/
type StructParser struct {
	props []StructPropInfo
}

/*
Give it each of the properties you want to parse into struct fields.

For example:

Given this type
type Person struct {
	DBid int64
	Name string
	Age int
	Occupation *string
}

and this Parser

Struct(
	Prop("Name", String(MinLen(1), MaxLen(255))),
	PropWithDefault("Age", Integer(MinI(0)), -1),
	Prop("Occupation", String(MinLen(1), MaxLen(255))),
)

 - The Name property is mandatory, and must have a length between 1 and 255
   characters (inclusive)
 - The Age property is optional. When provded, its value must be 0 or greater.
   When not provided, it will be assigned a value of -1.
 - The Occupation property is optional (because it's a pointer field). Pointer
   fields are left untouched when a value is not provided. If provided, the
   value must be a string between 1 and 255 characters long.

This example shows how pointer vs non-pointer fields are handled and also
demonstrates the difference between a field with a default value and a truely
optional field. The last thing to note is that default values are not validated
by the validation rules of the Parsing sub-type. As such, default values can be
a value that won't be allowed to be explicitely set. This is sometimes
preferable to using a Pointer, as the field can be used without having to
dereference and for writing to a DB, a nil is obviously different to a default
value.
*/
func Struct(props ...StructPropInfo) *StructParser {
	return &StructParser{props}
}

/*
We cache all the field lookup info here.
*/
func (p *StructParser) Prepare(t reflect.Type) error {
	// make sure it's a struct
	if t.Kind() != reflect.Struct {
		return fmt.Errorf(ERROR_BAD_OBJ_DEST, t)
	}

	// fill in the field for each prop
	fields := typeFields(t)
	for i := range fields {
		f := &fields[i]
		var prop *StructPropInfo

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

func (p *StructParser) getProp(name []byte) (int, *StructPropInfo) {
	// get the property
	var prop *StructPropInfo
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
func (p *StructParser) Parse(path Pather, s *Scanner, v interface{}) error {
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
	// reused to reference the prop
	var prop *StructPropInfo
	var propIndex int
	propPath := func() string {
		return fmt.Sprintf("%s%s", path(), prop.f.name)
	}

	for {
		// read the key, or '}'
		if tok, keyb, err := s.ReadToken(); tok == tokenError {
			return err
		} else if tok == tokenObjectEnd {
			break
		} else if tok != tokenString {
			return NewParseError("Expected object property name or '}' not " + tok.String())
		} else {
			// get the appropriate prop
			// we do this now, because ReadToken will invalidate keyb
			propIndex, prop = p.getProp(keyb[1 : len(keyb)-1])
			if prop != nil {
			}
		}

		// read the ':'
		if tok, _, err := s.ReadToken(); tok == tokenError {
			return err
		} else if tok != tokenPropSep {
			return NewParseError("Expected ':' not " + tok.String())
		}

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
			errs = errs.Add(path()+p.props[i].f.name, ERROR_PROP_REQUIRED)
		}
	}

	if len(errs) > 0 {
		return errs
	} else {
		return nil
	}
}
