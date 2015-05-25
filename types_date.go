package jsonv

import (
	"fmt"
	"reflect"
	"time"
)

const date_fmt = `"2006-01-02"`

var dateType = reflect.TypeOf(time.Now())

/*
Validator type for Dates
*/
type DateValidator interface {
	ValidateDate(time.Time) error
}

/*
Parses JSON strings value and stores it in a Go time.Time.

The string must be in the format "yyyy-mm-dd"
*/
type DateParser struct {
	vs []DateValidator
}

func Date(vs ...DateValidator) *DateParser {
	return &DateParser{vs}
}

func (p *DateParser) Prepare(t reflect.Type) error {
	if t != dateType {
		return fmt.Errorf("Want time.Time not %v", t)
	}

	return nil
}

func (p *DateParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewParseError(fmt.Sprintf(ERROR_INVALID_DATE, string(buf)))
	}

	if dest, ok := v.(*time.Time); !ok {
		return fmt.Errorf(ERROR_BAD_DATE_DEST, reflect.TypeOf(v), path)
	} else {
		var errs ValidationError

		val, err := time.Parse(date_fmt, string(buf))
		if err != nil {
			errs = errs.Add(path, err.Error())
			return errs
		}

		// validate the value
		for _, v := range p.vs {
			if err := v.ValidateDate(val); err != nil {
				errs = errs.Add(path, err.Error())
			}
		}
		if len(errs) > 0 {
			return errs
		}

		*dest = val
	}

	return nil
}
