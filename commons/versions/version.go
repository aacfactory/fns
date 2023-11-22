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

package versions

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"math"
	"strconv"
)

func New(major int, minor int, patch int) (v Version) {
	v = Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
	return
}

func Origin() Version {
	return Version{
		Major: 0,
		Minor: 0,
		Patch: 0,
	}
}

func Latest() Version {
	return Version{
		Major: math.MaxInt64,
		Minor: math.MaxInt64,
		Patch: math.MaxInt64,
	}
}

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (ver Version) Between(left Version, right Version) (ok bool) {
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
	if ver.Major == o.Major {
		if ver.Minor < o.Minor {
			ok = true
			return
		}
		if ver.Minor == o.Minor {
			if ver.Patch < o.Patch {
				ok = true
				return
			}
		}
		return
	}
	return
}

func (ver Version) Equals(o Version) (ok bool) {
	ok = ver.Major == o.Major && ver.Minor == o.Minor && ver.Patch == o.Patch
	return
}

func (ver Version) IsOrigin() (ok bool) {
	ok = ver.Major == 0 && ver.Minor == 0 && ver.Patch == 0
	return
}

func (ver Version) IsLatest() (ok bool) {
	ok = ver.Major == math.MaxInt64 && ver.Minor == math.MaxInt64 && ver.Patch == math.MaxInt64
	return
}

func (ver Version) String() (v string) {
	if ver.IsLatest() {
		v = "latest"
		return
	}
	v = fmt.Sprintf("v%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
	return
}

func Parse(v []byte) (ver Version, err error) {
	v = bytes.ToLower(bytes.TrimSpace(v))
	if bytes.Equal(v, []byte{'l', 'a', 't', 'e', 's', 't'}) {
		ver = Latest()
		return
	}
	if v[0] != 'v' {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", bytex.ToString(v))
		return
	}
	v = v[1:]
	items := bytes.Split(v, []byte{'.'})
	if len(items) != 3 {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", bytex.ToString(v))
		return
	}
	major, majorErr := strconv.Atoi(bytex.ToString(bytes.TrimSpace(items[0])))
	if majorErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", bytex.ToString(v))
		return
	}
	minor, minorErr := strconv.Atoi(bytex.ToString(bytes.TrimSpace(items[1])))
	if minorErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", bytex.ToString(v))
		return
	}
	patch, patchErr := strconv.Atoi(bytex.ToString(bytes.TrimSpace(items[2])))
	if patchErr != nil {
		err = errors.Warning("fns: parse version failed").WithCause(fmt.Errorf("invalid pattern")).WithMeta("version", bytex.ToString(v))
		return
	}
	ver = New(major, minor, patch)
	return
}
