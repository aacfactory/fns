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

package processes

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"unicode/utf8"
)

type Unit interface {
	Handle(ctx context.Context) (result interface{}, err error)
}

type Result struct {
	StepNo   int64
	StepNum  int64
	StepName string
	UnitNo   int64
	UnitNum  int64
	Data     interface{}
	Error    error
}

func (result Result) String() string {
	succeed := make([]byte, utf8.RuneLen(rune('√')))
	utf8.EncodeRune(succeed, rune('√'))
	status := "succeed"
	if result.Error != nil {
		if IsAbortErr(result.Error) {
			status = "aborted"
		} else {
			status = " failed"
		}
	}
	return fmt.Sprintf("[%d/%d] %s [%d/%d] %s", result.StepNo, result.StepNum, result.StepName, result.UnitNo, result.UnitNum, status)
}

type Step struct {
	no       int64
	name     string
	num      int64
	units    []Unit
	resultCh chan<- Result
}

func (step *Step) Execute(ctx context.Context) (err error) {
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}
	if step.units == nil || len(step.units) == 0 {
		return
	}
	unitNum := int64(len(step.units))
	stepResultCh := make(chan Result, unitNum)
	for i, unit := range step.units {
		unitNo := int64(i + 1)
		if unit == nil {
			step.resultCh <- Result{
				StepNo:   step.no,
				StepNum:  step.num,
				StepName: step.name,
				UnitNo:   unitNo,
				UnitNum:  unitNum,
				Data:     nil,
				Error:    errors.Warning("processes: unit is nil").WithMeta("step", step.name),
			}
			continue
		}
		go func(ctx context.Context, unitNo int64, unit Unit, step *Step, stepResultCh chan Result) {
			if ctx.Err() != nil {
				stepResultCh <- Result{
					StepNo:   step.no,
					StepNum:  step.num,
					StepName: step.name,
					UnitNo:   unitNo,
					UnitNum:  unitNum,
					Data:     nil,
					Error:    ctx.Err(),
				}
				return
			}
			data, unitErr := unit.Handle(ctx)
			defer func() {
				_ = recover()
			}()
			stepResultCh <- Result{
				StepNo:   step.no,
				StepNum:  step.num,
				StepName: step.name,
				UnitNo:   unitNo,
				UnitNum:  unitNum,
				Data:     data,
				Error:    unitErr,
			}
		}(ctx, unitNo, unit, step, stepResultCh)
	}
	resultErrs := errors.MakeErrors()
	executed := int64(0)
	for {
		result, ok := <-stepResultCh
		if !ok {
			err = errors.Warning("processes: panic")
			break
		}
		if result.Error != nil {
			resultErrs.Append(result.Error)
		}
		step.resultCh <- result
		executed++
		if executed >= unitNum {
			break
		}
	}
	err = resultErrs.Error()
	return
}
