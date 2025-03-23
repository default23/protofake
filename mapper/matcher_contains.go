package mapper

import (
	"reflect"
	"strings"
)

func (m *ValueMatcher) contains(val any) bool {
	want, value := reflect.ValueOf(m.Value), reflect.ValueOf(val)

	switch value.Kind() { //nolint:exhaustive
	case reflect.String:
		if want.Kind() == reflect.String {
			return strings.Contains(value.String(), want.String())
		}

		if want.Kind() == reflect.Int32 { // maybe it's a rune
			return strings.ContainsRune(value.String(), m.Value.(int32))
		}

		return false

	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if deepEqual(value.Index(i), want) {
				return true
			}
		}

		return false

	default:
		return false
	}
}
