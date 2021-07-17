package fns

import (
	"fmt"
	"strings"
	"time"
)

func ParseTime(value string) (result time.Time, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		result = time.Time{}
		return
	}
	result, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return
	}

	result, err = time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return
	}

	result, err = time.Parse("2006-01-02T15:04:05", value)
	if err == nil {
		return
	}

	result, err = time.Parse("2006-01-02 15:04:05", value)
	if err == nil {
		return
	}

	// ISO8601
	result, err = time.Parse("2006-01-02T15:04:05.000Z0700", value)
	if err == nil {
		return
	}

	err = fmt.Errorf("parse %s to time failed", value)
	return
}
