package jsonv

import (
	"fmt"
	"reflect"
	"strings"
)

/*
Parses a value using the provided parser and then ensures it is one of the
provided values.

Makes use of reflect.DeepEqual for equality checking, so using simple types is
recommended for both sanity and performance.
*/
type EnumParser struct {
	schema      SchemaType    // how do we parse it
	allowedVals []interface{} // what values are acceptable
	invalidMsg  string        // pre-built "value not valid" error
}

/*
SchemaType must work with the types of the provided values.

The provided values must all have the same underlying types.

Any of the above issues will be reported when Prepare is called.
*/
func Enum(s SchemaType, vals ...interface{}) *EnumParser {
	// Get a string representation of each.
	// TODO: Check if imps MarshalJSON and use that representation.
	var parts []string
	for _, v := range vals {
		parts = append(parts, fmt.Sprint(v))
	}

	return &EnumParser{s, vals, fmt.Sprintf("Must be one of: %s", strings.Join(parts, ","))}
}

func (p *EnumParser) Prepare(t reflect.Type) error {
	if !t.Comparable() {
		return fmt.Errorf("Field must be comparable")
	}

	// check that all the vals types match up with this type
	for _, v := range p.allowedVals {
		vt := reflect.TypeOf(v)
		if !vt.ConvertibleTo(t) {
			return fmt.Errorf("All values be convertable to the field type.")
		}
	}

	// prepare our sub-type if we need to
	if ps, ok := p.schema.(PreparedSchemaType); ok {
		return ps.Prepare(t)
	}

	return nil
}

func (p *EnumParser) Parse(path Pather, s *Scanner, v interface{}) error {
	// parse it as normal
	if err := p.schema.Parse(path, s, v); err != nil {
		return err
	}

	// get a reflect.Value of the parsed out value (de-ref ptr if needed)
	vinf := reflect.Indirect(reflect.ValueOf(v)).Interface()

	// check it's one of the accepted values
	for _, val := range p.allowedVals {
		if reflect.DeepEqual(val, vinf) {
			return nil
		}
	}

	var errs ValidationError
	return errs.Add(path(), p.invalidMsg)
}
