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
	"github.com/aacfactory/json"
	"time"
)

func NewTimeRange(left time.Time, right time.Time) (v *TimeRange) {
	v = &TimeRange{
		Beg: left,
		End: right,
	}
	return
}

// TimeRange
// @title TimeRange
// @description `['2006-01-01T15:04:06', '2006-01-01T15:04:06')`
type TimeRange struct {
	// Beg
	// @title Beg
	// @description `2006-01-01T15:04:06`
	Beg time.Time `json:"beg"`
	// End
	// @title End
	// @description `2006-01-01T15:04:06`
	End time.Time `json:"end"`
}

func (tr *TimeRange) IsZero() (ok bool) {
	ok = tr.Beg.IsZero() && tr.End.IsZero()
	return
}

func NewDateRange(left json.Date, right json.Date) (v *DateRange) {
	v = &DateRange{
		Beg: left,
		End: right,
	}
	return
}

// DateRange
// @title DateRange
// @description `['2006-01-01', '2006-01-01')`
type DateRange struct {
	// Beg
	// @title Beg
	// @description `2006-01-01`
	Beg json.Date `json:"beg"`
	// End
	// @title End
	// @description `2006-01-01`
	End json.Date `json:"end"`
}

func (dr *DateRange) IsZero() (ok bool) {
	ok = dr.Beg.IsZero() && dr.End.IsZero()
	return
}
