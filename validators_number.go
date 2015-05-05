package jsonv

import (
	"fmt"
	"math"
)

const ()

/*
Used to identify validators that can work on Integer values.
*/
type IntegerValidator interface {
	ValidateInteger(i int64) error
}

type IntegerValidatorFunc func(i int64) error

func (f IntegerValidatorFunc) ValidateInteger(i int64) error {
	return f(i)
}

type FloatValidator interface {
	ValidateFloat(f float64) error
}

type FloatValidatorFunc func(i float64) error

func (f FloatValidatorFunc) ValidateFloat(i float64) error {
	return f(i)
}

/*
Minimum int value validator.

Values must be >= m.
*/
func MinI(m int64) IntegerValidator {
	return IntegerValidatorFunc(func(i int64) error {
		if i >= m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MIN, m)
		}
	})
}

/*
Exclusive int minimum value validator.

Values must be > m.
*/
func MinEI(m int64) IntegerValidator {
	return IntegerValidatorFunc(func(i int64) error {
		if i > m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MIN_EX, m)
		}
	})
}

/*
Maximum int value validator.

Values must be <= m.
*/
func MaxI(m int64) IntegerValidator {
	return IntegerValidatorFunc(func(i int64) error {
		if i <= m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MAX, m)
		}
	})
}

/*
Exclusive int maximum value validator.

Values must be < m.
*/
func MaxEI(m int64) IntegerValidator {
	return IntegerValidatorFunc(func(i int64) error {
		if i < m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MAX_EX, m)
		}
	})
}

/*
Validates that the integer value is a multiple of another integer.
*/
func MulOfI(m int64) IntegerValidator {
	if m <= 0 {
		panic(fmt.Errorf("Multiple must be >= 0, %v is not valid", m))
	}
	return IntegerValidatorFunc(func(i int64) error {
		if i%m == 0 {
			return nil
		} else {
			return fmt.Errorf(ERROR_MULOF, m)
		}
	})
}

/*
Minimum float value validator.

Values must be >= m.
*/
func MinF(m float64) FloatValidator {
	return FloatValidatorFunc(func(i float64) error {
		if i >= m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MIN, m)
		}
	})
}

/*
Exclusive float minimum value validator.

Values must be > m.
*/
func MinEF(m float64) FloatValidator {
	return FloatValidatorFunc(func(f float64) error {
		if f > m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MIN_EX, m)
		}
	})
}

/*
Maximum float value validator.

Values must be <= m.
*/
func MaxF(m float64) FloatValidator {
	return FloatValidatorFunc(func(f float64) error {
		if f <= m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MAX, m)
		}
	})
}

/*
Exclusive float maximum value validator.

Values must be < m.
*/
func MaxEF(m float64) FloatValidator {
	return FloatValidatorFunc(func(i float64) error {
		if i < m {
			return nil
		} else {
			return fmt.Errorf(ERROR_MAX_EX, m)
		}
	})
}

/*
Validates that the integer value is a multiple of another integer.
*/
func MulOfF(m float64) FloatValidator {
	if m <= 0 || math.IsInf(m, 0) || math.IsNaN(m) {
		panic(fmt.Errorf("Multiple must be >= 0, %v is not valid", m))
	}
	return FloatValidatorFunc(func(f float64) error {
		if math.Mod(f, m) == 0 {
			return nil
		} else {
			return fmt.Errorf(ERROR_MULOF, m)
		}
	})
}
