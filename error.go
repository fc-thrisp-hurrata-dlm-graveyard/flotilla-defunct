package flotilla

type (
	Error string
)

func newError(message string) (e error) {
	return Error(message)
}

func (e Error) Error() string {
	return string(e)
}
