package mapper

import "reflect"

func (m *ValueMatcher) deepEqual(val any) bool {
	v1, v2 := reflect.ValueOf(m.Value), reflect.ValueOf(val)
	return deepEqual(v1, v2)
}

func deepEqual(v1, v2 reflect.Value) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return v1.IsValid() == v2.IsValid()
	}

	// Если типы совпадают — используем стандартное сравнение
	if v1.Type() == v2.Type() {
		return reflect.DeepEqual(v1.Interface(), v2.Interface())
	}

	// Numeric values, convert to float64 and compare
	if isNumeric(v1.Kind()) && isNumeric(v2.Kind()) {
		f1, ok1 := toFloat64(v1)
		f2, ok2 := toFloat64(v2)
		return ok1 && ok2 && f1 == f2
	}

	// strings
	if v1.Kind() == reflect.String && v2.Kind() == reflect.String {
		return v1.String() == v2.String()
	}

	// bools
	if v1.Kind() == reflect.Bool && v2.Kind() == reflect.Bool {
		return v1.Bool() == v2.Bool()
	}

	// slices
	if (v1.Kind() == reflect.Slice || v1.Kind() == reflect.Array) &&
		(v2.Kind() == reflect.Slice || v2.Kind() == reflect.Array) {
		if v1.Len() != v2.Len() {
			return false
		}
		for i := 0; i < v1.Len(); i++ {
			if !deepEqual(v1.Index(i), v2.Index(i)) {
				return false
			}
		}
		return true
	}

	// maps
	if v1.Kind() == reflect.Map && v2.Kind() == reflect.Map {
		if v1.Len() != v2.Len() {
			return false
		}
		for _, key := range v1.MapKeys() {
			val1 := v1.MapIndex(key)
			found := false
			for _, k2 := range v2.MapKeys() {
				if deepEqual(key, k2) && deepEqual(val1, v2.MapIndex(k2)) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// structs
	if v1.Kind() == reflect.Struct && v2.Kind() == reflect.Struct {
		fields1 := getExportedFields(v1)
		fields2 := getExportedFields(v2)

		if len(fields1) != len(fields2) {
			return false
		}
		for name, f1 := range fields1 {
			if f2, ok := fields2[name]; !ok || !deepEqual(f1, f2) {
				return false
			}
		}
		return true
	}

	// pointers
	if v1.Kind() == reflect.Ptr && v2.Kind() == reflect.Ptr {
		return deepEqual(v1.Elem(), v2.Elem())
	}

	return false
}

func isNumeric(k reflect.Kind) bool {
	switch k { //nolint:exhaustive
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func toFloat64(v reflect.Value) (float64, bool) {
	switch v.Kind() { //nolint:exhaustive
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

func getExportedFields(v reflect.Value) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() {
			fields[field.Name] = v.Field(i)
		}
	}
	return fields
}
