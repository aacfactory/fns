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

package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	deps = "https://deps.dev/_/s/go/p"
)

func LatestVersion(path string) (v string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		err = errors.Warning("sources: get version from deps.dev failed").WithCause(errors.Warning("sources: path is required"))
		return
	}
	path = strings.ReplaceAll(url.PathEscape(path), "/", "%2F")
	http.DefaultClient.Timeout = 2 * time.Second
	resp, getErr := http.Get(fmt.Sprintf("%s/%s", deps, path))
	if getErr != nil {
		if errors.Map(getErr).Contains(http.ErrHandlerTimeout) {
			v, err = LatestVersionFromProxy(path)
			return
		}
		err = errors.Warning("sources: get version from deps.dev failed").
			WithCause(getErr).
			WithMeta("path", path)
		return
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusGatewayTimeout {
			v, err = LatestVersionFromProxy(path)
			return
		}
		err = errors.Warning("sources: get version from deps.dev failed").
			WithCause(errors.Warning("status code is not ok").WithMeta("status", strconv.Itoa(resp.StatusCode))).
			WithMeta("path", path)
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = errors.Warning("sources: get version from deps.dev failed").WithCause(readErr).WithMeta("path", path)
		return
	}
	_ = resp.Body.Close()
	result := DepResult{}
	decodeErr := json.Unmarshal(body, &result)
	if decodeErr != nil {
		err = errors.Warning("sources: get version from deps.dev failed").WithCause(decodeErr).WithMeta("path", path)
		return
	}
	v = result.Version.Version
	if v == "" {
		err = errors.Warning("sources: get version from deps.dev failed").WithCause(errors.Warning("sources: version was not found")).WithMeta("path", path)
		return
	}
	return
}

type DepVersion struct {
	Version string `json:"version"`
}

type DepResult struct {
	Version DepVersion `json:"version"`
}

func LatestVersionFromProxy(path string) (v string, err error) {
	goproxy, hasProxy := os.LookupEnv("GOPROXY")
	if !hasProxy || goproxy == "" {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(errors.Warning("goproxy was not set")).WithMeta("path", path)
		return
	}
	proxys := strings.Split(goproxy, ",")
	proxy := ""
	for _, p := range proxys {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			proxy = p
			break
		}
	}
	if proxy == "" {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(errors.Warning("goproxy is invalid")).WithMeta("path", path)
		return
	}
	http.DefaultClient.Timeout = 2 * time.Second
	resp, getErr := http.Get(fmt.Sprintf("%s/%s/@v/list", proxy, path))
	if getErr != nil {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(getErr).WithMeta("path", path)
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(readErr).WithMeta("path", path)
		return
	}
	_ = resp.Body.Close()
	if len(body) == 0 {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(errors.Warning("sources: version was not found")).WithMeta("path", path)
		return
	}
	idx := bytes.LastIndexByte(body, '\n')
	if idx < 0 {
		v = string(body)
		return
	}
	body = body[0:idx]
	idx = bytes.LastIndexByte(body, '\n')
	if idx < 0 {
		v = string(body)
		return
	}
	v = string(body[idx+1:])
	if v == "" {
		err = errors.Warning("sources: get version from goproxy failed").WithCause(errors.Warning("sources: version was not found")).WithMeta("path", path)
		return
	}
	return
}
