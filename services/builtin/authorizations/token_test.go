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
	encoder := authorizations.DefaultTokenEncoder()
	buildErr := encoder.Construct(services.Options{
		Log:    nil,
		Config: opt,
	})
	if buildErr != nil {
		t.Errorf("%+v", buildErr)
		return
	}
	token, createErr := encoder.Encode(context.TODO(), authorizations.Authorization{
		Id:         "1",
		Account:    "1",
		Attributes: nil,
		ExpireAT:   time.Now(),
	})
	if createErr != nil {
		t.Errorf("%+v", createErr)
		return
	}
	fmt.Println("token:", token.String())
	parsed, parseErr := encoder.Decode(context.TODO(), token)
	if parseErr != nil {
		t.Errorf("%+v", parseErr)
		return
	}
	fmt.Println(fmt.Sprintf("%+v", parsed))
}
