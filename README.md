# jsonv
Golang validating JSON parser

Parses JSON whilst validating it. The created parsers generate useful error messages
with full paths to the JSON values that triggered the parse error.

As many errors as possible are accumulated before returning, allowing APIs to provide
more complete feedback rather than only propagating 1 error at a time.

# Example use

```golang
import "github.com/Satook/jsonv"

type Customer struct {
	Id          int64
	Name        string
	Email       string
	Phone       string

	PublicId     string
	ReminderSent bool
}

// Example phone number parser/validator
var PhoneNumber = jsonv.String(
	jsonv.Pattern(`^+?[ 0-9]{6,}$`, "Please enter a valid Phone number, it can start with '+'."),
)

// Example email parser/validator
var EmailAddress = jsonv.String(
	jsonv.Pattern(`[@][a-zA-Z0-9-_.]+$`, "Please enter an email address, i.e. contain an '@' followed by a domain."),
)

var CreateParser = jsonv.Parser(&Customer{},
	jsonv.Struct(
		jsonv.Prop("Name", jsonv.String()),
		jsonv.PropWithDefault("Email", EmailAddress, ""),
		jsonv.PropWithDefault("Phone", PhoneNumber, ""),
		jsonv.PropWithDefault("ReminderSent", jsonv.Boolean(), false),
  )
)
```

The `CreateParser` can parse JSON and validate all properties during parsing. If a property
isn't included in the parser definition, it will be left with its default value and never
altered by the parser.

To parse JSON from an `io.Reader` named `data`:

```golang
var cust Customer

if err := CreateParser.Parse(data, &cust); err != nil {
  if verr, ok := err.(jsonv.ValidationError); ok {
    // we should do someting with this error, maybe propagate it back to the client?
  } else {
    // you should never get here, but it's probably worth a
    panic(err)
  }
}
```

If `.Parse` returns nil, `cust` will be filled in with valid data.

# New types

In order to add new types that jsonv can understand, implement the `SchemaType` interface.
For examples, check out the types_*.go files.
