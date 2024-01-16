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

package main

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/cmd/fns/initialization"
	"github.com/aacfactory/fns/cmd/fns/ssc"
	"github.com/urfave/cli/v2"
	"os"
)

const (
	Name      = "FNS"
	Version   = "v1.2.82"
	Usage     = "see COMMANDS"
	Copyright = `Copyright 2024 Wang Min Xiang

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`
)

func main() {
	app := cli.NewApp()
	app.Name = Name
	app.Version = Version
	app.Usage = Usage
	app.Authors = []*cli.Author{
		{
			Name:  "Wang Min Xiang",
			Email: "wangminxiang@aacfactory.co",
		},
	}
	app.Copyright = Copyright
	app.Commands = []*cli.Command{
		initialization.Command,
		ssc.Command,
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		fmt.Println(fmt.Sprintf("%+v", err))
	}
}
