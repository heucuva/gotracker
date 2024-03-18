package config

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

func setValueToField(f reflect.Value, val string) error {
	var err error
	switch v := f.Interface().(type) {
	case *bool:
		*v, err = strconv.ParseBool(val)
	case *int64:
		*v, err = strconv.ParseInt(val, 0, 64)
	case *int32:
		var iv int64
		iv, err = strconv.ParseInt(val, 0, 32)
		*v = int32(iv)
	case *int16:
		var iv int64
		iv, err = strconv.ParseInt(val, 0, 16)
		*v = int16(iv)
	case *int8:
		var iv int64
		iv, err = strconv.ParseInt(val, 0, 8)
		*v = int8(iv)
	case *int:
		var iv int64
		iv, err = strconv.ParseInt(val, 0, 0)
		*v = int(iv)
	case *uint64:
		*v, err = strconv.ParseUint(val, 0, 64)
	case *uint32:
		var iv uint64
		iv, err = strconv.ParseUint(val, 0, 32)
		*v = uint32(iv)
	case *uint16:
		var iv uint64
		iv, err = strconv.ParseUint(val, 0, 16)
		*v = uint16(iv)
	case *uint8:
		var iv uint64
		iv, err = strconv.ParseUint(val, 0, 8)
		*v = uint8(iv)
	case *uint:
		var iv uint64
		iv, err = strconv.ParseUint(val, 0, 0)
		*v = uint(iv)
	case *string:
		*v = val
	case *[]bool:
		*v, err = parseCSVBoolArray(val)
	case *[]int64:
		*v, err = parseCSVIntArray[int64](val)
	case *[]int32:
		*v, err = parseCSVIntArray[int32](val)
	case *[]int16:
		*v, err = parseCSVIntArray[int16](val)
	case *[]int8:
		*v, err = parseCSVIntArray[int8](val)
	case *[]int:
		*v, err = parseCSVIntArray[int](val)
	case *[]uintptr:
		*v, err = parseCSVUintArray[uintptr](val)
	case *[]uint64:
		*v, err = parseCSVUintArray[uint64](val)
	case *[]uint32:
		*v, err = parseCSVUintArray[uint32](val)
	case *[]uint16:
		*v, err = parseCSVUintArray[uint16](val)
	case *[]uint8:
		*v, err = parseCSVUintArray[uint8](val)
	case *[]uint:
		*v, err = parseCSVUintArray[uint](val)
	case *[]string:
		*v, err = parseCSVStringArray(val)
	default:
		err = fmt.Errorf("unhandled type: %T", f.Interface())
	}

	return err
}

func parseCSVStringArray(s string) ([]string, error) {
	return csv.NewReader(strings.NewReader(s)).Read()
}

func parseCSVBoolArray(s string) ([]bool, error) {
	recs, err := parseCSVStringArray(s)
	if err != nil {
		return nil, err
	}

	out := make([]bool, len(recs))
	for i, rec := range recs {
		v, err := strconv.ParseBool(rec)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}

	return out, nil
}

func parseCSVIntArray[T constraints.Signed](s string) ([]T, error) {
	recs, err := parseCSVStringArray(s)
	if err != nil {
		return nil, err
	}
	var empty T

	var bitSize int
	switch any(empty).(type) {
	case int64:
		bitSize = 64
	case int32:
		bitSize = 32
	case int16:
		bitSize = 16
	case int8:
		bitSize = 8
	case int:
		bitSize = 0
	default:
		return nil, fmt.Errorf("unhandled type: %T", empty)
	}

	out := make([]T, len(recs))
	for i, rec := range recs {
		v, err := strconv.ParseInt(rec, 0, bitSize)
		if err != nil {
			return nil, err
		}
		out[i] = T(v)
	}

	return out, nil
}

func parseCSVUintArray[T constraints.Unsigned](s string) ([]T, error) {
	recs, err := parseCSVStringArray(s)
	if err != nil {
		return nil, err
	}

	var empty T

	var bitSize int
	switch any(empty).(type) {
	case uintptr:
		bitSize = 64
	case uint64:
		bitSize = 64
	case uint32:
		bitSize = 32
	case uint16:
		bitSize = 16
	case uint8:
		bitSize = 8
	case uint:
		bitSize = 0
	default:
		return nil, fmt.Errorf("unhandled type: %T", empty)
	}

	out := make([]T, len(recs))
	for i, rec := range recs {
		v, err := strconv.ParseUint(rec, 0, bitSize)
		if err != nil {
			return nil, err
		}
		out[i] = T(v)
	}

	return out, nil
}
