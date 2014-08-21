package flotilla

import "fmt"

type (
	FlotillaError struct {
		format     string
		parameters []interface{}
	}
)

func newError(format string, parameters ...interface{}) error {
	return &FlotillaError{
		format:     format,
		parameters: parameters,
	}
}

func (e *FlotillaError) Error() string {
	return fmt.Sprintf(e.format, e.parameters...)
}
