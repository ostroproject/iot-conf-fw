package confs

import (
	"log"
	"fmt"
	"flag"
	"strings"
)

const (
	Silent = uint(iota)
	LogError
	LogInfo
	LogDebug
	LogAll = LogDebug
)

var (
	initialized = false

	LogLevel    = LogError

	force       = false
	dropZone    = "/var/cache/confs"
	tmpDir      = "/var/cache/confs/tmp"
	defRoot     = "/usr/share/confs/ui"
	subTrees    = map[string]bool{"local":true, "common":true}
)

func init() {
	flag.UintVar(&LogLevel, "log-level", LogLevel, "log levels: 0=silent, 1=error, 2=info 3=debug")
	flag.BoolVar(&force, "force", force, "if true, files will be overwritten with identical content")
	flag.StringVar(&dropZone, "drop-zone", dropZone, "root directory DropZone")
	flag.StringVar(&defRoot, "definition-root", defRoot, "root directory of definitions")
}


func Initialize() error {
	var err error = nil

	if !initialized {
		if LogLevel > LogDebug {
			log.Printf("verbosity level %d is out of range (0 - %d)\n", LogLevel, LogDebug)
			log.Printf("using verbosity level %d\n", LogDebug)
			LogLevel = LogDebug
		}

		if IsValidPath(dropZone) {
			dropZone = strings.TrimRight(dropZone, "/")
		} else {
			Errorf("invalid DropZone root '%s': %v", dropZone, nameError)
			Infof("using hardwired '%s' as root of DropZone", "/var/cache/confs")
			err = pathError
		}

		tmpDir = fmt.Sprintf("%s/tmp", dropZone)

		if IsValidPath(defRoot) {
			defRoot = strings.TrimRight(defRoot, "/")
		} else {
			Errorf("invalid root of definitions '%s': %v", defRoot, nameError)
			Infof("using hardwired '%s' as root of definitions", "/usr/share/confs/ui")
			err = pathError
		}

		if LogLevel >= LogDebug {
			subtrees := ""
			for tree := range subTrees {
				subtrees += " " + tree
			}

			log.Printf("Confs pathes in use:\n")
			log.Printf("   drop zone root:      '%s'\n", dropZone)
			log.Printf("   tmp dir:             '%s'\n", tmpDir)
			log.Printf("   root of definitions: '%s'\n", defRoot)
			log.Printf("Accepted subtrees:%s\n", subtrees)
		}
	}

	return err
}
