package jsonv

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
)

type EOFReader struct {
}

func (r *EOFReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

type ErrorReader struct {
}

func (r *ErrorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("File is corrupt")
}

func tryParse(t SchemaType, json string, dest interface{}, want interface{}) error {
	s := NewScanner(bytes.NewBufferString(json))

	if err := t.Parse("", s, dest); err != nil {
		return err
	}

	// dest is a ptr, so get the actual value interface{}
	val := reflect.ValueOf(dest).Elem().Interface()
	if !reflect.DeepEqual(val, want) {
		return fmt.Errorf("val: Got %v, want %v", val, want)
	}

	return nil
}

func Test_SchemaTypeParse(t *testing.T) {
	cases := []struct {
		t    SchemaType
		json string
		want interface{}
	}{
		{Integer(), "24", int64(24)},
		{Integer(), "572", int64(572)},
		{Integer(), "-572", int64(-572)},

		{Boolean(), "true", true},
		{Boolean(), "false", false},

		{String(), `"false"`, "false"},
		{String(), `"Something with \n \\ "`, "Something with \n \\ "},
		{String(), `"Unicode!! \u2318"`, "Unicode!! \u2318"},

		// object
		// with all props
		{Object(Prop("Captcha", String()), Prop("Fullname", String())),
			`{"Captcha": "Zing", "Fullname":"Bob" }`, simpleStruct{"Zing", "Bob"}},
		// with extra prop (on struct but not requested
		{Object(Prop("Captcha", String())),
			`{"Captcha": "Zing", "Fullname":"Bob" }`, simpleStruct{"Zing", ""}},
	}

	for i, c := range cases {
		destType := reflect.TypeOf(c.want)
		if ps, ok := c.t.(PrecacheSchemaType); ok {
			if err := ps.Prepare(destType); err != nil {
				t.Error(err)
				continue
			}
		}

		dest := reflect.New(destType).Interface()
		if err := tryParse(c.t, c.json, dest, c.want); err != nil {
			t.Errorf("Case %d %v", i, err)
		}
	}
}

func Test_SchemaTypeParseErrors(t *testing.T) {
	// each case provides data that will fail validation
	cases := []struct {
		t    SchemaType
		json string
		dest interface{}
	}{
		{Integer(), "5.2", new(int64)},
		{Integer(MinI(7)), "5", new(int64)},
		{Integer(MaxI(3)), "5", new(int64)},

		{Boolean(), "twwrue", new(bool)},
		{Boolean(), "1", new(bool)},
	}

	for i, c := range cases {
		// see if we get a error as expected
		if err := tryParse(c.t, c.json, c.dest, c.dest); err == nil {
			t.Errorf("Case %d Valid: Didn't get any error", i)
		}

		// see if it handles unexpected EOF correctly
		s := NewScanner(&EOFReader{})
		if err := c.t.Parse("", s, c.dest); err != io.EOF {
			t.Errorf("Case %d EOF: Got non-EOF error %v", i, err)
		}

		// see if it handles random shitty error correctly
		s = NewScanner(&ErrorReader{})
		if err := c.t.Parse("", s, c.dest); err == nil {
			t.Errorf("Case %d RandomError: Didn't get any error", i)
		} else if _, ok := err.(ValidationError); ok {
			t.Errorf("Case %d RandomError: Got validation error %v, want IO error", i, err)
		}
	}
}
