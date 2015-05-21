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

type trainer struct {
	Captcha  string
	Fullname string
	Email    string
	Mobile   string
	Password string
}

func Test_SchemaTypeParse(t *testing.T) {
	type ptrStruct struct {
		Name  string
		Other *string
	}

	type manyStruct struct {
		Name  string
		IVal  int64
		BVal  bool
		SlVal []string
		StVal simpleStruct
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
		{Boolean(), "true", "true"},
		{Boolean(), "false", "false"},

		{String(), `"false"`, "false"},
		{String(), `"Something with \n \\ "`, "Something with \n \\ "},
		{String(), `"Unicode!! \u2318"`, "Unicode!! \u2318"},

		{Bytes(), `"false"`, []byte("false")},
		{Bytes(), `"Something with \n \\ "`, []byte("Something with \n \\ ")},

		{RawBytes(), `"false"`, []byte("false")},
		{RawBytes(), `"Something with \n \\ "`, []byte("Something with \\n \\\\ ")},

		// with all props
		{Object(Prop("Captcha", String()), Prop("Fullname", String())),
			`{"Captcha": "Zing", "Fullname":"Bob" }`, simpleStruct{"Zing", "Bob"}},
		// with extra prop (on struct but not requested
		{Object(Prop("Captcha", String())),
			`{"Captcha": "Zing", "Fullname":"Bob" }`, simpleStruct{"Zing", ""}},
		// with extra complex prop that was not requested
		{Object(Prop("Captcha", String())),
			`{"Captcha": "Zing", "Fullname":{"favs": [1,2,3], "zing": "zong"} }`, simpleStruct{"Zing", ""}},

		// structs with default props
		{Object(PropWithDefault("Name", String(), "Weee")), `{}`, manyStruct{Name: "Weee"}},
		{Object(PropWithDefault("IVal", Integer(), int64(76))), `{}`, manyStruct{IVal: 76}},
		{Object(PropWithDefault("BVal", Boolean(), true)), `{}`, manyStruct{BVal: true}},
		{Object(PropWithDefault("SlVal", Slice(String()), []string{"dood", "wood"})), `{}`, manyStruct{SlVal: []string{"dood", "wood"}}},
		{Object(PropWithDefault("StVal", Object(Prop("Captcha", String())), simpleStruct{"Zing", ""})), `{}`, manyStruct{StVal: simpleStruct{"Zing", ""}}},

		// mix default and non
		{Object(PropWithDefault("Name", String(), "Weee"), Prop("IVal", Integer())), `{"IVal": 12}`, manyStruct{Name: "Weee", IVal: 12}},

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

		// big enough to force a buffer re-size mid string.
		{Object(
			PropWithDefault("Captcha", String(), ""),
			Prop("Fullname", String()),
			Prop("Email", String()),
			Prop("Mobile", String()),
			Prop("Password", String()),
		),
			`{"Fullname":"kjsadhlkfjdshalkhjdfsa","Mobile":"2309485702349857","Email":"laksdjfh@asdlkihfalsdkifhj","Password":"alksdjfghlaksdf","Captcha":"03AHJ_VutuNyz928BySmbXvafmtG90YdwZdYCTCN0FYLE2IWnzXlpqb1GVAVmggjrMQqXak0mQMZQK5JI5y-5kfZcImtTjFW3tizGPU-RyBgrZ2mLXtZplYGBdRjHA7WHVrKuD4rjtJtZ6DOnGxwceNDJCdeaJopGFujvDqxMADt-ovlWC9_vLVfvjo-y_1hO0Wdw_QbWzPqeKy0FLGN5pv-dTnmd9WcwN2EW54V8Y4RkPnEMWgnzlJIdzVNoFpkHysQ_jR_jE1FfPQt5ZSbQw3Ey3p1dPSFp_ee7vSyk9QMyIqbgRXhB5kOXTCil87Oq6Fb76Y8cBt-hMzO8c8uk_aoWS0QdOTGvMtx1blQPECsCbAUjzuKHilH6beECyJzgA6nFQytQ2Ne1Dz1-y6ML6wg6ANeeAPjojbIo5xZGGXnY5ruzahIhsTZY"}`,
			trainer{"03AHJ_VutuNyz928BySmbXvafmtG90YdwZdYCTCN0FYLE2IWnzXlpqb1GVAVmggjrMQqXak0mQMZQK5JI5y-5kfZcImtTjFW3tizGPU-RyBgrZ2mLXtZplYGBdRjHA7WHVrKuD4rjtJtZ6DOnGxwceNDJCdeaJopGFujvDqxMADt-ovlWC9_vLVfvjo-y_1hO0Wdw_QbWzPqeKy0FLGN5pv-dTnmd9WcwN2EW54V8Y4RkPnEMWgnzlJIdzVNoFpkHysQ_jR_jE1FfPQt5ZSbQw3Ey3p1dPSFp_ee7vSyk9QMyIqbgRXhB5kOXTCil87Oq6Fb76Y8cBt-hMzO8c8uk_aoWS0QdOTGvMtx1blQPECsCbAUjzuKHilH6beECyJzgA6nFQytQ2Ne1Dz1-y6ML6wg6ANeeAPjojbIo5xZGGXnY5ruzahIhsTZY", "kjsadhlkfjdshalkhjdfsa", "laksdjfh@asdlkihfalsdkifhj", "2309485702349857", "alksdjfghlaksdf"}},
	}

	for i, c := range cases {
		t.Logf("Starting case %d", i)
		destPtr := reflect.New(reflect.TypeOf(c.want))
		if err := tryParse(c.t, c.json, destPtr.Interface(), c.want); err != nil {
			t.Fatalf("Case %d %v", i, err)
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
		{Slice(Integer(), MinItems(2)), "[]", new([]int64), []string{"/"}},
		{Slice(Integer(), MinItems(2)), "[1]", new([]int64), []string{"/"}},
		{Slice(Integer(), MaxItems(1)), "[1,2,3]", new([]int64), []string{"/"}},
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
