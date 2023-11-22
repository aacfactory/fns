/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package times

import (
	"fmt"
	"github.com/aacfactory/json"
	"reflect"
	"time"
)

func DateNow() Date {
	return MapToDate(time.Now())
}

func NewDate(year int, month time.Month, day int) Date {
	return Date{
		Year:  year,
		Month: month,
		Day:   day,
	}
}

func MapToDate(t time.Time) Date {
	return NewDate(t.Year(), t.Month(), t.Day())
}

func NewDateFromJsonDate(d json.Date) Date {
	return NewDate(d.Year, d.Month, d.Day)
}

type Date struct {
	Year  int
	Month time.Month
	Day   int
}

func (d *Date) UnmarshalJSON(p []byte) error {
	if p == nil || len(p) < 3 {
		return nil
	}
	p = p[1 : len(p)-1]
	v, parseErr := time.Parse("2006-01-02", string(p))
	if parseErr != nil {
		return fmt.Errorf("unmarshal %s failed, layout of date must be 2006-01-02, %v", string(p), parseErr)
	}
	d.Year = v.Year()
	d.Month = v.Month()
	d.Day = v.Day()
	return nil
}

func (d *Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", d.ToTime().Format("2006-01-02"))), nil
}

func (d *Date) ToTime() time.Time {
	if d.Year < 1 {
		d.Year = 1
	}
	if d.Month < 1 {
		d.Month = 1
	}
	if d.Day < 1 {
		d.Day = 1
	}
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.Local)
}

func (d *Date) IsZero() (ok bool) {
	ok = d.Year < 2 && d.Month < 2 && d.Day < 2
	return
}

func (d *Date) String() string {
	return d.ToTime().Format("2006-01-02")
}

func (d *Date) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	x := ""
	switch src.(type) {
	case string:
		x = src.(string)
	case []byte:
		x = string(src.([]byte))
	case time.Time:
		v := src.(time.Time)
		d.Year = v.Year()
		d.Month = v.Month()
		d.Day = v.Day()
		return nil
	default:
		return fmt.Errorf("fns: scan date raw value failed for %v is not supported", reflect.TypeOf(src).String())
	}
	if x == "" {
		return nil
	}
	v, parseErr := time.Parse("2006-01-02", x)
	if parseErr != nil {
		return fmt.Errorf("fns: scan date value failed, parse %s failed, %v", x, parseErr)
	}
	d.Year = v.Year()
	d.Month = v.Month()
	d.Day = v.Day()
	return nil
}
