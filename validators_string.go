package jsonschema

import (
	"fmt"
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
	if len(s) < m.l {
		return fmt.Errorf(ERROR_MAX_LEN_STR, m.l)
	}
	return nil
}
