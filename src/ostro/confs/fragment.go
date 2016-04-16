package confs

import (
	"fmt"
	"errors"
	"reflect"
)


var (
	sizeError   = errors.New("file size less tha 2B or exceeds 64kB")
	mapError    = errors.New("value cannot be mapped to directory tree")
)


type ConfFragment struct {
	path string		        // relative to DropZone
	typ string		        // eg. "json", "xml" or "ini"
	origin string
	content map[string]interface{} 
}

func NewConfFragment(confpath string, origin string, data []byte) (*ConfFragment, error) {

	cf := &ConfFragment{ path: confpath, origin: origin }
	err := error(nil)
	
	if !IsValidPath(confpath) {
		err = nameError
	} else {
		cf.typ, cf.content, err = Unmarshal(confpath, data)
	}

	return cf, err
}

func EmptyConfFragment(confpath string, typ, origin string) (*ConfFragment, error) {
	var cf *ConfFragment = nil
	var err error = nil
	
	if !IsValidPath(confpath) {
		err = nameError
	} else if typ != "json" && typ != "xml" && typ != "ini" {
		err = formatError
	} else {
		cf = &ConfFragment{ path: confpath, typ: typ, origin: origin, content: make(map[string]interface{}) }
	}

	return cf, err
}

func (me ConfFragment) String() string {
	b, err := me.Bytes(Pretty)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (me *ConfFragment) Path() string {
	return me.path
}

func (me *ConfFragment) Type() string {
	return me.typ
}

func (me *ConfFragment) Origin() string {
	return me.origin
}

func (me *ConfFragment) Bytes(pretty bool) ([]byte, error) {
	return me.ExportBytes(me.typ, pretty)
}

func (me *ConfFragment) ExportBytes(Type string, pretty bool) ([]byte, error) {
	return Marshal(me.path, Type, me.content, pretty)
}

func (me *ConfFragment) WriteDropZone() *Error {
	return me.ExportDropZone(me.typ)
}

func (me *ConfFragment) ExportDropZone(Type string) *Error {
	return me.traverseFragmentAndExport(Type, me.path, me.content)
}
	
func (me *ConfFragment) traverseFragmentAndExport(Type, confsPath string, tree map[string]interface{}) *Error {
	var ferr *Error
	var dropPath string
	
	if dropPath, ferr = CheckFilePath(confsPath, me.origin, true); ferr == nil {
		var content []byte
		var cerr error

		if content, cerr = me.ExportBytes(Type, Pretty); cerr != nil {
			return newError(cerr, me.path)
		}
		if err := writeFile(dropPath, content, me.origin); err != nil {
			return err
		}
		return nil
	}
	if _, err := CheckDirPath(confsPath, true); err != nil {
		return ferr
	}
	for name, value := range tree {
		extendedPath := fmt.Sprintf("%s/%s", confsPath, name)
		if reflect.ValueOf(value).Kind() != reflect.Map {
			return newError(mapError, extendedPath)
		}
		err := me.traverseFragmentAndExport(Type, extendedPath, tree[name].(map[string]interface{}))
		if err != nil {
			return err
		}
	}
	return nil
}
