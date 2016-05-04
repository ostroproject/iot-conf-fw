package connman

import (
	"fmt"
	"reflect"
	"github.com/godbus/dbus"
)

func DictStore(dict map[string]dbus.Variant, store reflect.Value) error {
	for name, vari := range dict {
		if f := store.FieldByName(name); f.IsValid() {			
			v := vari.Value()
		
			if !f.CanSet() {
				return fmt.Errorf("field '%s' is not writeable", name)
			}

			if f.Kind() != reflect.Struct {
				if reflect.TypeOf(v) != f.Type() {
					return fmt.Errorf("field '%s' type mismatch", name)
				}

				f.Set(reflect.ValueOf(v))
			} else {
				if err := DictStore(v.(map[string]dbus.Variant), f); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}
