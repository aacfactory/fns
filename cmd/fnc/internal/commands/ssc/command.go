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

package ssc

import (
	"github.com/aacfactory/errors"
	"github.com/urfave/cli/v2"
	"strings"
)

var Command = &cli.Command{
	Name:        "ssc",
	Aliases:     nil,
	Usage:       "fnc ssc --cn=common_name .",
	Description: "create self signed ca",
	ArgsUsage:   "",
	Category:    "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Required: false,
			Name:     "cn",
			Value:    "",
			Usage:    "common name",
		},
		&cli.BoolFlag{
			Required: false,
			Name:     "full",
			Value:    false,
			Usage:    "create all tls files",
		},
	},
	Action: func(ctx *cli.Context) (err error) {
		cn := ctx.String("cn")
		if cn == "" {
			cn = "FNS"
		}
		full := ctx.Bool("full")
		outputDir := strings.TrimSpace(ctx.Args().First())
		if outputDir == "" {
			err = errors.Warning("fnc: create ssc failed").WithCause(errors.Warning("output is undefined"))
			return
		}
		err = generate(cn, full, outputDir)
		if err != nil {
			err = errors.Warning("fnc: create ssc failed").WithCause(err)
			return
		}
		return
	},
}
