package confs

import (
	"reflect"
)

func MergeFragment(fragment, result map[string]interface{}) error {
	for n, fi := range fragment {
		fk := reflect.ValueOf(fi).Kind()
		ri := result[n]

		if ri != nil && fk == reflect.ValueOf(ri).Kind() && fk == reflect.Map {
			MergeFragment(
				fi.(map[string]interface{}),
				ri.(map[string]interface{}))
		} else {
			result[n] = fi
		}
	}

	return nil
}
