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
			Required: true,
			Usage:    "project go mod path",
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
		if projectPath == "" {
			err = errors.Warning("fns: init fns project failed").WithCause(errors.Warning("path is required")).WithMeta("dir", projectDir)
			return
		}
		writeErr := base.Write(ctx.Context, projectPath, projectDir)
		if writeErr != nil {
			err = errors.Warning("fns: init fns project failed").WithCause(writeErr).WithMeta("dir", projectDir).WithMeta("path", projectPath)
			return
		}
		fmt.Println("fns: project has been initialized, please run `go mod tidy` to fetch requires and run `go generate` to generate source files!")
		return
	},
}
