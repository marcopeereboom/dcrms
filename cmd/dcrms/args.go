package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inhies/go-bytesize"
)

// HasTrailingSlashes returns an error if any system has a trailing slash.
func HasTrailingSlashes(systems []string) error {
	for k := range systems {
		if strings.HasSuffix(systems[k], "/") {
			return fmt.Errorf("trailing slash on: %v", systems[k])
		}
	}
	return nil
}

// DoesNotHaveTrailingSlashes returns an error if any system does not have a
// trailing slash.
func DoesNotHaveTrailingSlashes(systems []string) error {
	for k := range systems {
		if !strings.HasSuffix(systems[k], "/") {
			return fmt.Errorf("no trailing slash on: %v",
				systems[k])
		}
	}
	return nil
}

func ParseArgs(args []string) (map[string]string, error) {
	m := make(map[string]string, len(args))
	for k := range args {
		a := strings.SplitN(args[k], "=", 2)
		if len(a) == 0 {
			return nil, fmt.Errorf("no argument: %v", args[k])
		}
		a1 := a[0]
		var a2 string
		if len(a) > 1 {
			a2 = a[1]
		}
		if _, ok := m[a1]; ok {
			return nil, fmt.Errorf("duplicate argument: %v", a1)
		}
		m[a1] = a2
	}
	return m, nil
}

func ArgAsInt(arg string, args map[string]string) (int, error) {
	if a, ok := args[arg]; ok {
		return strconv.Atoi(a)
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsByte(arg string, args map[string]string) (byte, error) {
	if a, ok := args[arg]; ok {
		x, err := strconv.ParseUint(a, 10, 64)
		if x > 255 {
			return 0, fmt.Errorf("%v not a byte", arg)
		}
		return byte(x), err
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsUint(arg string, args map[string]string) (uint, error) {
	if a, ok := args[arg]; ok {
		x, err := strconv.ParseUint(a, 10, 64)
		return uint(x), err
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsString(arg string, args map[string]string) (string, error) {
	if a, ok := args[arg]; ok {
		return a, nil
	}
	return "", fmt.Errorf("argument not found: %v", arg)
}

func ArgAsBool(arg string, args map[string]string) (bool, error) {
	if a, ok := args[arg]; ok {
		if a == "1" || strings.ToLower(a) == "true" {
			return true, nil
		}
		if a == "0" || strings.ToLower(a) == "false" {
			return false, nil
		}
	}
	return false, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsStringSlice(arg string, args map[string]string) ([]string, error) {
	if a, ok := args[arg]; ok {
		val := strings.Split(a, ",")
		return val, nil
	}
	return nil, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsDuration(arg string, args map[string]string) (time.Duration, error) {
	if a, ok := args[arg]; ok {
		return time.ParseDuration(a)
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsSize(arg string, args map[string]string) (bytesize.ByteSize, error) {
	if a, ok := args[arg]; ok {
		return bytesize.Parse(a)
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}

func ArgAsFloat(arg string, args map[string]string) (float64, error) {
	if a, ok := args[arg]; ok {
		return strconv.ParseFloat(a, 64)
	}
	return 0, fmt.Errorf("argument not found: %v", arg)
}
