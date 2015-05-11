package jsonv

import (
	"testing"
)

func Test_StringValidators(t *testing.T) {
	cases := []struct {
		v       StringValidator
		val     string
		isValid bool
	}{
		{MinLen(0), "", true},
		{MinLen(1), "", false},
		{MinLen(1), "A", true},
		{MinLen(1), "Apple", true},
		{MinLen(72), "A whole lot of letters so that we can be sure something is in there.....", true},

		{MaxLen(0), "", true},
		{MaxLen(0), "z", false},
		{MaxLen(1), "", true},
		{MaxLen(1), "sasas", false},

		{Pattern("[a-z]+", "Must be at least one lowercase letter"), "sasas", true},
		{Pattern("[a-z]+", ""), "SASASA", false},
		{Pattern("[a-z]+", ""), "   sasas     ", true},    // should be non-anchored
		{Pattern("^[a-z]+", ""), "sasas     ", true},      // but can be
		{Pattern("^[a-z]+", ""), "    sasas     ", false}, // but can be
		{Pattern("[a-z]+$", ""), "   sasas", true},
		{Pattern("[a-z]+$", ""), "   sasas     ", false},
		{Pattern("Z[a-z]+", ""), "Zsasas", true},
		{Pattern("Z[a-z]+", ""), "sasas", false},
	}

	for i, c := range cases {
		err := c.v.ValidateString(c.val)
		if !c.isValid && err == nil {
			t.Errorf("Case %d, Val %d: Got no error, wanted one", i, c.val)
		} else if c.isValid && err != nil {
			t.Errorf("Case %d, Val %d: Got error \"%v\", wanted nil", i, c.val, err)
		}
	}
}
