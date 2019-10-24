package nf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IMQS/serviceauth"
	"github.com/IMQS/serviceauth/permissions"
	"github.com/julienschmidt/httprouter"
)

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
	wrapper := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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
	router.Handle(method, path, wrapper)
}

type httpFallback struct {
	indexFile string
}

func (h *httpFallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("fallback: %v\n", r.RequestURI)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.Error(w, "Not a valid API", 404)
		return
	}
	http.ServeFile(w, r, h.indexFile)
}

func pathExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

// HandleStaticFiles creates a catch-all handler that serves up static files if they exist, or returns /index.html if the path does not exist.
// The function first tries to see if www/dist exists, relative to the current directory, and if this does exist, then it uses that.
// If www/dist does not exist, and we're running in a container, then the function looks for /var/www, and if that exists, then it uses that.
// If neither of these options succeed, then the function panics
func HandleStaticFiles(router *httprouter.Router) {
	pwd, _ := os.Getwd()
	wwwFilesRoot := ""
	{
		relative := filepath.Join(pwd, "www", "dist")
		container := "/var/www"
		if pathExists(relative) {
			// This is when running in dev mode, and you type "go run main.go"
			wwwFilesRoot = relative
		} else if IsRunningInContainer() && pathExists(container) {
			wwwFilesRoot = container
		} else {
			panic(fmt.Sprintf("Unable to find a 'www' root directory in %v or %v", relative, container))
		}
	}

	staticFilesStrip := http.StripPrefix("/static", http.FileServer(http.Dir(wwwFilesRoot)))
	staticFiles := http.FileServer(http.Dir(wwwFilesRoot))
	router.Handle("GET", "/static/*path", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		//fmt.Printf("GET /static: %v\n", r.RequestURI)
		staticFilesStrip.ServeHTTP(w, r)
	})
	router.Handle("GET", "/robots.txt", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		staticFiles.ServeHTTP(w, r)
	})
	router.Handle("GET", "/favicon.ico", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		staticFiles.ServeHTTP(w, r)
	})
	router.NotFound = &httpFallback{
		indexFile: filepath.Join(wwwFilesRoot, "index.html"),
	}
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

// SendPong sends a reply to an HTTP ping request, which checks if the service is alive.
func SendPong(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=0, no-cache")
	fmt.Fprintf(w, `{"Timestamp": %v}`, time.Now().Unix())
}
