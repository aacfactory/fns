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

package initialization

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/fns/initialization/base"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var Command = &cli.Command{
	Name:        "init",
	Aliases:     nil,
	Usage:       "fns init --mod={mod} --img={docker image name} --work={true} --version={go version} {project dir}",
	Description: "init fns project",
	ArgsUsage:   "",
	Category:    "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "mod",
			Aliases:  []string{"m"},
			Required: true,
			Usage:    "project go mod path",
		},
		&cli.BoolFlag{
			Name:     "work",
			Aliases:  []string{"w"},
			Required: false,
			Usage:    "use go work",
		},
		&cli.StringFlag{
			Name:     "img",
			Aliases:  []string{"i"},
			Required: false,
			Usage:    "project docker image name",
		},
		&cli.StringFlag{
			Name:     "version",
			Required: false,
			Usage:    "go version, e.g.: 1.21.0",
		},
	},
	Action: func(ctx *cli.Context) (err error) {
		projectDir := strings.TrimSpace(ctx.Args().First())
		if projectDir == "" {
			projectDir = "."
		}
		if !filepath.IsAbs(projectDir) {
			projectDir, err = filepath.Abs(projectDir)
			if err != nil {
				err = errors.Warning("fns: init fns project failed").WithCause(err).WithMeta("dir", projectDir)
				return
			}
		}
		projectDir = filepath.ToSlash(projectDir)
		projectPath := strings.TrimSpace(ctx.String("mod"))
		img := strings.TrimSpace(ctx.String("img"))
		work := ctx.Bool("work")
		goVersion := ctx.String("version")
		if goVersion == "" {
			goVersion = runtime.Version()[2:]
		}
		if !ValidGoVersion(goVersion) {
			err = errors.Warning("fns: init fns project failed").WithCause(fmt.Errorf("go version must be gte %s", runtime.Version())).WithMeta("dir", projectDir).WithMeta("path", projectPath)
			return
		}
		writeErr := base.Write(ctx.Context, goVersion, projectPath, img, work, projectDir)
		if writeErr != nil {
			err = errors.Warning("fns: init fns project failed").WithCause(writeErr).WithMeta("dir", projectDir).WithMeta("path", projectPath)
			return
		}
		if work {
			fmt.Println("fns: project has been initialized, please run `go work sync` and `go mod tidy` to fetch requires and run `go generate` to generate source files!")
		} else {
			fmt.Println("fns: project has been initialized, please run `go mod tidy` to fetch requires and run `go generate` to generate source files!")
		}
		return
	},
}

func ParseGoVersion(s string) (v Version) {
	ss := strings.Split(s, ".")
	if len(ss) > 0 {
		v.Major, _ = strconv.Atoi(ss[0])
	}
	if len(ss) > 1 {
		v.Miner, _ = strconv.Atoi(ss[1])
	}
	if len(ss) > 2 {
		v.Miner, _ = strconv.Atoi(ss[2])
	}
	return
}

type Version struct {
	Major int
	Miner int
	Patch int
}

func ValidGoVersion(target string) (ok bool) {
	if !modfile.GoVersionRE.MatchString(target) {
		return
	}
	tv := ParseGoVersion(target)
	rv := ParseGoVersion(runtime.Version()[2:])
	if tv.Major >= rv.Major {
		if tv.Miner >= rv.Miner {
			if tv.Patch >= rv.Patch {
				ok = true
			}
		}
	}
	return
}
