package confs

import (
	"fmt"
	"log"
	"strings"
)

func printf(level uint, format string, args ...interface{}) {
	if level <= LogLevel {
		log.Printf("%s\n", strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
	}
}

func Errorf(format string, args ...interface{}) {
	printf(LogError, format, args...)
}

func Infof(format string, args ...interface{}) {
	printf(LogInfo, format, args...)
}

func Debugf(format string, args ...interface{}) {
	printf(LogDebug, format, args...)
}
