package jsonschema

import (
	"bytes"
	"reflect"
	"testing"
)

type trainer struct {
	Captcha      string
	Fullname     string
	Email        string
	Mobile       string
	Password     string
	PasswordHash []byte `json:"-"`
}

func Test_ParseSimpleSuccess(t *testing.T) {
	var testInt int64

	cases := []struct {
		baseType interface{}
		schema   SchemaType
		json     string
		want     interface{}
	}{
		{&testInt, Integer(), "123", int64(123)},
	}

	for i, c := range cases {
		parser := Parser(c.baseType, c.schema)
		destPtr := reflect.New(reflect.Indirect(reflect.ValueOf(c.baseType)).Type())
		destVal := destPtr.Elem()

		t.Logf("Running parser")
		if err := parser.Parse(bytes.NewBufferString(c.json), destPtr.Interface()); err != nil {
			t.Errorf("Error in case %d: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(destVal.Interface(), c.want) {
			t.Errorf("Got %v, want %v", destVal.Interface(), c.want)
		}
	}
}
