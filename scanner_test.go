package jsonschema

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_scannerTokens(t *testing.T) {
	cases := []struct {
		json string
		tok  TokenType
		val  []byte
	}{
		{"{", tokenObjectBegin, []byte("{")},
		{" {", tokenObjectBegin, []byte("{")},
		{"\t{", tokenObjectBegin, []byte("{")},
		{"\n{", tokenObjectBegin, []byte("{")},
		{"\r{", tokenObjectBegin, []byte("{")},
		{" \t\n\n\r\t   { \t\t", tokenObjectBegin, []byte("{")},
		{"}", tokenObjectEnd, []byte("}")},
		{"[", tokenArrayBegin, []byte("[")},
		{"]", tokenArrayEnd, []byte("]")},
		{`  , `, tokenItemSep, []byte(",")},
		{`  : `, tokenPropSep, []byte(":")},
		{"true", tokenTrue, []byte("true")},
		{"false", tokenFalse, []byte("false")},
		{"null", tokenNull, []byte("null")},
		{"0", tokenNumber, []byte("0")},
		{"5", tokenNumber, []byte("5")},
		{"-5", tokenNumber, []byte("-5")},
		{"0.1", tokenNumber, []byte("0.1")},
		{"-0.1", tokenNumber, []byte("-0.1")},
		{"0.123", tokenNumber, []byte("0.123")},
		{"1234567890", tokenNumber, []byte("1234567890")},
		{"2e+12", tokenNumber, []byte("2e+12")},
		{"2e-12", tokenNumber, []byte("2e-12")},
		{"2e12", tokenNumber, []byte("2e12")},
		{"2.3e+9", tokenNumber, []byte("2.3e+9")},
		{"0.2e-5", tokenNumber, []byte("0.2e-5")},
		{"0.2e5", tokenNumber, []byte("0.2e5")},
		{",", tokenItemSep, []byte(",")},
		{`""`, tokenString, []byte(`""`)},
		{`"Abc"`, tokenString, []byte(`"Abc"`)},
		{`"A\"b\\c"`, tokenString, []byte(`"A\"b\\c"`)},
		{`"\"A\"b\\c"`, tokenString, []byte(`"\"A\"b\\c"`)},
		{`  "Abc"  `, tokenString, []byte(`"Abc"`)},
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
			t.Errorf("Case %d val: Got %v, want %v", i, b, c.val)
		}
	}
}
