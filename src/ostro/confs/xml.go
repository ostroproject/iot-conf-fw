package confs

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"bytes"
	"regexp"
	"encoding/xml"
)

type xmlReader struct {
	offset int
	content []byte
}

func newXmlReader(content string) *xmlReader {
	trimmed := []byte(strings.Trim(content, " \t\n") + "\n")
	return &xmlReader{0, trimmed}
}

func (me *xmlReader) Read(p []byte) (int, error) {
	lgh := len(me.content) - me.offset

	if lgh <= 0 {
		return 0, io.EOF
	}

	if lgh > len(p) {
		return 0, io.ErrShortBuffer
	}
	
	for i:= 0;  i < lgh;  i++ {
		p[i] = me.content[me.offset + i]
	}
	me.offset += lgh
	
	return lgh, nil
}


func getTokenName(token xml.Token) string {
	name := []byte(reflect.ValueOf(token).FieldByName("Name").FieldByName("Local").String())
	dup := make([]byte, len(name))
	copy(dup, name)
	return string(dup)
}

func getTokenAttributes(token xml.Token) []xml.Attr {
	return reflect.ValueOf(token).FieldByName("Attr").Interface().([]xml.Attr)
}

func getAttributeName(attr xml.Attr) string {
	name := reflect.ValueOf(attr).FieldByName("Name").FieldByName("Local").String()
	return strings.Title(name)
}

func getAttributeValue(attr xml.Attr) string {
	value := []byte(reflect.ValueOf(attr).FieldByName("Value").String())
	dup := make([]byte, len(value))
	copy(dup, value)
	return string(dup)
}


func findStartToken(d *xml.Decoder) (xml.Token, error) {
	var err error
	
	for token, err := d.Token();  err == nil;  token, err = d.Token() {
		switch token.(type) {
		case xml.StartElement:
			return token, nil
		}
	}

	return nil, err
}


func unmarshalObject(d *xml.Decoder, objectName string) (interface{}, error) {
	var token xml.Token
	var objectFields map[string]interface{}
	var arrayFields []interface{}
	var err error

	name := ""
	emptyValue := interface{}("")
	value := emptyValue
	hasValue := false
	hasObject := false
	
	for token, err = d.Token();   err == nil;   token, err = d.Token() {
		switch token.(type) {
		case xml.CharData:
			charData := bytes.Trim(reflect.ValueOf(token).Bytes(), " \t\n")
			if len(charData) > 0 {
				if hasValue || hasObject {
					return emptyValue, fmt.Errorf("unexpected value '%s'", charData)
				} else {
					hasValue = true
					dup := make([]byte, len(charData))
					copy(dup, charData)
					value = string(dup)
				}
			}

		case xml.StartElement:
			name = getTokenName(token)
			if hasValue {
				return emptyValue, fmt.Errorf("unexpected tag <%s>\n", name)
			}
			if !hasObject {
				hasObject = true
				objectFields = make(map[string]interface{})
				value = objectFields
			}
			if field, err := unmarshalObject(d, name); err != nil {
				return emptyValue, err
			} else {
				currentValue := objectFields[name]
				if currentValue == nil {
					for _, attr := range getTokenAttributes(token) {
						objectFields[name + getAttributeName(attr)] = getAttributeValue(attr)
					}
					objectFields[name] = field
				} else {
					v := reflect.ValueOf(currentValue)
					if v.Kind() == reflect.Slice {
						arrayFields = currentValue.([]interface{})
					} else {
						arrayFields = []interface{}{currentValue}
					}
					objectFields[name] = append(arrayFields, field)
				}
			}
			
		case xml.EndElement:
			end := getTokenName(token)
			if end == objectName {
				return value, nil
			} else {
				return emptyValue, fmt.Errorf("unexpected end-tag </%s>", end)
			}
		}
	}

	return emptyValue, fmt.Errorf("incomplete/malformed XML")
}

func HasValidXmlProlog(xml string) bool {
	validXML := regexp.MustCompile(`<\?xml([^?]*)\?>`)
	singleSpace := regexp.MustCompile(`([ \t\f\n]+)`)
	hasVersion := false
	hasEncoding := false

	prolog := validXML.Find([]byte(xml))
	if prolog == nil {
		return false
	}

	attrs := strings.Split(strings.Trim(string(singleSpace.ReplaceAll(prolog[5:len(prolog)-2],[]byte(" "))), " \t"), " ")

	for _,a := range attrs {
		kv := strings.Split(a, "=")

		if len(kv) != 2 {
			return false
		}

		k := strings.Trim(kv[0], " \t")
		v := strings.Trim(kv[1], " \t")

		switch k {
		case "version":
			hasVersion = true
		case "encoding":
			if v != `"UTF-8"` {
				return false
			}
			hasEncoding = true
		}
	}

	if !hasVersion || !hasEncoding {
		return false
	}

	return true
}

func UnmarshalXmlObject(xmlString string) (map[string]interface{}, error) {
	var startToken xml.Token
	d := xml.NewDecoder(newXmlReader(xmlString))
	result := make(map[string]interface{})	
	err := error(nil)

	if startToken, err = findStartToken(d); err == nil {
		var value interface{}
		key := getTokenName(startToken)
		value, err = unmarshalObject(d, key)
		result[key] = value
	}

	return result, err
}

func UnmarshalXmlArray(xmlString string) ([]interface{}, error) {
	var token xml.Token
	var name string
	
	d := xml.NewDecoder(newXmlReader(xmlString))
	emptyResult := make([]interface{}, 0)
	result := make([]interface{}, 0)
	hasValue := false
	hasObject := false
	hasName := false
	elemName := ""
	
	for err := error(nil);   err == nil;   token, err = d.Token() {
		switch token.(type) {
		case xml.CharData:
			charData := bytes.Trim(reflect.ValueOf(token).Bytes(), " \t\n")
			if len(charData) > 0 {
				if hasValue || hasObject || !hasName {
					return emptyResult, fmt.Errorf("unexpected value '%s'", charData)
				} else {
					hasValue = true
					dup := make([]byte, len(charData))
					copy(dup, charData)
					result = append(result, string(dup))
				}
			}

		case xml.StartElement:
			name = getTokenName(token)
			if !hasName {
				if len(elemName) == 0 {
					elemName = name
				} else if name != elemName {
					return emptyResult, fmt.Errorf("unexpected tag <%s>\n", name)
				}
				hasName = true
			} else {
				if hasValue {
					return emptyResult, fmt.Errorf("unexpected tag <%s>\n", name)
				}
				hasObject = true
				var value interface{}
				if value, err = unmarshalObject(d, name); err != nil {
					return emptyResult, err
				}
				result = append(result, value)
			}
			
		case xml.EndElement:
			end := getTokenName(token)
			if end != elemName {
				return emptyResult, fmt.Errorf("unexpected end-tag </%s>", end)
			}
			hasName = false
			hasValue = false
		}
	}

	return result, nil
}
