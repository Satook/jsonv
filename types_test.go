package jsonschema

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
	}

	for i, c := range cases {
		dest := reflect.New(reflect.TypeOf(c.want)).Interface()
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
	}

	for i, c := range cases {
		// see if we get a validation error correctly
		if err := tryParse(c.t, c.json, c.dest, c.dest); err == nil {
			t.Errorf("Case %d Valid: Didn't get any error", i)
		} else if _, ok := err.(ValidationError); !ok {
			t.Errorf("Case %d Valid: Got non-validation error %v, %v", i, reflect.TypeOf(err), err)
		}

		// see if it handles unexpected EOF correctly
		s := NewScanner(&EOFReader{})
		if err := c.t.Parse("", s, c.dest); err == nil {
			t.Errorf("Case %d EOF: Didn't get any error", i)
		} else if err != io.EOF {
			t.Errorf("Case %d EOF: Got non-EOF error %v", i, err)
		}

		// see if it handles random shitty error correctly
		s = NewScanner(&ErrorReader{})
		if err := c.t.Parse("", s, c.dest); err == nil {
			t.Errorf("Case %d RandomError: Didn't get any error", i)
		}
	}
}
