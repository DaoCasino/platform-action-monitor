// Package run implements an actor-runner with deterministic teardown. It is
// somewhat similar to package errgroup, except it does not require actor
// goroutines to understand context semantics. This makes it suitable for use in
// more circumstances; for example, goroutines which are handling connections
// from net.Listeners, or scanning input from a closable io.Reader.
package rungroup

import (
	"action-monitor/pkg/tools/signals"
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

func NewNamedGroup(ctx context.Context, name string) *Group {
	group := &Group{name: name}
	rootCtx, rootCtxCancel := context.WithCancel(ctx)
	group.AddNamed(
		"root-context-cancel",
		func() error { <-rootCtx.Done(); return errors.New("Root context cancelled") },
		func(error) { rootCtxCancel() },
	)

	signalCtx, signalCancel := context.WithCancel(context.Background())
	group.AddNamed(
		"signal_catcher",
		func() error { return signals.CatchStopSignalSimple(signalCtx) },
		func(err error) { signalCancel() },
	)

	return group
}

// Group collects actors (functions) and runs them concurrently.
// When one actor (function) returns, all actors are interrupted.
// The zero value of a Group is useful.
type Group struct {
	name   string
	actors []*actor
	lock   sync.RWMutex
}

func wrapError(err error, name string, caller string) error {
	return fmt.Errorf("error in group %s : %v. Caller: %s", name, err, caller)
}

// Add an actor (function) to the group. Each actor must be pre-emptable by an
// interrupt function. That is, if interrupt is invoked, execute should return.
// Also, it must be safe to call interrupt even after execute has returned.
//
// The first actor (function) to return interrupts all running actors.
// The error is passed to the interrupt functions, and is returned by Run.
func (g *Group) AddNamed(name string, execute func() error, interrupt func(error)) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.actors = append(g.actors, &actor{name, execute, interrupt, false})
}

// Deprecated: use AddNamed
func (g *Group) Add(execute func() error, interrupt func(error)) {
	pc, fn, line, _ := runtime.Caller(1)
	stasck := debug.Stack()
	caller := fmt.Sprintf("%s[%s:%d]\n%s", runtime.FuncForPC(pc).Name(), fn, line, string(stasck))
	_ = caller
	fnline := fmt.Sprintf("%s:%d", fn, line)
	g.AddNamed(fnline, execute, interrupt)
}

// Add an actor (function) that use context
func (g *Group) AddWithContextNamed(name string, execute func(ctx context.Context) error) {
	localCtx, localCtxCancel := context.WithCancel(context.Background())
	g.AddNamed(name, func() error { return execute(localCtx) }, func(error) { localCtxCancel() })
}

// Run all actors (functions) concurrently.
// When the first actor returns, all others are interrupted.
// Run only returns when all actors have exited.
// Run returns the error returned by the first exiting actor.
func (g *Group) Run() error {
	if len(g.actors) == 0 {
		return nil
	}

	// Run each actor.
	errors := make(chan error, len(g.actors))

	g.lock.Lock()
	for _, a := range g.actors {
		go func(a *actor) {
			err := a.execute()

			g.lock.Lock()
			a.finished = true
			g.lock.Unlock()

			errors <- wrapError(err, g.name, a.caller)
		}(a)
	}
	g.lock.Unlock()

	// Wait for the first actor to stop.
	err := <-errors

	logrus.Infof("Rungroup %s first error %v", g.name, err)

	// watch to help with halted actors on shutdown
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	defer watcherCancel()

	watcher := time.NewTicker(time.Second)
	defer watcher.Stop()

	go func() {
		for {
			select {
			case <-watcherCtx.Done():
				return
			case <-watcher.C:
				g.lock.RLock()
				for _, actor := range g.actors {
					if !actor.finished {
						logrus.Errorf("Waiting for finish group %s. Actor %s", g.name, actor.caller)
					}
				}
				g.lock.RUnlock()
			}
		}
	}()

	// Signal all actors to stop.
	g.lock.RLock()
	for _, a := range g.actors {
		a.interrupt(err)
	}
	g.lock.RUnlock()

	// Wait for all actors to stop.
	for i := 1; i < cap(errors); i++ {
		<-errors
	}

	// Return the original error.
	return err
}

type actor struct {
	caller    string
	execute   func() error
	interrupt func(error)
	finished  bool
}
