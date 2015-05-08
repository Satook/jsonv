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

	if ps, ok := t.(PreparedSchemaType); ok {
		destType := reflect.Indirect(reflect.ValueOf(want)).Type()
		if err := ps.Prepare(destType); err != nil {
			return err
		}
	}

	if err := t.Parse("/", s, dest); err != nil {
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
	type ptrStruct struct {
		Name  string
		Other *string
	}

	bobStr := "Bob"

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
		// with extra complex prop that was not requested
		{Object(Prop("Captcha", String())),
			`{"Captcha": "Zing", "Fullname":{"favs": [1,2,3], "zing": "zong"} }`, simpleStruct{"Zing", ""}},

		{Slice(Object(Prop("Captcha", String()))),
			`[{"Captcha": "Zings", "Fullname":"Bobs" }]`, []simpleStruct{{"Zings", ""}}},
		{Slice(Integer()),
			`[1,2,3,45, -12]`, []int64{1, 2, 3, 45, -12}},

		// test that a struct with Pointer attrs is handled properly
		{Object(
			Prop("Name", String()),
			Prop("Other", String()),
		), `{"Name": "Zing", "Other":"Bob" }`, ptrStruct{"Zing", &bobStr}},
		// test that nils come across properly
		{Object(
			Prop("Name", String()),
			Prop("Other", String()),
		), `{"Name": "Zing"}`, ptrStruct{"Zing", nil}},
	}

	for i, c := range cases {
		destPtr := reflect.New(reflect.TypeOf(c.want))
		if err := tryParse(c.t, c.json, destPtr.Interface(), c.want); err != nil {
			t.Errorf("Case %d %v", i, err)
		}

		got := destPtr.Elem().Interface()
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Case %d: Got %+v, want %+v", i, got, c.want)
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
		{Integer(), "a", new(int64)},
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

func Test_SchemaTypeValidationErrors(t *testing.T) {
	// each case provides data that will fail validation
	cases := []struct {
		t         SchemaType
		json      string
		dest      interface{}
		wantPaths []string
	}{
		{Integer(), "5.2", new(int64), []string{"/"}},
		{Integer(MinI(7)), "5", new(int64), []string{"/"}},
		{Integer(MaxI(3)), "5", new(int64), []string{"/"}},

		{String(MaxLen(2)), `"TOo long"`, new(string), []string{"/"}},

		// check the slice validators
		{Slice(Integer(), MinLen(2)), "[]", new([]int64), []string{"/"}},
		{Slice(Integer(), MinLen(2)), "[1]", new([]int64), []string{"/"}},
		{Slice(Integer(), MaxLen(1)), "[1,2,3]", new([]int64), []string{"/"}},
		// check slice also collects up validation errors from sub-types
		{Slice(Integer(MaxI(5))), "[1,7,3]", new([]int64), []string{"/1/"}},
		{Slice(Integer(MaxI(5))), "[12,1,7,3]", new([]int64), []string{"/0/", "/2/"}},

		// check object validators
		//  required fields
		{Object(Prop("Captcha", String()), Prop("Fullname", String())),
			`{"Captcha": "Zing"}`, new(simpleStruct), []string{"/Fullname"}},
		{Object(Prop("Captcha", String()), Prop("Fullname", String())),
			`{}`, new(simpleStruct), []string{"/Captcha", "/Fullname"}},

		// check object collects up validation errors from sub-types
		{Object(Prop("Captcha", String(MaxLen(2)))),
			`{"Captcha": "Zing"}`, new(simpleStruct), []string{"/Captcha"}},
	}

	for i, c := range cases {
		t.Logf("Starting case %d", i)

		// see if we get a error as expected
		if err := tryParse(c.t, c.json, c.dest, c.dest); err == nil {
			t.Errorf("Case %d Valid: Didn't get any error", i)
		} else {
			t.Log(err)
			verr := err.(ValidationError)

			gotPaths := make([]string, len(verr))
			for i, e := range verr {
				gotPaths[i] = e.Path
			}

			if !reflect.DeepEqual(gotPaths, c.wantPaths) {
				t.Errorf("Got paths %v, want %v", gotPaths, c.wantPaths)
			}
		}
	}
}
