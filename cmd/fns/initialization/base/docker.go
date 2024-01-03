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

package base

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	dockerfile = `# USAGE
# docker build -t $$IMAGE_NAME:latest --build-arg VERSION=${VERSION} .

FROM golang:1.21-alpine3.19 AS builder

ARG VERSION=v0.0.1
ENV GO111MODULE on
# Enable goproxy
ENV GOPROXY https://goproxy.cn,direct

WORKDIR /build

COPY . .

RUN mkdir /dist \
    && go generate \
    && go build -ldflags "-X main.Version=${VERSION}" -o /dist/fapp \
    && cp -r configs /dist/configs


FROM alpine3.19

COPY --from=builder /dist /

# Note: when use UTC, then discard this.
RUN sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.cernet.edu.cn/alpine#g' /etc/apk/repositories &&  \
    apk add tzdata  \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /

# Note: expose port which is defined in config
EXPOSE 18080

ENTRYPOINT ["./fapp"]
`
)

func DockerImageNameFromMod(mp string) (name string) {
	modItems := strings.Split(mp, "/")
	modItemsLen := len(modItems)
	if modItemsLen == 1 {
		name = mp
	} else {
		appName := modItems[modItemsLen-1]
		domain := modItems[0]
		if modItemsLen > 2 {
			for i := modItemsLen - 2; i > 0; i-- {
				appName = modItems[i] + "-" + appName
			}
		}
		name = fmt.Sprintf("%s:%s", domain, appName)
	}
	return
}

func NewDockerFile(path string, dir string, modPath string) (mf *DockerFile, err error) {
	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			err = errors.Warning("fns: new main file failed").WithCause(err).WithMeta("dir", dir)
			return
		}
	}
	name := DockerImageNameFromMod(modPath)
	mf = &DockerFile{
		name:     name,
		path:     path,
		filename: filepath.ToSlash(filepath.Join(dir, "Dockerfile")),
	}
	return
}

type DockerFile struct {
	name     string
	path     string
	filename string
}

func (f *DockerFile) Name() (name string) {
	name = f.filename
	return
}

func (f *DockerFile) Write(_ context.Context) (err error) {
	content := strings.Replace(dockerfile, "$$IMAGE_NAME", f.name, 1)
	writeErr := os.WriteFile(f.filename, []byte(content), 0644)
	if writeErr != nil {
		err = errors.Warning("fns: dockerfile write failed").WithCause(writeErr).WithMeta("filename", f.filename)
		return
	}
	return
}
