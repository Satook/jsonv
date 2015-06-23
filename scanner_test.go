package jsonv

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func Test_scannerTokens(t *testing.T) {
	cases := []struct {
		json string
		tok  TokenType
		val  []byte
	}{
		{"{", TokenObjectBegin, []byte("{")},
		{" {", TokenObjectBegin, []byte("{")},
		{"\t{", TokenObjectBegin, []byte("{")},
		{"\n{", TokenObjectBegin, []byte("{")},
		{"\r{", TokenObjectBegin, []byte("{")},
		{" \t\n\n\r\t   { \t\t", TokenObjectBegin, []byte("{")},
		{"}", TokenObjectEnd, []byte("}")},
		{"[", TokenArrayBegin, []byte("[")},
		{"]", TokenArrayEnd, []byte("]")},
		{`  , `, TokenItemSep, []byte(",")},
		{`  :, `, TokenPropSep, []byte(":")},
		{"true", TokenTrue, []byte("true")},
		{"false,", TokenFalse, []byte("false")},
		{"null", TokenNull, []byte("null")},
		{"0 ", TokenNumber, []byte("0")},
		{"5 ", TokenNumber, []byte("5")},
		{"-5,", TokenNumber, []byte("-5")},
		{"0.1,", TokenNumber, []byte("0.1")},
		{"-0.1 ", TokenNumber, []byte("-0.1")},
		{"0.123 ", TokenNumber, []byte("0.123")},
		{"1234567890  ", TokenNumber, []byte("1234567890")},
		{"2e+12", TokenNumber, []byte("2e+12")},
		{"2e-12", TokenNumber, []byte("2e-12")},
		{"2e12", TokenNumber, []byte("2e12")},
		{"2.3e+9", TokenNumber, []byte("2.3e+9")},
		{"0.2e-5", TokenNumber, []byte("0.2e-5")},
		{"0.2e5", TokenNumber, []byte("0.2e5")},
		{",", TokenItemSep, []byte(",")},
		{`""`, TokenString, []byte(`""`)},
		{`"Abc"`, TokenString, []byte(`"Abc"`)},
		{`"A\"b\\c"`, TokenString, []byte(`"A\"b\\c"`)},
		{`"\"A\"b\\c"`, TokenString, []byte(`"\"A\"b\\c"`)},
		{`  "Abc"  `, TokenString, []byte(`"Abc"`)},
	}

	for i, c := range cases {
		t.Logf("Starting case: %d\n", i)
		s := NewScanner(bytes.NewBufferString(c.json))

		tok, b, err := s.ReadToken()
		if err != nil {
			t.Errorf("Case %d error: %v", i, err)
		} else if tok != c.tok {
			t.Errorf("Case %d token: Got %v, want %v", i, tok, c.tok)
		} else if !reflect.DeepEqual(b, c.val) {
			t.Errorf("Case %d val: Got \"%s\", want \"%s\"", i, b, c.val)
		}
	}
}

// test skipValue
// Used by Object when it needs to jump an unneeded property.
//
// Test
//	skip null, string, number, bool, array, object {}, object {props}, object {{},{},{}}
//
func Test_scannerSkipValue(t *testing.T) {
	cases := []string{
		`{"fake": null, "actual": "test"}`,
		`{"fake": false, "actual": "test"}`,
		`{"fake": true, "actual": "test"}`,
		`{"fake": "a string", "actual": "test"}`,
		`{"fake": "\"", "actual": "test"}`,
		`{"fake": 123123123, "actual": "test"}`,
		`{"fake": 12.2, "actual": "test"}`,
		`{"fake": -12.2e23, "actual": "test"}`,
		`{"fake": [], "actual": "test"}`,
		`{"fake": [{},{}], "actual": "test"}`,
		`{"fake": [1,true, null], "actual": "test"}`,
		`{"fake": {}, "actual": "test"}`,
		`{"fake": {"diff": "val", "age": 42}, "actual": "test"}`,
		`{"fake": {"diff": "val", "age": 42, "sub": {}}, "actual": "test"}`,
		`{"fake": {"diff": "val", "age": 42, "sub": {"has": null}}, "actual": "test"}`,
	}

	want1 := []TokenType{TokenObjectBegin, TokenString, TokenPropSep}
	want2 := []TokenType{TokenItemSep, TokenString, TokenPropSep, TokenString, TokenObjectEnd}

	for i, json := range cases {
		t.Logf("Starting case %d: %s\n", i, json)
		s := NewScanner(bytes.NewBufferString(json))

		// read the first bits
		for _, w := range want1 {
			if tok, _, err := s.ReadToken(); tok != w {
				if err != nil {
					t.Fatal(err)
				} else {
					t.Fatalf("Got token: %v, want %v", tok, w)
				}
				return
			}
		}

		// skip a value (complex or whatever)
		if err := s.SkipValue(); err != nil {
			t.Fatal(err)
		}

		// finish up
		for _, w := range want2 {
			if tok, _, err := s.ReadToken(); tok != w {
				if err != nil {
					t.Fatal(err)
				} else {
					t.Fatalf("Got token: %v, want %v", tok, w)
				}
				return
			}
		}

		// make sure we're at the end
		if tok, buf, err := s.ReadToken(); err != io.EOF {
			t.Fatalf("Got token: %v, buf %v, err %v, want EOF", tok, buf, err)
		}
	}
}

func Test_scannerLargeSource(t *testing.T) {
	data1 := []byte(`{"Name": "Angelo","Age":24,"Friends":["Bob","Jim","Jenny"]}`)
	data := make([]byte, len(data1)*1024+2+1023)
	for i := 0; i < 1024; i++ {
		offset := 1 + (len(data1)+1)*i
		copy(data[offset:], data1)
		data[offset+len(data1)] = ','
	}
	data[0] = '['
	data[len(data)-1] = ']'

	wantToks := []TokenType{TokenObjectBegin,
		TokenString, TokenPropSep, TokenString, TokenItemSep,
		TokenString, TokenPropSep, TokenNumber, TokenItemSep,
		TokenString, TokenPropSep, TokenArrayBegin,
		TokenString, TokenItemSep, TokenString, TokenItemSep, TokenString,
		TokenArrayEnd, TokenObjectEnd, TokenItemSep,
	}
	lenWantToks := len(wantToks)

	// read 1024 objects + ',' chars without a trailing ',' char
	toksToRead := lenWantToks*1024 - 1

	// start scanning
	s := NewScanner(bytes.NewReader(data))

	// read array start
	tok, _, err := s.ReadToken()
	if tok != TokenArrayBegin {
		t.Fatalf("Got %v, err %v. Want %v", tok, err, TokenArrayBegin)
	}

	for i := 0; i < toksToRead; i++ {
		got, buf, err := s.ReadToken()
		if got == TokenError {
			t.Fatal(err)
		}

		t.Logf("token: %v %s", got, buf)

		want := wantToks[i%lenWantToks]
		if got != want {
			t.Fatalf("Token %d: Got %v, want %v", i, got, want)
		}
	}

	// read the array end
	if tok, _, err := s.ReadToken(); tok != TokenArrayEnd {
		t.Fatalf("Got %v, err %v. Want %v", tok, err, TokenArrayEnd)
	}
}
