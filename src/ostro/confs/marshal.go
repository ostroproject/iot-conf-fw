package confs

import (
	"path"
	"bytes"
	"errors"
	"encoding/json"
	"encoding/xml"
)

const (
	Compatct = false
	Pretty = true
)

var (
	formatError = errors.New("unsupported or invalid format")
)


func Marshal(Path string, Type string, Value map[string]interface{}, pretty bool) ([]byte, error) {
	buf := []byte{}
	err := error(nil)

	if !IsValidPath(Path) {
		err = nameError
	} else {
		switch Type {
		case "ini":
			fallthrough
		case "json":
			if pretty {
				buf, err = json.MarshalIndent(Value, "", "    ")
			} else {
				buf, err = json.Marshal(Value)
			}
		case "xml":
			xmlContent := make(map[string]interface{})
			xmlContent[path.Base(Path)] = Value
			if pretty {
				buf, err = xml.MarshalIndent(xmlContent, "", "    ")
			} else {
				buf, err = xml.Marshal(xmlContent)
			}
			buf = append([]byte(xml.Header), buf...)
		default:
			err = formatError
		}
	}

	return buf, err
}


func Unmarshal(Path string, Data []byte) (string, map[string]interface{}, error) {
	var value map[string]interface{}
	var xmlValue map[string]interface{}
	var typ string
	var ok bool

	trimmedData := bytes.Trim(Data, " \t\n\r\f")
	err :=  error(nil)

	if !IsValidPath(Path) {
		err = nameError
	} else if len(trimmedData) < 2 || len(trimmedData) > 65536 {
		err = sizeError
	} else {
		switch {
		case trimmedData[0] == '{' && trimmedData[len(trimmedData)-1] == '}':
			typ = "json"
			value = make(map[string]interface{})
			err = json.Unmarshal(trimmedData, &value)

		case HasValidXmlProlog(string(trimmedData)):
			typ = "xml"
			if xmlValue, err = UnmarshalXmlObject(string(trimmedData)); err == nil {
				if value, ok = xmlValue[path.Base(Path)].(map[string]interface{}); !ok {
					err = formatError
				}
			}

		default:
			err = formatError
		}
	}

	return typ, value, err
}
