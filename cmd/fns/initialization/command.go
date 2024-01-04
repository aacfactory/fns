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
	"path/filepath"
	"strings"
)

var Command = &cli.Command{
	Name:        "init",
	Aliases:     nil,
	Usage:       "fns init --mod={mod} {project dir}",
	Description: "init fns project",
	ArgsUsage:   "",
	Category:    "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "mod",
			Aliases:  []string{"m"},
			Required: false,
			Usage:    "project go mod path",
		},
		&cli.StringFlag{
			Name:     "img",
			Aliases:  []string{"i"},
			Required: false,
			Usage:    "project docker image name",
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
		if projectPath == "" {
			// huh
			if err = useForm(); err != nil {
				err = errors.Warning("fns: init fns project failed").WithCause(err).WithMeta("dir", projectDir)
				return
			}
			projectPath = modPath
			img = dockerImageName
			return
		}
		writeErr := base.Write(ctx.Context, projectPath, img, projectDir)
		if writeErr != nil {
			err = errors.Warning("fns: init fns project failed").WithCause(writeErr).WithMeta("dir", projectDir).WithMeta("path", projectPath)
			return
		}
		fmt.Println("fns: project has been initialized, please run `go mod tidy` to fetch requires and run `go generate` to generate source files!")
		return
	},
}
