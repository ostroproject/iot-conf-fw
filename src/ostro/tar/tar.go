package tar

import (
	"os"
	"io"
	"path"
	"strings"
	"errors"
	"archive/tar"
	"ostro/confs"
)

const (
	TarOriginated = "Tar"
)

var (
	prefixError = errors.New("prefix error")
	pathError   = errors.New("path contains invalid characters")
	formatError = errors.New("unsupported or invalid file format")
	sizeError   = errors.New("file exceeds 64kB")
	lengthError = errors.New("only part could be written")
	originError = errors.New("not Tar originated")
)

type Confs struct {
	path string
	prefix string
	file *os.File
	flag int
	tr *tar.Reader
}

func Initialize() bool {
	if err := confs.Initialize(TarOriginated); err != nil {
		return false;
	}

	return true
}

func NewConfs(tarPath, prefix string) (*Confs, bool)  {
	if len(prefix) < 1 || prefix[0] != '.' {
		confs.Errorf("Invalid prefix '%s': does not start with '.'\n", prefix)
		return nil, false
	}

	if prefix == "/" {
		prefix = ""
	}

	return &Confs{ path: tarPath, prefix: prefix, file: nil}, true
}

func (me *Confs) open(flag int) (success bool) {
	var err error = nil

	success = true

	if me.file == nil || me.flag != flag {
		me.close()
		if me.file, err = os.OpenFile(me.path, flag, 0444); err == nil {
			me.flag = flag
		} else {
			confs.Errorf("failed to open '%s' (flags 0x%x): %v\n", me.path, flag, err)
			success = false
		}
	}

	return
}

func (me *Confs) close() {
	if me.file != nil {
		me.file.Close()
		me.file = nil
	}
}

func (me *Confs) ExtractFiles(pattern string) bool {
	failed := []string{}
	
	if !me.open(os.O_RDONLY) {
		return false
	}

	reader := tar.NewReader(me.file)

	for {
		var confPath string

		hdr, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			confs.Errorf("failed to read '%s': %v\n", me.path, err)
			me.close()
			return false
		}
		info := hdr.FileInfo()

		if confPath, err = me.getConfPath(hdr.Name); err != nil {
			failed = printExtractError(failed, err, hdr)
			continue
		}

		switch {
		case info.Mode().IsRegular():
			if content, err := getFileContent(reader, hdr.Name, int(hdr.Size)); err != nil {
				failed = printExtractError(failed, err, hdr)
			} else {
				frag, err := confs.NewConfFragment(confPath, TarOriginated, content)
				if err != nil {
					failed = printExtractError(failed, err, hdr)
				}
				if err := frag.WriteDropZone(); err != nil {
					failed = printExtractError(failed, err, hdr)
				}
			}
		case info.IsDir():
		default:
			confs.Debugf("Skipping non-regular file '%s'\n", hdr.Name);
		}
	}

	me.close()
	
	if len(failed) > 0 {
		confs.Errorf("failed to extract some files from '%s': %s\n",
			me.path, strings.Join(failed, ", "))
		return false
	}
				
	return true
}

func (me *Confs) getConfPath(name string) (string, error) {
	if !strings.HasPrefix(name, me.prefix) {
		return "", prefixError
	}

	confPath := strings.TrimPrefix(name, me.prefix)

	if !confs.IsValidPath(confPath) {
		return "", pathError
	}

	return confPath, nil
}

func getFileContent(reader *tar.Reader, name string, size int) ([]byte, error) {
	noContent := []byte("")

	if size > 65536 {
		return noContent, sizeError
	}

	var content = make([]byte, size)

	if read, err := reader.Read(content);  read != size || err != nil {
		if err != nil {
			return noContent, err
		} else {
			return noContent, sizeError
		}
	}

	return content, nil
}


func printExtractError(failed []string, err error, hdr *tar.Header) []string {
	name := path.Base(hdr.Name)

	confs.Debugf("failed to extract '%s': %v", name, err)

	return append(failed, name)
}
