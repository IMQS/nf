package nf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// RunProtected runs 'func' inside a panic handler that recognizes our special errors,
// and sends the appropriate HTTP response.
func RunProtected(w http.ResponseWriter, r *http.Request, p httprouter.Params, handler func(w http.ResponseWriter, r *http.Request, p httprouter.Params)) {
	defer func() {
		if r := recover(); r != nil {
			if hErr, ok := r.(HTTPError); ok {
				http.Error(w, hErr.Message, hErr.Code)
			} else if err, ok := r.(error); ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else if err, ok := r.(string); ok {
				http.Error(w, err, http.StatusInternalServerError)
			} else {
				http.Error(w, "Unrecognized panic", http.StatusInternalServerError)
			}
		}
	}()

	handler(w, r, p)
}

// Handle adds a protected HTTP route to router (ie handle will run inside RunProtected)
func Handle(router *httprouter.Router, method, path string, handle httprouter.Handle) {
	wrapper := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		RunProtected(w, r, p, handle)
	}
	router.Handle(method, path, wrapper)
}

func ParseID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func SendJson(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(obj)
	Check(err)
	w.Write(b)
}

func SendID(w http.ResponseWriter, id interface{}) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%v", id)
}

func SendOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}
