/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipfix

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
)

// This is taken from Kubernetes' controller-runtime/log package, except for not exposing
// any types, which appears unnecessary, but the implementation of delegated logging is
// kinda neat.
func SetLogger(l logr.Logger) {
	logFullfilled.Store(true)
	rootLog.Fulfill(l.GetSink())
}

func FromContext(ctx context.Context, keysAndValues ...interface{}) logr.Logger {
	log := Log
	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = logger
		}
	}
	return log.WithValues(keysAndValues...)
}

func IntoContext(ctx context.Context, l logr.Logger) context.Context {
	return logr.NewContext(ctx, l)
}

func eventuallyFulfillRoot() {
	if logFullfilled.Load() {
		return
	}
	if time.Since(rootLogCreated).Seconds() >= 30 {
		if logFullfilled.CompareAndSwap(false, true) {
			stack := debug.Stack()
			stackLines := bytes.Count(stack, []byte{'\n'})
			sep := []byte{'\n', '\t', '>', ' ', ' '}

			fmt.Fprintf(os.Stderr,
				"ipfix.SetLogger(...) was never called; logs will not be displayed.\nDetected at:%s%s", sep,
				// prefix every line, so it's clear this is a stack trace related to the above message
				bytes.Replace(stack, []byte{'\n'}, sep, stackLines-1),
			)
			SetLogger(logr.New(nullLogSink{}))
		}
	}
}

var (
	logFullfilled atomic.Bool
)

var (
	rootLog, rootLogCreated = func() (*delegatingLogSink, time.Time) {
		return newDelegatingLogSink(nullLogSink{}), time.Now()
	}()
	Log = logr.New(rootLog)
)

type nullLogSink struct{}

var _ logr.LogSink = nullLogSink{}

func (nullLogSink) Init(logr.RuntimeInfo) {}

func (nullLogSink) Info(_ int, _ string, _ ...interface{}) {}

func (nullLogSink) Error(_ error, _ string, _ ...interface{}) {}

func (nullLogSink) Enabled(_ int) bool {
	return false
}

func (log nullLogSink) WithName(_ string) logr.LogSink {
	return log
}

func (log nullLogSink) WithValues(_ ...interface{}) logr.LogSink {
	return log
}

type loggerPromise struct {
	logger        *delegatingLogSink
	childPromises []*loggerPromise
	promisesLock  sync.Mutex

	name *string
	tags []interface{}
}

func (p *loggerPromise) WithName(l *delegatingLogSink, name string) *loggerPromise {
	res := &loggerPromise{
		logger:       l,
		name:         &name,
		promisesLock: sync.Mutex{},
	}

	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	p.childPromises = append(p.childPromises, res)
	return res
}

func (p *loggerPromise) WithValues(l *delegatingLogSink, tags ...interface{}) *loggerPromise {
	res := &loggerPromise{
		logger:       l,
		tags:         tags,
		promisesLock: sync.Mutex{},
	}

	p.promisesLock.Lock()
	defer p.promisesLock.Unlock()
	p.childPromises = append(p.childPromises, res)
	return res
}

func (p *loggerPromise) Fulfill(parentLogSink logr.LogSink) {
	sink := parentLogSink
	if p.name != nil {
		sink = sink.WithName(*p.name)
	}

	if p.tags != nil {
		sink = sink.WithValues(p.tags...)
	}

	p.logger.lock.Lock()
	p.logger.logger = sink
	if withCallDepth, ok := sink.(logr.CallDepthLogSink); ok {
		p.logger.logger = withCallDepth.WithCallDepth(1)
	}
	p.logger.promise = nil
	p.logger.lock.Unlock()

	for _, childPromise := range p.childPromises {
		childPromise.Fulfill(sink)
	}
}

type delegatingLogSink struct {
	lock    sync.RWMutex
	logger  logr.LogSink
	promise *loggerPromise
	info    logr.RuntimeInfo
}

func (l *delegatingLogSink) Init(info logr.RuntimeInfo) {
	eventuallyFulfillRoot()
	l.lock.Lock()
	defer l.lock.Unlock()
	l.info = info
}

func (l *delegatingLogSink) Enabled(level int) bool {
	eventuallyFulfillRoot()
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.logger.Enabled(level)
}

func (l *delegatingLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	eventuallyFulfillRoot()
	l.lock.RLock()
	defer l.lock.RUnlock()
	l.logger.Info(level, msg, keysAndValues...)
}

func (l *delegatingLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	eventuallyFulfillRoot()
	l.lock.RLock()
	defer l.lock.RUnlock()
	l.logger.Error(err, msg, keysAndValues...)
}

func (l *delegatingLogSink) WithName(name string) logr.LogSink {
	eventuallyFulfillRoot()
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.promise == nil {
		sink := l.logger.WithName(name)
		if withCallDepth, ok := sink.(logr.CallDepthLogSink); ok {
			sink = withCallDepth.WithCallDepth(-1)
		}
		return sink
	}

	res := &delegatingLogSink{logger: l.logger}
	promise := l.promise.WithName(res, name)
	res.promise = promise

	return res
}

func (l *delegatingLogSink) WithValues(tags ...interface{}) logr.LogSink {
	eventuallyFulfillRoot()
	l.lock.RLock()
	defer l.lock.RUnlock()

	if l.promise == nil {
		sink := l.logger.WithValues(tags...)
		if withCallDepth, ok := sink.(logr.CallDepthLogSink); ok {
			sink = withCallDepth.WithCallDepth(-1)
		}
		return sink
	}

	res := &delegatingLogSink{logger: l.logger}
	promise := l.promise.WithValues(res, tags...)
	res.promise = promise

	return res
}

func (l *delegatingLogSink) Fulfill(actual logr.LogSink) {
	if actual == nil {
		actual = nullLogSink{}
	}
	if l.promise != nil {
		l.promise.Fulfill(actual)
	}
}

func newDelegatingLogSink(initial logr.LogSink) *delegatingLogSink {
	l := &delegatingLogSink{
		logger:  initial,
		promise: &loggerPromise{promisesLock: sync.Mutex{}},
	}
	l.promise.logger = l
	return l
}
