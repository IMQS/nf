package nf

import (
	"net/http"
)

// HTTPError is an object that can be panic'ed, and the outer HTTP handler function
// will return the appropriate HTTP error message
type HTTPError struct {
	Code    int
	Message string
}

// Panic creates an HTTPError object and panics it
func Panic(code int, message string) {
	panic(HTTPError{code, message})
}

// PanicForbidden panics with a 400 Forbidden
func PanicForbidden() {
	panic(HTTPError{http.StatusForbidden, "Forbidden"})
}

// Check causes a panic if err is not nil
func Check(err error) {
	if err != nil {
		panic(err)
	}
}
