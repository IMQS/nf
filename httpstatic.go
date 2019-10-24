package nf

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type httpFallback struct {
	publicPath string // eg /facilities, or /leasing (prefix of URL)
	indexFile  string // eg /var/www/index.html (absolute path on disk)
}

func (h *httpFallback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("fallback: %v\n", r.RequestURI)
	if h.publicPath != "" && !strings.HasPrefix(r.URL.Path, h.publicPath) {
		http.Error(w, "Invalid router config. All URLs to this service should being with "+h.publicPath, 404)
		return
	}
	if strings.HasPrefix(r.URL.Path, h.publicPath+"/api/") {
		// This is helpful for developers, instead of just getting back index.html
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
// publicPath is something like '/facilities', '/leasing', or whatever your root path is in the IMQS router.
// publicPath may also be empty, if this service runs alone.
// The function first tries to see if www/dist exists, relative to the current directory, and if this does exist, then it uses that.
// If www/dist does not exist, and we're running in a container, then the function looks for /var/www, and if that exists, then it uses that.
// If neither of these options succeed, then the function panics
func HandleStaticFiles(router *httprouter.Router, publicPath string) {
	if publicPath != "" && publicPath[0] != '/' {
		panic("publicPath must either be empty, or start with a slash")
	}
	if publicPath != "" && publicPath[len(publicPath)-1] == '/' {
		// remove trailing slash
		publicPath = publicPath[:len(publicPath)-1]
	}
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

	publicStatic := publicPath + "/static"

	// This strips "/facilities/static"
	staticFilesStrip := http.StripPrefix(publicStatic, http.FileServer(http.Dir(wwwFilesRoot)))

	router.Handle("GET", publicStatic+"/*path", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		//fmt.Printf("GET %v: %v\n", publicStatic, r.RequestURI)
		staticFilesStrip.ServeHTTP(w, r)
	})

	// I suspect these URLs will never be hit, but it seems prudent to leave them in
	var staticFiles http.Handler
	if publicPath == "" {
		staticFiles = http.FileServer(http.Dir(wwwFilesRoot))
	} else {
		// This strips "/facilities"
		staticFiles = http.StripPrefix(publicPath, http.FileServer(http.Dir(wwwFilesRoot)))
	}
	router.Handle("GET", publicPath+"/robots.txt", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		staticFiles.ServeHTTP(w, r)
	})
	router.Handle("GET", publicPath+"/favicon.ico", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		staticFiles.ServeHTTP(w, r)
	})

	// Everything else returns index.html
	router.NotFound = &httpFallback{
		publicPath: publicPath,
		indexFile:  filepath.Join(wwwFilesRoot, "index.html"),
	}
}
