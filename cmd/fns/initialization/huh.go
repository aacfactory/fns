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
	"github.com/charmbracelet/huh"
	"golang.org/x/mod/module"
)

var (
	confirm         = false
	modPath         = ""
	dockerImageName = ""
)

func useForm() (err error) {

	form := huh.NewForm(
		// mod and docker image
		huh.NewGroup(
			huh.NewInput().
				Title("What's your project mod path?").
				Value(&modPath).
				Validate(func(s string) error {
					return module.CheckFilePath(s)
				}),
			huh.NewInput().
				Title("What's your project docker image name?").
				Value(&dockerImageName),
		),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Confirm?").
				Value(&confirm),
		),
	)

	if !confirm {
		err = fmt.Errorf("aborted")
		return
	}

	err = form.Run()
	return
}
