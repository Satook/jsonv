package jsonv

import (
	"testing"
)

func Test_IntValidators(t *testing.T) {
	cases := []struct {
		v       IntegerValidator
		val     int64
		isValid bool
	}{
		// max value tests
		{MaxI(0), 0, true},
		{MaxI(0), 1, false},
		{MaxI(10001), 10001, true},
		{MaxI(10001), 10000, true},
		{MaxI(10001), 10002, false},
		{MaxI(10001), 10003, false},
		{MaxI(-10001), -10002, true},
		{MaxI(-10001), -10001, true},
		{MaxI(-10001), -10000, false},
		{MaxEI(0), -1, true},
		{MaxEI(0), 0, false},
		{MaxEI(0), 1, false},
		{MaxEI(3568989), 3568988, true},
		{MaxEI(3568989), 3568989, false},

		// Min value tests
		{MinI(0), 0, true},
		{MinI(0), -1, false},
		{MinI(10001), 10001, true},
		{MinI(10001), 10002, true},
		{MinI(10001), 10000, false},
		{MinI(10001), 9999, false},
		{MinI(-10001), -10000, true},
		{MinI(-10001), -10001, true},
		{MinI(-10001), -10002, false},
		{MinEI(0), 1, true},
		{MinEI(0), 0, false},
		{MinEI(0), -1, false},
		{MinEI(3568989), 3568990, true},
		{MinEI(3568989), 3568989, false},

		// MulOf value tests
		{MulOfI(1), 9, true},
		{MulOfI(2), 2, true},
		{MulOfI(2), -74, true},
		{MulOfI(2), 237550682, true},
		{MulOfI(2), 9, false},
		{MulOfI(2), -9, false},
		{MulOfI(3), 9, true},
		{MulOfI(3), -9, true},
	}

	for i, c := range cases {
		err := c.v.ValidateInteger(c.val)
		if !c.isValid && err == nil {
			t.Errorf("Case %d, Val %d: Got no error, wanted one", i, c.val)
		} else if c.isValid && err != nil {
			t.Errorf("Case %d, Val %d: Got error \"%v\", wanted nil", i, c.val, err)
		}
	}
}

func Test_FloatValidators(t *testing.T) {
	cases := []struct {
		v       FloatValidator
		val     float64
		isValid bool
	}{
		// max value tests
		{MaxF(0), 0, true},
		{MaxF(0), 1, false},
		{MaxF(10001), 10001, true},
		{MaxF(10001), 10000, true},
		{MaxF(10001), 10002, false},
		{MaxF(10001), 10003, false},
		{MaxF(-10001), -10002, true},
		{MaxF(-10001), -10001, true},
		{MaxF(-10001), -10000, false},
		{MaxEF(0), -1, true},
		{MaxEF(0), 0, false},
		{MaxEF(0), 1, false},
		{MaxEF(3568989), 3568988, true},
		{MaxEF(3568989), 3568989, false},

		// Min value tests
		{MinF(0), 0, true},
		{MinF(0), -1, false},
		{MinF(10001), 10001, true},
		{MinF(10001), 10002, true},
		{MinF(10001), 10000, false},
		{MinF(10001), 9999, false},
		{MinF(-10001), -10000, true},
		{MinF(-10001), -10001, true},
		{MinF(-10001), -10002, false},
		{MinEF(0), 1, true},
		{MinEF(0), 0, false},
		{MinEF(0), -1, false},
		{MinEF(3568989), 3568990, true},
		{MinEF(3568989), 3568989, false},

		// MulOf value tests
		{MulOfF(1), 9, true},
		{MulOfF(2), 2, true},
		{MulOfF(2), -74, true},
		{MulOfF(2), 237550682, true},
		{MulOfF(2), 9, false},
		{MulOfF(2), -9, false},
		{MulOfF(3), 9, true},
		{MulOfF(3), -9, true},
	}

	for i, c := range cases {
		err := c.v.ValidateFloat(c.val)
		if !c.isValid && err == nil {
			t.Errorf("Case %d, Val %d: Got no error, wanted one", i, c.val)
		} else if c.isValid && err != nil {
			t.Errorf("Case %d, Val %d: Got error \"%v\", wanted nil", i, c.val, err)
		}
	}
}
