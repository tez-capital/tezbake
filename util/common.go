package util

import (
	"reflect"
)

func AssignIfEmpty[T comparable](val *T, candidate T) {
	if val == nil {
		return
	}
	var zero T
	if *val == zero {
		*val = candidate
	}
}

func AssignIfEmptyR(val reflect.Value, candidate reflect.Value) {
	if val.IsZero() {
		val.Set(candidate)
	}
}
func AssignStructFieldsIfEmptyR(targetReflect reflect.Value, sourceReflect reflect.Value) {
	for i := 0; i < targetReflect.NumField(); i++ {
		fieldType := targetReflect.Type().Field(i)

		// If PkgPath is non-empty, the field is unexported, so skip it
		if fieldType.PkgPath != "" {
			continue
		}
		field := targetReflect.Field(i)
		switch field.Kind() {
		case reflect.Struct:
			AssignStructFieldsIfEmptyR(field, sourceReflect.Field(i))
		default:
			AssignIfEmptyR(field, sourceReflect.Field(i))
		}
	}

}

func AssignStructFieldsIfEmpty[T any](target *T, source *T) {
	targetReflect := reflect.ValueOf(target).Elem()
	sourceReflect := reflect.ValueOf(source).Elem()
	AssignStructFieldsIfEmptyR(targetReflect, sourceReflect)
}
