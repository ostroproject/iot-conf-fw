package neard

import (
	"reflect"
	"github.com/godbus/dbus"
	"ostro/confs"
)

func DictStore(dict map[string]dbus.Variant, store reflect.Value) bool {
	for name, vari := range dict {
		if f := store.FieldByName(name); f.IsValid() {			
			v := vari.Value()
		
			if !f.CanSet() {
				confs.Errorf("field '%s' is not writeable", name)
				return false
			}

			if f.Kind() != reflect.Struct {
				if reflect.TypeOf(v) != f.Type() {
					confs.Errorf("field '%s' type mismatch", name)
					return false
				}

				f.Set(reflect.ValueOf(v))
			} else {
				if !DictStore(v.(map[string]dbus.Variant), f) {
					return false
				}
			}
		}
	}
	
	return true
}
