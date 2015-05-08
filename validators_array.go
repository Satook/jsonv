package jsonv

import (
	"fmt"
	"reflect"
)

type SliceValidator interface {
	ValidateSlice(reflect.Value) error
}

type SliceValidatorFunc func(reflect.Value) error

func (f SliceValidatorFunc) ValidateSlice(v reflect.Value) error {
	return f(v)
}

func (m *MinLenV) ValidateSlice(v reflect.Value) error {
	if v.Len() < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_ARR, m.l)
	}
	return nil
}

func (m *MaxLenV) ValidateSlice(v reflect.Value) error {
	if v.Len() > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_ARR, m.l)
	}
	return nil
}
