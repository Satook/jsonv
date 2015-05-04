package jsonschema

import (
	"fmt"
	"regexp"
)

type StringValidator interface {
	ValidateString(s string) error
}

func (m *MinLenV) ValidateString(s string) error {
	if len(s) < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_STR, m.l)
	}
	return nil
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

func (p *PatternV) ValidateString(s string) error {
	if p.r.MatchString(s) {
		return nil
	} else {
		return fmt.Errorf(ERROR_PATTERN_MATCH, p.r.String())
	}
}

func Pattern(re string) *PatternV {
	return &PatternV{regexp.MustCompile(re)}
}
