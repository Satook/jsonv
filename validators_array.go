package jsonv

import (
	"fmt"
	"reflect"
)

type ArrayValidator interface {
	ValidateArray(i interface{}) error
}

func (m *MinLenV) ValidateArray(i interface{}) error {
	if reflect.ValueOf(i).Len() < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_ARR, m.l)
	}
	return nil
}

func (m *MaxLenV) ValidateArray(i interface{}) error {
	if reflect.ValueOf(i).Len() > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_ARR, m.l)
	}
	return nil
}
