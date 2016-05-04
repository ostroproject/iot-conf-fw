package rest

import (
	"fmt"
	"os"
	"bytes"
	"io/ioutil"
	"path/filepath"
	"encoding/json"
	"golang.org/x/sys/unix"
	"ostro/confs"
)



func traverseDirectoryTree(path string, values map[string]interface{}) error {
	var (
		info os.FileInfo
		entries []os.FileInfo
		value map[string]interface{}
		err error
	)

	if info, err = os.Stat(path); err != nil {
		return err
	}

	if info.IsDir() {
		if entries, err = ioutil.ReadDir(path); err != nil {
			return err
		}
		
		value = make(map[string]interface{})
		
		for _, e := range entries {
			if err = traverseDirectoryTree(fmt.Sprintf("%s/%s", path, e.Name()), value); err != nil {
				return err
			}
		}		
	} else {
		if value, err = readFile(path); err != nil {
			return err
		}
	}

	if len(value) > 0 {
		values[filepath.Base(path)] = value
	}

	return nil
}

func readFile(path string) (map[string]interface{}, error) {
	var (
		file *os.File
		info os.FileInfo
		values map[string]interface{}
		err error
	)
	
	if file, err = os.OpenFile(path, os.O_RDONLY, 0444);  err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		f.Close()
	}(file)


	if info, err = file.Stat(); err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		return values, nil
	}	
	if info.Size() > 65536 {
		return nil, fmt.Errorf("invalid content: too large")
	}

	size := int(info.Size())
	raw := make([]byte, size)

	if len, err := file.Read(raw); err != nil || len != size {
		if err != nil {
			return nil, err
		} else {
			return nil, fmt.Errorf("partial read: %d vs %d", len, size)
		}
	}

	content := bytes.Trim(raw, " \t\n\f")

	if len(content) < 2 {
		return nil, fmt.Errorf("invalid content: too short")
	}

	switch {
	case content[0] == '{' && content[len(content)-1] == '}':
		values = make(map[string]interface{})
		if err = json.Unmarshal(content, &values); err != nil {
			return nil, err
		}

	case confs.HasValidXmlProlog(string(content)):
		var xmlval map[string]interface{}
		if xmlval, err = confs.UnmarshalXmlObject(string(content)); err != nil {
			return nil, err
		}
		if len(xmlval) == 0 {
			return nil, fmt.Errorf("invalid content: malformed")
		}
		for k,v := range xmlval {
			if k == filepath.Base(path) {
				values = v.(map[string]interface{})
			} else {
				values = xmlval
			}
		}
		
	default:
		return nil, fmt.Errorf("invalid content: unsupported format")
	}

	return values, nil
}

func writeFile(path string, values map[string]interface{}) error {
	var (
		file *os.File
		info os.FileInfo
		tmpName string
		content []byte
		size int
		err error
	)

	if info, err = os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	} else {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", path)
		}
	
		origin := make([]byte, len(RestOriginated))
		_, err = unix.Getxattr(path, OriginAttr, origin)
		if err != nil || !bytes.Equal([]byte(RestOriginated), origin) {
			return fmt.Errorf("not owner of %s", path)
		}
	}

	if content, err = json.MarshalIndent(interface{}(values), "", "    "); err != nil {
		return err
	}
	
	if file, err = ioutil.TempFile(fileHandler.tmpdir, filepath.Base(path)); err == nil {
		if info, err = file.Stat(); err == nil {
			tmpName = fmt.Sprintf("%s/%s", fileHandler.tmpdir, info.Name())
			if size, err = file.Write(content); err == nil && size != len(content) {
				err = fmt.Errorf("partial write")
			}
		}
	}
	file.Close()
	if err != nil {
		return err
	}

	if err = unix.Setxattr(tmpName, OriginAttr, []byte(RestOriginated), 1); err != nil {
		os.Remove(tmpName)
		return err
	}
	
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}

	return nil
}
