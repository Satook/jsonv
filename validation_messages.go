package jsonschema

const (
	ERROR_INVALID_INT  = "Must be an integer, got value %v"
	ERROR_PARSE_INT    = "Error parsing integer, %v"
	ERROR_BAD_INT_DEST = "Cannot assign integer to target variable of type %v"
	ERROR_INVALID_BOOL = "Must be an boolean, got value %v"
	ERROR_PARSE_BOOL   = "Error parsing bool, %v"
	ERROR_MIN_LEN_STR  = "Must be at least %d characters long"
	ERROR_MIN_LEN_ARR  = "Must contain at least %d items"
	ERROR_MAX_LEN_STR  = "Must be no more than %d characters long"
	ERROR_MAX_LEN_ARR  = "Must contain no more than %d items"
)
