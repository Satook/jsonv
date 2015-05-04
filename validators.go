/**
Defines validators for the various JSON-Schema types.

These implement the base validation functionality, allowing values to be
validated as they're parsed.

The implementation are per JSON primative type, to allow type-safe (and simpler)
code.

See the relevant validators_{JSONTYPE}.go file for the implementation of each
validators logic for a given type.

Note: Number validators are strongly type to either Integer or Float types.
*/
package jsonschema

import (
	"fmt"
)

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
	if l < 0 {
		panic(fmt.Errorf("Minimum allowed length must be >= 0"))
	}
	return &MinLenV{l}
}

/*
The Max Length validator.

Can work with:
 - Strings
 - Arrays (golang slices)
*/
type MaxLenV struct {
	l int
}

func MaxLen(l int) *MaxLenV {
	if l < 0 {
		panic(fmt.Errorf("Maximum allowed length must be >= 0"))
	}
	return &MaxLenV{l}
}
