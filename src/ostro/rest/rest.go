package rest

import (
	"fmt"
	"log"
	"os"
	"io"
	"strings"
	"net/url"
	"net/http"
	"path/filepath"
	"encoding/json"
	"ostro/confs"
)


type FileHandler struct {
	addr string
	port int
	pattern string
	prefix string
	tmpdir string
}

const (
	HashAttr string = "user.hash"
	OriginAttr string = "user.origin"
	
	RestOriginated = "Rest"
)


var (
	fileHandler *FileHandler = nil

	allowedMethods = map[string]bool{
		"options":true,
		"get": true,
		"put": true,
		"patch": true,
		"delete": true}
)


func NewFileHandler(addr string, port int, pattern, prefix, tmpdir, certFile, keyFile string) error {
	if fileHandler == nil {
		log.Printf("checking '%s'\n", tmpdir)
		if err := os.MkdirAll(filepath.Dir(tmpdir), 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %v", tmpdir, err)
		}

		fileHandler = &FileHandler{
			addr: addr,
			port: port,
			pattern: strings.TrimRight(pattern, "/"),
			prefix: prefix,
			tmpdir: tmpdir}

		mux := http.NewServeMux()
		mux.HandleFunc(pattern, httpRequestHandler)

		srv := &http.Server{
			Addr: fmt.Sprintf("%s:%d", addr, port),
			Handler: mux,
			MaxHeaderBytes: 4096}
		srv.SetKeepAlivesEnabled(false)

		go func(s *http.Server) {
      if certFile == "" || keyFile == "" {
        log.Print(s.ListenAndServe())
      } else {
        log.Print(s.ListenAndServeTLS(certFile, keyFile))
      }
		}(srv)
	} else {
		pat := strings.TrimRight(pattern, "/")
		if fileHandler.addr != addr || fileHandler.port != port || fileHandler.pattern != pat || fileHandler.prefix != prefix {
			return fmt.Errorf("attempt to create multiple rest.FileHandlers")
		}
	}

	return nil
}

func httpRequestHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) > 0 {
		http.Error(w, "queries are not allowed", http.StatusBadRequest)
		return
	}
	
	rp := fmt.Sprintf("%s%s", fileHandler.prefix, strings.TrimRight(strings.TrimPrefix(r.URL.Path, fileHandler.pattern), "/"))
	
	log.Printf("**** rest %s %s\n", r.Method, rp)

	setAccessControlFields(w, r)

	switch r.Method {
	case "OPTIONS":
		handleOptions(rp, w, r)
	case "GET":
		getValues(rp, w, r)
	case "PUT":
		setValues(rp, w, r)
	case "PATCH":
		http.Error(w, "OK", http.StatusOK)
	case "DELETE":
		http.Error(w, "OK", http.StatusOK)
	default:
		http.Error(w, fmt.Sprintf("'%s' method not supported", r.Method), http.StatusMethodNotAllowed)
	}
}

func handleOptions(rp string, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "OK", http.StatusOK)
}

func getValues(rp string, w http.ResponseWriter, r *http.Request) {
	var (
		err error
		reply []byte
	)

	values := make(map[string]interface{})

	for _, tree := range []string{"factory", "common", "local"} {
		treeValues := make(map[string]interface{})
		path := strings.Replace(rp, "local", tree, 1)
		
		if err = traverseDirectoryTree(path, treeValues); err != nil {
			if !os.IsNotExist(err) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		log.Printf("     values in '%s': %v\n", path, treeValues)

		for _, root := range treeValues {
			confs.MergeFragment(root.(map[string]interface{}), values)
			break
		}
	}

	if reply, err = json.MarshalIndent(values, "", "    "); err != nil {
		http.Error(w, fmt.Sprintf("failed to produce JSON: %v", err),
			http.StatusInternalServerError);
		return
	}
	
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(reply)))

	w.Write(reply)
}

func setValues(rp string, w http.ResponseWriter, r *http.Request) {
	var (
		newValues, values map[string]interface{}
		err error
	)
	
	if r.ContentLength > 65536 {
		http.Error(w, "Excessive request content", http.StatusBadRequest)
		return
	}

	size := int(r.ContentLength)

	if size < 2 {
		http.Error(w, "Undersized request content", http.StatusBadRequest)
		return
	}
	
	content := make([]byte, size)
	if len, err := r.Body.Read(content); (err != nil && err != io.EOF) || len != size {
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request content: %v", err),
				http.StatusInternalServerError)
		} else {
			http.Error(w, "Failed to read request content: partial read",
				http.StatusInternalServerError)
		}
		return
	}

	newValues = make(map[string]interface{})
	if err = json.Unmarshal(content, &newValues); err != nil {
		http.Error(w, fmt.Sprintf("malformed request content: %v", err),
			http.StatusBadRequest)
		return
	}

	if values, err = readFile(rp); err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("error during read '%s': %v", rp, err),
				http.StatusInternalServerError)
			return
		}
		values = make(map[string]interface{})
	}

	if err = confs.MergeFragment(newValues, values); err != nil {
		http.Error(w, fmt.Sprintf("fragment merging failed: %v", err),
			http.StatusInternalServerError)
		return
	}

	if err = writeFile(rp, values); err != nil {
		http.Error(w, fmt.Sprintf("writing to '%s' failed: %v", rp, err),
			http.StatusInternalServerError)
		return
	}
	
	http.Error(w, "OK", http.StatusOK)
}

func allowOrigin(host, origin string) bool {
	return true
}

func setAllowOrigin(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	origin := r.Header.Get("Origin")
	
	if origURL, err := url.Parse(origin); err == nil {
		origHost := origURL.Host

		if host != "" && origHost != "" && allowOrigin(host, origHost) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}
}

func setAllowMethods(w http.ResponseWriter, r *http.Request) {
	request := strings.Replace(r.Header.Get("Access-Control-Request-Method"), " ", "", -1)
	allow := []string{}

	if request != "" {
		for _, method := range strings.Split(request, ",") {
			if allowedMethods[strings.ToLower(method)] {
				allow = append(allow, method)
			}
		}
		if len(allow) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allow, ", "))
		}
	}
}
	
func setAllowHeaders(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("Access-Control-Request-Headers")
	if request != "" {
		w.Header().Set("Access-Control-Allow-Headers", request)
	}
}

func setAccessControlFields(w http.ResponseWriter, r *http.Request) {
	setAllowOrigin(w,r)
	setAllowMethods(w,r)
	setAllowHeaders(w,r)
}
