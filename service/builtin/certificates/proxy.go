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

package certificates

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

func Create(ctx context.Context, cert *Certificate) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: create failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	_, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, createFn, service.NewArgument(cert), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	return
}

func Get(ctx context.Context, id string) (cert *Certificate, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: get failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, getFn, service.NewArgument(id), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	if !future.Exist() {
		return
	}
	cert = &Certificate{}
	scanErr := future.Scan(cert)
	if scanErr != nil {
		err = errors.Warning("certificates: get failed").WithCause(scanErr)
		return
	}
	return
}

func Remove(ctx context.Context, id string) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: remove failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	_, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, removeFn, service.NewArgument(id), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	return
}
