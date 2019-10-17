package nf

import (
	"fmt"
	"net/http"
)

// HTTPError is an object that can be panic'ed, and the outer HTTP handler function.
// Will return the appropriate HTTP error message.
type HTTPError struct {
	Code    int
	Message string
}

// Panic creates an HTTPError object and panics it.
func Panic(code int, message string) {
	panic(HTTPError{code, message})
}

// PanicBadRequest panics with a 400 Bad Request.
func PanicBadRequest() {
	panic(HTTPError{http.StatusBadRequest, "Bad Request"})
}

// PanicBadRequestf panics with a 400 Bad Request.
func PanicBadRequestf(format string, args ...interface{}) {
	panic(HTTPError{http.StatusBadRequest, fmt.Sprintf(format, args...)})
}

// PanicForbidden panics with a 403 Forbidden.
func PanicForbidden() {
	panic(HTTPError{http.StatusForbidden, "Forbidden"})
}

// PanicNotFound panics with a 404 Not Found.
func PanicNotFound() {
	panic(HTTPError{http.StatusNotFound, "Not Found"})
}

// PanicServerError panics with a 500 Internal Server Error
func PanicServerError(msg string) {
	panic(HTTPError{http.StatusInternalServerError, msg})
}

// PanicServerErrorf panics with a 500 Internal Server Error
func PanicServerErrorf(format string, args ...interface{}) {
	panic(HTTPError{http.StatusInternalServerError, fmt.Sprintf(format, args...)})
}

// Check causes a panic if err is not nil.
func Check(err error) {
	if err != nil {
		panic(err)
	}
}
