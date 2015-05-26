package benchmarks

import (
	"bitbucket.org/activelytrain/jsonv"
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

type BasicStruct struct {
	Name    string
	Age     int64
	Friends []string
}

var structSchema = jsonv.Struct(
	jsonv.Prop("Name", jsonv.String()),
	jsonv.Prop("Age", jsonv.Integer()),
	jsonv.Prop("Friends", jsonv.Slice(jsonv.String())),
)

var basicParser = jsonv.Parser(&BasicStruct{}, structSchema)
var sliceParser = jsonv.Parser([]BasicStruct{}, jsonv.Slice(structSchema))

func Benchmark_ParseSimple(b *testing.B) {
	data := []byte(`{"Name": "Angelo","Age":24,"Friends":["Bob","Jim","Jenny"]}`)
	blen := len(data)

	b.RunParallel(func(pb *testing.PB) {
		var dest BasicStruct
		buf := bytes.NewReader(data)

		for pb.Next() {
			buf.Seek(0, 0)

			if err := basicParser.Parse(buf, &dest); err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(blen))
		}
	})
}

func Benchmark_STDParseSimple(b *testing.B) {
	data := []byte(`{"Name": "Angelo","Age":24,"Friends":["Bob","Jim","Jenny"]}`)
	blen := len(data)

	b.RunParallel(func(pb *testing.PB) {
		var dest BasicStruct

		for pb.Next() {
			if err := json.Unmarshal(data, &dest); err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(blen))
		}
	})
}

func Benchmark_ParseLarge(b *testing.B) {
	data1 := []byte(`{"Name": "Angelo","Age":24,"Friends":["Bob","Jim","Jenny"]}`)
	data := make([]byte, len(data1)*1024+2+1023)
	for i := 0; i < 1024; i++ {
		offset := 1 + (len(data1)+1)*i
		copy(data[offset:], data1)
		data[offset+len(data1)] = ','
	}
	data[0] = '['
	data[len(data)-1] = ']'
	blen := len(data)

	b.RunParallel(func(pb *testing.PB) {
		var dest []BasicStruct
		buf := bytes.NewReader(data)

		for pb.Next() {
			buf.Seek(0, 0)

			if err := sliceParser.Parse(buf, &dest); err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(blen))
		}
	})
}

func Benchmark_STDParseLarge(b *testing.B) {
	want1 := BasicStruct{"Angelo", 24, []string{"Bob", "Jim", "Jenny"}}
	data1 := []byte(`{"Name": "Angelo","Age":24,"Friends":["Bob","Jim","Jenny"]}`)
	data := make([]byte, len(data1)*1024+2+1023)
	want := make([]BasicStruct, 1024)
	for i := 0; i < 1024; i++ {
		offset := 1 + (len(data1)+1)*i
		copy(data[offset:], data1)
		data[offset+len(data1)] = ','
		want[i] = want1
	}
	data[0] = '['
	data[len(data)-1] = ']'
	blen := len(data)

	b.RunParallel(func(pb *testing.PB) {
		var dest []BasicStruct
		buf := bytes.NewReader(data)

		for pb.Next() {
			buf.Seek(0, 0)

			if err := json.Unmarshal(data, &dest); err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(blen))

			if !reflect.DeepEqual(dest, want) {
				b.Fatalf("Got != want")
			}
		}
	})
}
