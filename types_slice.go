package jsonv

import (
	"fmt"
	"reflect"
)

/*
Parses a JSON value into an array whos values are a single type.
*/
type SliceParser struct {
	elemType reflect.Type
	schema   SchemaType
	vs       []SliceValidator
}

func Slice(s SchemaType, vs ...SliceValidator) *SliceParser {
	return &SliceParser{schema: s, vs: vs}
}

func (p *SliceParser) Prepare(t reflect.Type) error {
	// make sure it's a struct
	if t.Kind() != reflect.Slice {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, t)
	}

	p.elemType = t.Elem()

	// prepare our sub-type if we need to
	if ps, ok := p.schema.(PreparedSchemaType); ok {
		return ps.Prepare(p.elemType)
	}

	return nil
}

func (p *SliceParser) Parse(path Pather, s *Scanner, v interface{}) error {
	// check we have a ptr to a struct
	ptrVal := reflect.ValueOf(v)
	ptrType := ptrVal.Type()
	if ptrType.Kind() != reflect.Ptr || ptrVal.IsNil() {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, ptrVal.Type())
	}
	val := ptrVal.Elem()
	valType := val.Type()
	if valType.Kind() != reflect.Slice {
		return fmt.Errorf(ERROR_BAD_SLICE_DEST, ptrVal.Type())
	}

	// read the '['
	tok, _, err := s.ReadToken()
	if tok == TokenError {
		return err
	} else if tok != TokenArrayBegin {
		return NewParseError("Expected '[' not " + tok.String())
	}

	finished := false

	// see if we have at least 1 value
	if tok, err := s.PeekToken(); err != nil {
		return err
	} else if tok == TokenArrayEnd {
		// actually consume it
		s.ReadToken()
		finished = true
	}

	// this is where we'll store all the validation errors
	var errs ValidationError

	// now read val then ','|']'
	i := 0
	itemPath := func() string {
		return fmt.Sprintf("%s%d/", path(), i)
	}
	for !finished {
		// next up must be a value
		// Grow the slice if necessary
		if i >= val.Cap() {
			newcap := val.Cap() + val.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(val.Type(), val.Len(), newcap)
			reflect.Copy(newv, val)
			val.Set(newv)
		}
		if i >= val.Len() {
			val.SetLen(i + 1)
		}

		// read in the value
		itemPtr := val.Index(i).Addr().Interface()
		if err := p.schema.Parse(itemPath, s, itemPtr); err != nil {
			if verr, ok := err.(ValidationError); ok {
				errs = errs.AddMany(verr)
			} else {
				return err
			}
		}

		i++

		// we want either a ',' or a ']'
		if tok, _, err := s.ReadToken(); tok == TokenError {
			return err
		} else if tok == TokenArrayEnd {
			finished = true
		} else if tok == TokenItemSep {
			continue
		} else {
			return NewParseError("Expected ',' or '[' not " + tok.String())
		}
	}

	// validate the contents
	for _, v := range p.vs {
		if err := v.ValidateSlice(val); err != nil {
			errs = errs.Add(path(), err.Error())
		}
	}
	if len(errs) > 0 {
		return errs
	} else {
		return nil
	}
}
