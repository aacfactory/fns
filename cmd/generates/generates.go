package generates

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/urfave/cli/v2"
)

func New(bin string, annotations ...sources.FnAnnotationCodeWriter) (generator *Generator) {
	// action
	action := &Action{
		annotations: annotations,
	}
	// app
	app := cli.NewApp()
	app.Name = "generator"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:     "verbose",
			EnvVars:  []string{"FNC_VERBOSE"},
			Aliases:  []string{"v"},
			Usage:    "verbose output",
			Required: false,
		},
		&cli.StringFlag{
			Name:      "work",
			Aliases:   []string{"w"},
			Usage:     "set workspace file path",
			Required:  false,
			EnvVars:   []string{"FNC_WORK"},
			TakesFile: false,
		},
	}
	app.Usage = fmt.Sprintf("%s {project path}", bin)
	app.Action = action.Handle
	// generator
	generator = &Generator{
		app: app,
	}
	return
}

type Generator struct {
	app *cli.App
}

func (generator *Generator) Execute(ctx context.Context, args ...string) (err error) {
	err = generator.app.RunContext(ctx, args)
	return
}

type Action struct {
	annotations sources.FnAnnotationCodeWriters
}

func (action *Action) Handle(ctx *cli.Context) (err error) {

	return
}
