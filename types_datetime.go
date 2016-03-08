package jsonv

import (
	"fmt"
	"reflect"
	"time"
)

const datetime_fmt = `"2006-01-02 15:04:05"`

var dateTimeType = reflect.TypeOf(time.Now())

/*
Validator type for DateTimes
*/
type DateTimeValidator interface {
	ValidateDateTime(time.Time) error
}

type DateTimeValidatorFunc func(time.Time) error

func (f DateTimeValidatorFunc) ValidateDateTime(t time.Time) error {
	return f(t)
}

/*
Parses JSON strings value and stores it in a Go time.Time.

The string must be in the format `"2016-03-10T23:00:00.000Z"`
*/
type DateTimeParser struct {
	vs []DateTimeValidator
}

func DateTime(vs ...DateTimeValidator) *DateTimeParser {
	return &DateTimeParser{vs}
}

func (p *DateTimeParser) Prepare(t reflect.Type) error {
	if t != dateTimeType {
		return fmt.Errorf("Want time.Time not %v", t)
	}

	return nil
}

func (p *DateTimeParser) Parse(path Pather, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == TokenError {
		return err
	} else if tok != TokenString {
		return NewParseError(ERROR_INVALID_DATE_TIME, string(buf))
	}

	if dest, ok := v.(*time.Time); !ok {
		return NewParseError(ERROR_BAD_DATE_TIME_DEST, reflect.TypeOf(v), path())
	} else {
		var errs ValidationError

		val, err := time.Parse(datetime_fmt, string(buf))
		if err != nil {
			errs = errs.Add(path(), err.Error())
			return errs
		}

		// validate the value
		for _, v := range p.vs {
			if err := v.ValidateDateTime(val); err != nil {
				errs = errs.Add(path(), err.Error())
			}
		}
		if len(errs) > 0 {
			return errs
		}

		*dest = val
	}

	return nil
}
