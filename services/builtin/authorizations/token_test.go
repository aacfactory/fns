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

package authorizations_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/builtin/authorizations"
	"github.com/aacfactory/json"
	"testing"
	"time"
)

func TestDefaultTokens(t *testing.T) {
	config := json.NewObject()
	_ = config.Put("key", "key")
	opt, _ := configures.NewJsonConfig(config.Raw())
	tokens := authorizations.DefaultTokens()
	buildErr := tokens.Build(services.ComponentOptions{
		AppId:      "0",
		AppName:    "0",
		AppVersion: versions.Version{},
		Log:        nil,
		Config:     opt,
	})
	if buildErr != nil {
		t.Errorf("%+v", buildErr)
		return
	}
	attrs := services.RequestUserAttributes{}
	_ = attrs.Set("attr0", "0")
	token, createErr := tokens.Format(context.TODO(), authorizations.FormatTokenParam{
		Id:          "0",
		UserId:      "user:0",
		Attributes:  attrs,
		Expirations: 1 * time.Second,
	})
	if createErr != nil {
		t.Errorf("%+v", createErr)
		return
	}
	fmt.Println("token:", token)
	parsed, parseErr := tokens.Parse(context.TODO(), token)
	if parseErr != nil {
		t.Errorf("%+v", parseErr)
		return
	}
	fmt.Println(parsed.Valid, parsed.Id, parsed.UserId, parsed.Attributes, parsed.ExpireAT)
}
