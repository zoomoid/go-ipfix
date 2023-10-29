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
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// This is taken from Kubernetes' controller-runtime/log package, except for not exposing
// any types, which appears unnecessary, but the implementation of delegated logging is
// kinda neat.
func SetLogger(l logr.Logger) {
	loggerSetLock.Lock()
	defer loggerSetLock.Unlock()

	loggerWasSet = true
	dlog.Fulfill(l.GetSink())
}

func fromContext(ctx context.Context, keysAndValues ...interface{}) logr.Logger {
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

func init() {
	go func() {
		time.Sleep(30 * time.Second)
		loggerSetLock.Lock()
		defer loggerSetLock.Unlock()

		if !loggerWasSet {
			dlog.Fulfill(nullLogSink{})
		}
	}()
}

var (
	loggerSetLock sync.Mutex
	loggerWasSet  bool

	dlog = newDelegatingLogSink(nullLogSink{})
	Log  = logr.New(dlog)
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

// Init implements logr.LogSink.
func (l *delegatingLogSink) Init(info logr.RuntimeInfo) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.info = info
}

// Enabled tests whether this Logger is enabled.  For example, commandline
// flags might be used to set the logging verbosity and disable some info
// logs.
func (l *delegatingLogSink) Enabled(level int) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.logger.Enabled(level)
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to
// the log line.  The key/value pairs can then be used to add additional
// variable information.  The key/value pairs should alternate string
// keys and arbitrary values.
func (l *delegatingLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	l.logger.Info(level, msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as context.
// It functions similarly to calling Info with the "error" named value, but may
// have unique behavior, and should be preferred for logging errors (see the
// package documentations for more information).
//
// The msg field should be used to add context to any underlying error,
// while the err field should be used to attach the actual error that
// triggered this log line, if present.
func (l *delegatingLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	l.logger.Error(err, msg, keysAndValues...)
}

// WithName provides a new Logger with the name appended.
func (l *delegatingLogSink) WithName(name string) logr.LogSink {
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
