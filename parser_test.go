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
