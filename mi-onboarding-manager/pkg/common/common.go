/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"reflect"
	"strings"
)

func parseFieldName(f reflect.StructField) (name string, ignore bool) {
	tag := f.Tag.Get("json")

	if tag == "" {
		return f.Name, false
	}

	if tag == "-" {
		return "", true
	}

	if i := strings.Index(tag, ","); i != -1 {
		if i == 0 {
			return f.Name, false
		}
		return tag[:i], false
	}

	return tag, false
}

func GetFields(b interface{}) []string {
	val := reflect.ValueOf(b)
	n := val.Type().NumField()
	fields := make([]string, 0, n)
	for i := 0; i < n; i++ {
		f, ignore := parseFieldName(val.Type().Field(i))
		if !ignore && f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func GetValues(b interface{}) []interface{} {
	v := reflect.ValueOf(b)
	values := make([]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
	}
	return values
}

func GetMapValues(b interface{}) map[string]interface{} {
	v := reflect.ValueOf(b)
	n := v.Type().NumField()
	values := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		f, ignore := parseFieldName(v.Type().Field(i))
		if !ignore && f != "" {
			values[f] = v.Field(i).Interface()
		}
	}
	return values
}
