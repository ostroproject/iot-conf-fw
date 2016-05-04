package main

import (
	"log"
	"flag"
	"strings"
	"ostro/rest"
	"ostro/confui"
)


func main() {
	var httpPrefixRaw, restPrefixRaw, dropZoneRootRaw, uiRootRaw string
	var httpPort, restPort int;

	flag.IntVar(&restPort, "rest-port", 4984, "REST server port")
	flag.StringVar(&restPrefixRaw, "rest-prefix", "/confs/local", "REST resource prefix")
	flag.StringVar(&dropZoneRootRaw, "drop-zone", "/var/cache/confs", "root of the Drop Zone")

	flag.IntVar(&httpPort, "http-port", 8080, "HTTP server port")
	flag.StringVar(&httpPrefixRaw, "http-prefix", "/confs", "UI URL prefix")
	flag.StringVar(&uiRootRaw, "ui-files", "/usr/share/confs/ui", "root directory of the UI files")

	flag.Parse()

	if restPort < 1 || restPort > 65535 {
		log.Fatalf("invalid rest-port %d: out of range 1 - 65535", restPort)
	}
	restPrefix := strings.TrimRight(restPrefixRaw, "/")
	dropZoneRoot := strings.TrimRight(dropZoneRootRaw, "/")

	
	if httpPort < 1 || httpPort > 65535 {
		log.Fatalf("invalid http-port %d: out of range 1 - 65535", httpPort)
	}
	httpPrefix := strings.TrimRight(httpPrefixRaw, "/")
	uiRoot := strings.TrimRight(uiRootRaw, "/")
	
	rest.NewFileHandler("", restPort, restPrefix + "/", dropZoneRoot + "/local", dropZoneRoot + "/tmp")
	confui.NewServer("", httpPort, httpPrefix + "/", uiRoot)
	
	select {}
}
