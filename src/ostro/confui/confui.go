package confui

import (
	"fmt"
	"log"
	"os"
	"strings"
	"net/http"
)

const (
	htmlTemplate= `<!DOCTYPE html>
<html>
  <head>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <script type="text/javascript" src="/confs/infra/toolgen.js"></script>
    <script type="text/javascript" src="/confs/infra/%sgen.js"></script>
    <script type="text/javascript" src="/confs%s.js"></script>

    <link rel="stylesheet" type="text/css" href="/confs/infra/page.css">
  </head>
  <body>
  </body>
</html>
`
)

var (
	uiServer *Server = nil
)

type Server struct {
	addr string
	port int
	pattern string
	prefix string
}

func NewServer(addr string, port int, pattern, prefix string) error {
	if uiServer == nil {
		uiServer = &Server{
			addr: addr,
			port: port,
			pattern: strings.TrimRight(pattern, "/"),
			prefix: prefix}

		mux := http.NewServeMux()
		mux.HandleFunc(pattern, httpRequestHandler)

		srv := &http.Server{
			Addr: fmt.Sprintf("%s:%d", addr, port),
			Handler: mux,
			MaxHeaderBytes: 4096}

		go func(s *http.Server) {
			log.Print(s.ListenAndServe())
		}(srv)
	} else {
		if uiServer.addr != addr || uiServer.port != port || uiServer.pattern != pattern || uiServer.prefix != prefix {
			return fmt.Errorf("attempt to create multiple confui.Servers")
		}
	}

	return nil
}

func httpRequestHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimRight(strings.TrimPrefix(r.URL.Path, uiServer.pattern), "/")
	log.Printf("**** http: '%s'\n", r.URL.Path)

	if strings.HasPrefix(relPath, "/infra") || strings.HasSuffix(relPath, ".js") {
		filePath := fmt.Sprintf("%s%s", uiServer.prefix, relPath)
		log.Printf("     serving file '%s'\n", filePath)
		http.ServeFile(w, r, filePath)
	} else {
		generateHtmlResponse(w, r, relPath)
	}
}

func generateHtmlResponse(w http.ResponseWriter, r *http.Request, relPath string) {
	var filePath, gen, content string
		
	if relPath == "" {
		relPath = "/root"
		filePath = fmt.Sprintf("%s%s", uiServer.prefix, relPath)
		gen = "dir"
	} else {
		filePath = fmt.Sprintf("%s%s", uiServer.prefix, relPath)
		gen = "form"
		log.Printf("     checking '%s' ...\n", filePath)
		if info, err := os.Stat(filePath); err == nil {
			if info.IsDir() {
				gen = "dir"
			}
		}
	}

	log.Printf("     generate '%s'\n", gen)

	jsFile := fmt.Sprintf("%s.js", filePath)
	if _, err := os.Stat(jsFile); err != nil {
		log.Printf("can't find '%s' file", jsFile);
		http.NotFound(w, r)
		return
	}

	content = fmt.Sprintf(htmlTemplate, gen, relPath)

	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Write([]byte(content))
}
