package meta

import (
	"encoding/json"
	"strconv"
)

func GetBool(m map[string]string, key string) (bool, error) {
	if m == nil {
		return false, nil
	}
	return strconv.ParseBool(m[key])
}

func GetInt(m map[string]string, key string) (int, error) {
	if m == nil {
		return 0, nil
	}
	s, ok := m[key]
	if !ok {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func GetString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func GetList(m map[string]string, key string) ([]string, error) {
	if m == nil {
		return []string{}, nil
	}
	s, ok := m[key]
	if !ok {
		return []string{}, nil
	}
	v := make([]string, 0)
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}

func GetMap(m map[string]string, key string) (map[string]string, error) {
	if m == nil {
		return map[string]string{}, nil
	}
	s, ok := m[key]
	if !ok {
		return map[string]string{}, nil
	}
	v := make(map[string]string)
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}
