/*
 * Copyright 2021 Wang Min Xiang
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
 */

package versions

import (
	"fmt"
	"github.com/aacfactory/errors"
	"math"
	"strconv"
	"strings"
)

func New(major int, minor int, patch int) (v Version) {
	v = Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
	return
}

func Min() Version {
	return Version{
		Major: 0,
		Minor: 0,
		Patch: 0,
	}
}

func Max() Version {
	return Version{
		Major: math.MaxInt,
		Minor: math.MaxInt,
		Patch: math.MaxInt,
	}
}

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (ver Version) Between(left Version, right Version) (ok bool) {
	if right.IsZero() {
		right.Major = math.MaxInt
		right.Minor = math.MaxInt
		right.Patch = math.MaxInt
	}
	if ver.Major >= left.Major && ver.Major < right.Major {
		if ver.Minor >= left.Minor && ver.Minor < right.Minor {
			if ver.Patch >= left.Patch && ver.Patch < right.Patch {
				ok = true
			}
		}
	}
	return
}

func (ver Version) LessThan(o Version) (ok bool) {
	if ver.Major < o.Major {
		ok = true
		return
	}
	if ver.Minor < o.Minor {
		ok = true
		return
	}
	if ver.Patch < o.Patch {
		ok = true
		return
	}
	return
}

func (ver Version) IsZero() (ok bool) {
	ok = ver.Major == 0 && ver.Minor == 0 && ver.Patch == 0
	return
}

func (ver Version) String() (v string) {
	v = fmt.Sprintf("v%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
	return
}

func Parse(v string) (ver Version, err error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v[0] != 'v' {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", v)
		return
	}
	v = v[1:]
	items := strings.Split(v, ".")
	if len(items) != 3 {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", v)
		return
	}
	major, majorErr := strconv.Atoi(strings.TrimSpace(items[0]))
	if majorErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", v)
		return
	}
	minor, minorErr := strconv.Atoi(strings.TrimSpace(items[1]))
	if minorErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", v)
		return
	}
	patch, patchErr := strconv.Atoi(strings.TrimSpace(items[2]))
	if patchErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", v)
		return
	}
	ver = New(major, minor, patch)
	return
}
