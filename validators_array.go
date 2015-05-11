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

/*
The Min Length validator.
*/
type MinItemsV struct {
	l int
}

func MinItems(l int) *MinItemsV {
	if l < 0 {
		panic(fmt.Errorf("Minimum allowed length must be >= 0"))
	}
	return &MinItemsV{l}
}

func (m *MinItemsV) ValidateSlice(v reflect.Value) error {
	if v.Len() < m.l {
		return fmt.Errorf(ERROR_MIN_LEN_ARR, m.l)
	}
	return nil
}

/*
The Max Length validator.
*/
type MaxItemsV struct {
	l int
}

func MaxItems(l int) *MaxItemsV {
	if l < 0 {
		panic(fmt.Errorf("Maximum allowed length must be >= 0"))
	}
	return &MaxItemsV{l}
}

func (m *MaxItemsV) ValidateSlice(v reflect.Value) error {
	if v.Len() > m.l {
		return fmt.Errorf(ERROR_MAX_LEN_ARR, m.l)
	}
	return nil
}
