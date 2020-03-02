package nf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/IMQS/serviceauth"
	"github.com/IMQS/serviceauth/permissions"
	"github.com/julienschmidt/httprouter"
)

// BypassAuth changes the `HandleAuthenticated` function to affectively be the
// `Handle` function. The value of the auth token becomes `nil`.
var BypassAuth bool = false

// AuthenticatedHandler is an HTTP handler function that has already had authentication information read from the auth service.
type AuthenticatedHandler func(w http.ResponseWriter, r *http.Request, p httprouter.Params, auth *serviceauth.Token)

// RunProtected runs 'func' inside a panic handler that recognizes our special errors,
// and sends the appropriate HTTP response if a panic does occur.
func RunProtected(w http.ResponseWriter, handler func()) {
	defer func() {
		if rec := recover(); rec != nil {
			if hErr, ok := rec.(HTTPError); ok {
				http.Error(w, hErr.Message, hErr.Code)
			} else if err, ok := rec.(error); ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else if err, ok := rec.(string); ok {
				http.Error(w, err, http.StatusInternalServerError)
			} else {
				http.Error(w, "Unrecognized panic", http.StatusInternalServerError)
			}
		}
	}()

	handler()
}

// Handle adds a protected HTTP route to router (ie handle will run inside RunProtected, so you get a panic handler).
func Handle(router *httprouter.Router, method, path string, handle httprouter.Handle) {
	wrapper := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		RunProtected(w, func() { handle(w, r, p) })
	}
	router.Handle(method, path, wrapper)
}

// HandleAuthenticated adds a protected HTTP route to router (ie handle will run inside RunProtected, so you get a panic handler).
// Any permission specified in needPermissions must be present in the authentication information, otherwise the handler
// will not call your 'handle' function, but will return with 403 Forbidden.
// In addition, the authentication token must have the 'enabled' permission set, otherwise a 403 Forbidden is returned, with
// the response body "User Disabled".
func HandleAuthenticated(router *httprouter.Router, method, path string, handle AuthenticatedHandler, needPermissions []int) {
	var wrapper func(w http.ResponseWriter, r *http.Request, p httprouter.Params)
	if BypassAuth {
		wrapper = func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			RunProtected(w, func() { handle(w, r, p, nil) })
		}
	} else {
		wrapper = func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			authCode, authMsg, authToken := serviceauth.GetToken(r)
			if authCode != http.StatusOK {
				http.Error(w, authMsg, authCode)
				return
			}
			if !authToken.HasPermByID(permissions.PermEnabled) {
				http.Error(w, "User Disabled", http.StatusForbidden)
				return
			}
			RunProtected(w, func() { handle(w, r, p, authToken) })
		}
	}
	router.Handle(method, path, wrapper)
}

// ParseID parses a 64-bit integer, and returns zero on failure.
func ParseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

// ReadJSON reads the body of the request, and unmarshals it into 'obj'.
func ReadJSON(r *http.Request, obj interface{}) {
	if r.Body == nil {
		Panic(http.StatusBadRequest, "ReadJSON failed: Request body is empty")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(obj)
	if err != nil {
		Panic(http.StatusBadRequest, "ReadJSON failed: Failed to decode JSON - "+err.Error())
	}
}

// SendJSON encodes 'obj' to JSON, and sends it as an HTTP application/json response.
func SendJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(obj)
	Check(err)
	w.Write(b)
}

// SendText converts text (of type string) into a the array and sends it as an
// HTTP text/plain response
func SendText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain")
	b := []byte(text)
	w.Write(b)
}

// SendBytes sends text as an HTTP text/plain response
func SendBytes(w http.ResponseWriter, bytes []byte) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write(bytes)
}

// SendYML encodes 'obj' to a byte array, and sends it as an HTTP text/yaml response.
func SendYML(w http.ResponseWriter, yml []byte) {
	w.Header().Set("Content-Type", "text/yml")
	w.Write(yml)
}

// SendID encodes 'id' as a string, and sends it as an HTTP text/plain response.
func SendID(w http.ResponseWriter, id interface{}) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%v", id)
}

// SendOK sends "OK" as a text/plain response.
func SendOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

// SendPong sends a reply to an HTTP ping request, which checks if the service
// is alive.
func SendPong(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=0, no-cache")
	fmt.Fprintf(w, `{"Timestamp": %v}`, time.Now().Unix())
}
