package generates

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cmd/generates/processes"
	"github.com/aacfactory/fns/cmd/generates/sources"
	"github.com/aacfactory/fns/cmd/generates/writers"
	"github.com/urfave/cli/v2"
	"path/filepath"
	"strings"
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
	verbose := ctx.Bool("verbose")
	projectDir := strings.TrimSpace(ctx.Args().First())
	if projectDir == "" {
		projectDir = "."
	}
	if !filepath.IsAbs(projectDir) {
		projectDir, err = filepath.Abs(projectDir)
		if err != nil {
			err = errors.Warning("generates: generate failed").WithCause(err).WithMeta("dir", projectDir)
			return
		}
	}
	projectDir = filepath.ToSlash(projectDir)
	work := ctx.String("work")
	if work != "" {
		work = strings.TrimSpace(work)
		if work == "" {
			err = errors.Warning("generates: generate failed").WithCause(errors.Warning("work option is invalid"))
			return
		}
	}
	process, processErr := action.process(ctx.Context, projectDir, work)
	if processErr != nil {
		err = errors.Warning("generates: generate failed").WithCause(processErr).WithMeta("dir", projectDir)
		return
	}
	results := process.Start(ctx.Context)
	for {
		result, ok := <-results
		if !ok {
			if verbose {
				fmt.Println("generates: generate finished")
			}
			break
		}
		if verbose {
			fmt.Println(result, "->", fmt.Sprintf("[%d/%d]", result.UnitNo, result.UnitNum), result.Data)
		}
		if result.Error != nil {
			fmt.Println(fmt.Sprintf("%+v", result.Error))
		}
	}
	return
}

func (action *Action) process(ctx context.Context, project string, workspace string) (controller processes.ProcessController, err error) {
	// parse mod
	moduleFilename := filepath.Join(project, "go.mod")
	var mod *sources.Module
	if workspace != "" {
		mod, err = sources.NewWithWork(moduleFilename, workspace)
	} else {
		mod, err = sources.New(moduleFilename)
	}
	if err != nil {
		err = errors.Warning("sources: generate failed").WithCause(err)
		return
	}
	parseModErr := mod.Parse(ctx)
	if parseModErr != nil {
		err = errors.Warning("sources: generate failed").WithCause(parseModErr)
		return
	}
	// parse services
	services, parseServicesErr := mod.Services()
	if parseServicesErr != nil {
		err = errors.Warning("sources: generate failed").WithCause(parseServicesErr)
		return
	}
	// make process controller
	process := processes.New()
	if len(services) == 0 {
		controller = process
		return
	}

	functionParseUnits := make([]processes.Unit, 0, 1)
	serviceCodeFileUnits := make([]processes.Unit, 0, 1)
	for _, service := range services {
		for _, function := range service.Functions {
			functionParseUnits = append(functionParseUnits, function)
		}
		serviceCodeFileUnits = append(serviceCodeFileUnits, writers.Unit(writers.NewServiceFile(service)))
	}
	process.Add("services: parsing", functionParseUnits...)
	process.Add("services: writing", serviceCodeFileUnits...)
	process.Add("services: deploys", writers.Unit(writers.NewDeploysFile(filepath.ToSlash(filepath.Join(mod.Dir, "modules")), services)))
	controller = process
	return
}