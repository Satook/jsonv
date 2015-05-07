package jsonv

import (
	"bytes"
	"reflect"
	"testing"
)

type simpleStruct struct {
	Captcha  string
	Fullname string
}

func Test_ParseSimpleSuccess(t *testing.T) {
	cases := []struct {
		schema SchemaType
		json   string
		want   interface{}
	}{
		{Integer(), "123", int64(123)},
		{Boolean(), "true", true},
		{
			Object(
				Prop("Captcha", String()),
				Prop("Fullname", String()),
			),
			`{"Captcha": "Zing", "Fullname":"Bob" }`,
			simpleStruct{"Zing", "Bob"},
		},
	}

	for i, c := range cases {
		destPtr := reflect.New(reflect.TypeOf(c.want)) // allocate a fresh object, same type as c.want
		parser := Parser(destPtr.Interface(), c.schema)

		t.Logf("Running parser")
		if err := parser.Parse(bytes.NewBufferString(c.json), destPtr.Interface()); err != nil {
			t.Errorf("Error in case %d: %v", i, err)
			continue
		}

		dest := destPtr.Elem().Interface()
		if !reflect.DeepEqual(dest, c.want) {
			t.Errorf("Got %v, want %v", dest, c.want)
		}
	}
}

// Bad types tests
// Want to make sure all the different parsers are capable of checking the types
// at construction time, not only at parsing time.

func Test_parserBadTypes(t *testing.T) {
	type dumbStruct struct {
		Silly string
	}
	type intName struct {
		Name int64
	}

	cases := []struct {
		s SchemaType
		t interface{}
	}{
		// straight type checks
		{Integer(), new(int8)},
		{Integer(), new(float64)},
		{Boolean(), new(float64)},
		{String(), new(float64)},
		{Object(), new(float64)},
		{Slice(Object()), new(float64)},

		// nested type checks
		// dest type have all the props
		{Object(
			Prop("Name", String()),
		), new(dumbStruct)},
		{Object(
			Prop("Name", String()),
			Prop("Silly", String()),
		), new(dumbStruct)},
		// dest type props must have a type that each prop parser can map to
		{Object(Prop("Name", String())), new(intName)},

		// slices too!
		{Slice(Object(Prop("Name", String()))), make([]dumbStruct, 0, 10)},
		{Slice(Object(Prop("Name", String()))), make([]intName, 0, 10)},
	}

	for i, c := range cases {
		if _, err := ParserError(c.t, c.s); err == nil {
			t.Errorf("Case %d: Expected error, got nil", i)
		}
	}
}
