/**
Defines validators for the various JSON-Schema types.

These implement the base validation functionality, allowing values to be
validated as they're parsed.

The implementation are per JSON primative type, to allow type-safe (and simpler)
code.

See the relevant validators_{JSONTYPE}.go file for the implementation of each
validators logic for a given type.
*/
package jsonschema

/*
The Min Length validator.

Can work with:
 - Strings
 - Arrays (golang slices)
*/
type MinLenV struct {
	l int
}

func MinLen(l int) *MinLenV {
	return &MinLenV{l}
}

type MaxLenV struct {
	l int
}

func MaxLen(l int) *MaxLenV {
	return &MaxLenV{l}
}
