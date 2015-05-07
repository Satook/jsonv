package jsonv

import (
	"fmt"
	"reflect"
)

type SliceValidator interface {
	ValidateSlice(i interface{}) error
}

type SliceValidatorFunc func(i interface{}) error

func (f SliceValidatorFunc) ValidateSlice(i interface{}) error {
	return f(i)
}

func (m *MinLenV) ValidateSlice(i interface{}) error {
	if reflect.ValueOf(i).Len() < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_ARR, m.l)
	}
	return nil
}

func (m *MaxLenV) ValidateSlice(i interface{}) error {
	if reflect.ValueOf(i).Len() > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_ARR, m.l)
	}
	return nil
}
