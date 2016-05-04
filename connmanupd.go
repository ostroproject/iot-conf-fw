package main

import (
	"flag"
	"strings"
	"ostro/connman"
	"ostro/connmanupd"
)



func main() {
	var inputFile, configDir, logLevel string

	flag.StringVar(&inputFile, "input-file", "/tmp/connman.conf",
		"path of the JSON input file")
	flag.StringVar(&configDir, "config-dir", "/var/lib/connman",
		"directory where Connman looks for its configuration")
	flag.StringVar(&logLevel, "log-level", "error",
		"one of 'fatal', 'error', 'info', 'debug' or 'all'")

	flag.Parse()

	setLogLevel(logLevel)
		
	conf, cerr := connmanupd.NewConf(inputFile)
	if cerr != nil {
		connman.Printf(connman.LogFatal, "Config file error: %v\n", cerr)
	}

	ierrs := conf.Install(configDir)
	fatal := false
	for component, failure := range ierrs {
		level := connman.LogError
		if strings.HasPrefix(failure.Error(), connmanupd.NoService) {
			level = connman.LogDebug
		} else {
			fatal = true
		}
		connman.Printf(level, "Installation of %s configuration failed: %v\n", component, failure);
	}

	if fatal {
		connman.Printf(connman.LogFatal, "installation of configuration failed\n")
	}
}


func setLogLevel(level string) {
	switch level {
	case "all":
		connman.LogLevel = connman.LogAll
	case "debug":
		connman.LogLevel = connman.LogDebug
	case "info":
		connman.LogLevel = connman.LogInfo
	case "error":
		connman.LogLevel = connman.LogError
	case "fatal":
		connman.LogLevel = connman.LogFatal
	default:
		connman.Printf(connman.LogFatal, "invalid log level '%s'\n", level)
	}
}
