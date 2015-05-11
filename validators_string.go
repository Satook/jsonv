package jsonv

import (
	"fmt"
	"regexp"
)

type StringValidator interface {
	ValidateString(s string) error
}

type StringValidatorFunc func(s string) error

func (f StringValidatorFunc) ValidateString(s string) error {
	return f(s)
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

type PatternV struct {
	r *regexp.Regexp
}

func Pattern(re string) *PatternV {
	return &PatternV{regexp.MustCompile(re)}
}

func (p *PatternV) ValidateString(s string) error {
	if p.r.MatchString(s) {
		return nil
	} else {
		return fmt.Errorf(ERROR_PATTERN_MATCH, p.r.String())
	}
}
