package sprig

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/dustin/go-humanize"
)

func toBytes(v interface{}) string {
	s, _ := mustToBytes(v)
	return s
}

func mustToBytes(v interface{}) (string, error) {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return humanize.Bytes(uint64(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := n.Float64(); err != nil {
			return "", err
		} else {
			return humanize.Bytes(uint64(math.Round(f))), nil
		}
	case string:
		if n == "" {
			return "", nil
		} else if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return humanize.Bytes(uint64(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := strconv.ParseFloat(n, 64); err != nil {
			return "", err
		} else {
			return humanize.Bytes(uint64(math.Round(f))), nil
		}
	case int32:
		return humanize.Bytes(uint64(n)), nil
	case int64:
		return humanize.Bytes(uint64(n)), nil
	case int:
		return humanize.Bytes(uint64(n)), nil
	case float32:
		return humanize.Bytes(uint64(math.Round(float64(n)))), nil
	case float64:
		return humanize.Bytes(uint64(math.Round(n))), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("unknown %T with value %v", v, v)
}

func toIBytes(v interface{}) string {
	s, _ := mustToIBytes(v)
	return s
}

func mustToIBytes(v interface{}) (string, error) {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return humanize.IBytes(uint64(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := n.Float64(); err != nil {
			return "", err
		} else {
			return humanize.IBytes(uint64(math.Round(f))), nil
		}
	case string:
		if n == "" {
			return "", nil
		} else if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return humanize.IBytes(uint64(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := strconv.ParseFloat(n, 64); err != nil {
			return "", err
		} else {
			return humanize.IBytes(uint64(math.Round(f))), nil
		}
	case int32:
		return humanize.IBytes(uint64(n)), nil
	case int64:
		return humanize.IBytes(uint64(n)), nil
	case int:
		return humanize.IBytes(uint64(n)), nil
	case float32:
		return humanize.IBytes(uint64(math.Round(float64(n)))), nil
	case float64:
		return humanize.IBytes(uint64(math.Round(n))), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("unknown %T with value %v", v, v)
}

func toComma(v interface{}) string {
	s, _ := mustToComma(v)
	return s
}

func mustToComma(v interface{}) (string, error) {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return humanize.Comma(i), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := n.Float64(); err != nil {
			return "", err
		} else {
			return humanize.Commaf(f), nil
		}
	case string:
		if n == "" {
			return "", nil
		} else if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return humanize.Comma(i), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := strconv.ParseFloat(n, 64); err != nil {
			return "", err
		} else {
			return humanize.Commaf(f), nil
		}
	case int32:
		return humanize.Comma(int64(n)), nil
	case int64:
		return humanize.Comma(n), nil
	case int:
		return humanize.Comma(int64(n)), nil
	case float32:
		return humanize.Commaf(float64(n)), nil
	case float64:
		return humanize.Commaf(n), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("unknown %T with value %v", v, v)
}

func formatNumber(format string, v interface{}) string {
	s, _ := mustFormatNumber(format, v)
	return s
}

func mustFormatNumber(format string, v interface{}) (string, error) {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return humanize.FormatInteger(format, int(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := n.Float64(); err != nil {
			return "", err
		} else {
			return humanize.FormatFloat(format, f), nil
		}
	case string:
		if n == "" {
			return "", nil
		} else if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return humanize.FormatInteger(format, int(i)), nil
		}
		// Return a float64 (default json.Decode() behavior)
		// An overflow will return an error
		if f, err := strconv.ParseFloat(n, 64); err != nil {
			return "", err
		} else {
			return humanize.FormatFloat(format, f), nil
		}
	case int32:
		return humanize.FormatInteger(format, int(n)), nil
	case int64:
		return humanize.FormatInteger(format, int(n)), nil
	case int:
		return humanize.FormatInteger(format, n), nil
	case float32:
		return humanize.FormatFloat(format, float64(n)), nil
	case float64:
		return humanize.FormatFloat(format, n), nil
	case nil:
		return "", nil
	}
	return "", fmt.Errorf("unknown %T with value %v", v, v)
}
