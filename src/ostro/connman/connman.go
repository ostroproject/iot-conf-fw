package connman

import (
	"fmt"
	"log"
	"github.com/godbus/dbus"
)

type Server struct {
	conn *dbus.Conn
	address string
}

const (
	LogAll int = iota
	LogDebug
	LogInfo
	LogError
	LogFatal
)

var (
	LogLevel int = LogError
	DisabledTechnologies []string = []string{}
	srv *Server = nil
)

func Printf(logLevel int, fmt string, args ...interface{}) {
	if logLevel >= LogLevel {
		if logLevel == LogFatal {
			log.Fatalf(fmt, args...)
		} else {
			log.Printf(fmt, args...)
		}
	}
}

func AddDisabledTechnology(typ string) {
	for _, t := range DisabledTechnologies {
		if t == typ {
			return
		}
	}
	
	DisabledTechnologies = append(DisabledTechnologies, typ)
}

func NewServer() (*Server, error) {
	if srv == nil {
		if conn, err := dbus.SystemBus(); err != nil {
			return nil, fmt.Errorf("Failed to get D-Bus System Bus: %v\n", err)
		} else {
			srv = &Server{
				conn: conn,
				address: "net.connman"}
		}
	}

	return srv, nil
}

