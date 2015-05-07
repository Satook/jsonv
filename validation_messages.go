package jsonv

const (
	// messages for bad destination types
	ERROR_BAD_INT_DEST    = "Cannot assign integer to target variable of type %v, path %v"
	ERROR_BAD_FLOAT_DEST  = "Cannot assign float to target variable of type %v, path %v"
	ERROR_BAD_STRING_DEST = "Cannot assign string to target variable of type %v, path %v"
	ERROR_BAD_BOOL_DEST   = "Cannot assign boolean to target variable of type %v, path %v"
	ERROR_BAD_OBJ_DEST    = "Must be a non-nil ptr to a struct, not %v"
	ERROR_BAD_SLICE_DEST  = "Must be a non-nil ptr to a slice, not %v"

	ERROR_INVALID_INT = "Expected an integer, got %v"
	ERROR_PARSE_INT   = "Error parsing integer, %v"

	ERROR_INVALID_BOOL = "Expected a boolean, got %v"
	ERROR_PARSE_BOOL   = "Error parsing bool, %v"

	ERROR_MIN_LEN_STR   = "Must be at least %d characters long"
	ERROR_MAX_LEN_STR   = "Must be no more than %d characters long"
	ERROR_PATTERN_MATCH = "Must match regex pattern %v"

	ERROR_MIN_LEN_ARR = "Must contain at least %d items"
	ERROR_MAX_LEN_ARR = "Must contain no more than %d items"

	// general number validation errros
	ERROR_MAX_EX = "Must be less than %v"
	ERROR_MAX    = "Must be less than or equal to %v"
	ERROR_MIN_EX = "Must be greater than %v"
	ERROR_MIN    = "Must be greater than or equal to %v"
	ERROR_MULOF  = "Must be a multiple of %v"
)
