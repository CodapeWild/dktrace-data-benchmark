/*
 *   Copyright (c) 2023 CodapeWild
 *   All rights reserved.

 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at

 *   http://www.apache.org/licenses/LICENSE-2.0

 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package agent

import (
	"context"
	"log"
)

type Agent interface{}

type Amplifier interface {
	ThreadRoutine(ID int, ctx context.Context, endpoint string, repeat int, trace any, threadDown chan int) error
}

type AmplifierFunc func(ID int, ctx context.Context, endpoint string, repeat int, trace any, threadDown chan int) error

func (ampf AmplifierFunc) ThreadRoutine(ID int, ctx context.Context, endpoint string, repeat int, trace any, threadDown chan int) error {
	return ampf(ID, ctx, endpoint, repeat, trace, threadDown)
}

type GeneralAmplifier struct {
	name            string
	threads, repeat int
	threadRoutine   AmplifierFunc
	close           chan struct{}
}

func (gamp *GeneralAmplifier) StartThreads(ctx context.Context, endpoint string, in chan any) (finish chan struct{}, err error) {
	if err = ctx.Err(); err != nil {
		return
	}

	finish = make(chan struct{})
	go func() {
		var (
			threadDown = make(chan int)
			finished   = 0
		)
		for {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Println(err.Error())
				} else {
					log.Println("GeneralAmplifier context done")
				}
			case <-gamp.close:
				log.Printf("GeneralAmplifier for %s exits", gamp.name)
			case trace := <-in:
				for i := 1; i <= gamp.threads; i++ {
					go gamp.ThreadRoutine(i, ctx, endpoint, gamp.repeat, trace, threadDown)
				}
			case tdID := <-threadDown:
				log.Printf("%s: thread: %d down", gamp.name, tdID)
				if finished++; gamp.threads == finished {
					log.Printf("%s: all threds finished", gamp.name)
					finish <- struct{}{}

					return
				}
			}
		}
	}()

	return
}

func (gamp *GeneralAmplifier) ThreadRoutine(ID int, ctx context.Context, endpoint string, repeat int, trace any, threadDown chan int) error {
	return gamp.threadRoutine(ID, ctx, endpoint, repeat, trace, threadDown)
}

func (gamp *GeneralAmplifier) Close() {
	select {
	case <-gamp.close:
	default:
		close(gamp.close)
	}
}

func NewGeneralAmplifier(name string, threads, repeat int, handler AmplifierFunc) *GeneralAmplifier {
	return &GeneralAmplifier{
		name:          name,
		threads:       threads,
		repeat:        repeat,
		threadRoutine: handler,
		close:         make(chan struct{}),
	}
}
