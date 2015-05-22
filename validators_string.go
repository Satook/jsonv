package jsonv

import (
	"fmt"
	"regexp"
)

type StringValidator interface {
	ValidateString(s string) error
}

type BytesValidator interface {
	ValidateBytes(b []byte) error
}

type StringValidatorFunc func(s string) error

func (f StringValidatorFunc) ValidateString(s string) error {
	return f(s)
}

type BytesValidatorFunc func(b []byte) error

func (f BytesValidatorFunc) ValidateBytes(b []byte) error {
	return f(b)
}

/*
The Min Length validator.
*/
type MinLenV struct {
	l int
}

func MinLen(l int) *MinLenV {
	if l < 0 {
		panic(fmt.Errorf("Minimum allowed length must be >= 0"))
	}
	return &MinLenV{l}
}

func (m *MinLenV) ValidateString(s string) error {
	if len(s) < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_STR, m.l)
	}
	return nil
}

func (m *MinLenV) ValidateBytes(b []byte) error {
	if len(b) < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_STR, m.l)
	}
	return nil
}

/*
The Max Length validator.
*/
type MaxLenV struct {
	l int
}

func MaxLen(l int) *MaxLenV {
	if l < 0 {
		panic(fmt.Errorf("Maximum allowed length must be >= 0"))
	}
	return &MaxLenV{l}
}

func (m *MaxLenV) ValidateString(s string) error {
	if len(s) > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_STR, m.l)
	}
	return nil
}

func (m *MaxLenV) ValidateBytes(b []byte) error {
	if len(b) > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_STR, m.l)
	}
	return nil
}

type PatternV struct {
	r   *regexp.Regexp
	msg string
}

/*
Builds a regex pattern based string validator.

re: The regex string used for validation.
message: A human friendly message to use in the ValidationError

Note: Will panic if re fails to compile.
*/
func Pattern(re, message string) *PatternV {
	return &PatternV{regexp.MustCompile(re), message}
}

func (p *PatternV) ValidateString(s string) error {
	if p.r.MatchString(s) {
		return nil
	} else {
		return fmt.Errorf("%v", p.msg)
	}
}
