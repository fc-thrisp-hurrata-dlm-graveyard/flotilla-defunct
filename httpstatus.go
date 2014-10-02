package flotilla

type (
	// A map of HttpStatus instances, keyed by status code
	HttpStatuses map[int][]HandlerFunc
)
