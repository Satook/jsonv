package jsonschema

/*
Used to identify validators that can work on Integer values.
*/
type IntegerValidator interface {
	ValidateInteger(i int64) error
}

type FloatValidator interface {
	ValidateInteger(f float64) error
}
