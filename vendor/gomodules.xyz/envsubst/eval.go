package envsubst

import "os"

// Eval replaces ${var} in the string based on the mapping function.
func Eval(s string, mapping func(string) string) (string, error) {
	t, err := Parse(s)
	if err != nil {
		return s, err
	}
	// convert mapping to match new mapper function
	mapper := func(node string, key string, args []string) (string, []string, error) {
		return mapping(key), args, nil
	}
	return t.Execute(mapper)
}

// EvalEnv replaces ${var} in the string according to the values of the
// current environment variables. References to undefined variables are
// replaced by the empty string.
func EvalEnv(s string) (string, error) {
	return Eval(s, os.Getenv)
}

func EvalMap(s string, values map[string]string) (string, error) {
	if values == nil {
		values = make(map[string]string)
	}
	mapper := func(node string, key string, args []string) (string, []string, error) {
		v, ok := values[key]
		// return error if key not found and default not specified
		if !ok && !isDefault(node) {
			return "", nil, &valueNotFoundError{key}
		}
		// if key found, remove args for default
		// so that empty value will not be replaced by default value
		if ok && isDefault(node) {
			return v, nil, nil
		}
		return v, args, nil
	}

	t, err := Parse(s)
	if err != nil {
		return s, err
	}
	return t.Execute(mapper)
}

func isDefault(name string) bool {
	switch name {
	case "=", ":=", ":-":
		return true
	default:
		return false
	}
}
