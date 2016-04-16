package main

import (
	"fmt"
	"log"
	"flag"
	"os"
	"time"
	"ostro/tar"
)

func main() {
	var (
		tarDir, tarFile, archivePrefix string
		cfg *tar.Confs
		success bool
	)

	flag.StringVar(&tarDir, "tar-dir", "", "directory where the tar file is")
	flag.StringVar(&tarFile, "tar-file", "confs.tar", "tar file name")
	flag.StringVar(&archivePrefix, "archive-prefix", ".", "prefix to strip from tar files")

	flag.Parse()

	tar.Initialize()

	tarFilePath := fmt.Sprintf("%s/%s", tarDir, tarFile)

	checkIfFileExists(tarFilePath)
	
	if cfg, success = tar.NewConfs(tarFilePath, archivePrefix); !success {
		os.Exit(1)
	}

	if success = cfg.ExtractFiles("*"); !success {
		os.Exit(1)
	}
}


func checkIfFileExists(path string) {
	wait := 100 * time.Millisecond
	
	for i := 0;  i < 12;  i++ {
		if info, err := os.Stat(path); err == nil {
			if !info.Mode().IsRegular() {
				log.Fatalf("'%s' is not a regular file\n", path);
			}
			if info.Size() > int64(65536) {
				log.Fatalf("Size of '%s' exceeds the allowed 64kB\n", path)
			}
			return
		}
		if i > 5 {
			wait *= 2
		}
		time.Sleep(wait)
	}

	log.Fatalf("File '%s' not found\n", path)
}
