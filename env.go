package hyperex

import "reflect"

// toEnvMap converts various environment types to map[string]any.
func toEnvMap(env any) map[string]any {
	if env == nil {
		return nil
	}

	if m, ok := env.(map[string]any); ok {
		return m
	}

	rv := reflect.ValueOf(env)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		m := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			m[iter.Key().String()] = iter.Value().Interface()
		}
		return m
	}

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		return structToMap(rv)
	}

	return nil
}

func structToMap(rv reflect.Value) map[string]any {
	rt := rv.Type()
	m := make(map[string]any, rv.NumField())

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		name, ok := envFieldName(field)
		if !ok {
			continue
		}
		m[name] = rv.Field(i).Interface()
	}

	return m
}

func envFieldName(field reflect.StructField) (string, bool) {
	if !field.IsExported() {
		return "", false
	}
	name := field.Tag.Get("expr")
	if name == "-" {
		return "", false
	}
	if name == "" {
		name = field.Name
	}
	return name, true
}
