package rungroup_test

import (
	"action-monitor/pkg/tools/rungroup"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestZero(t *testing.T) {
	var g rungroup.Group
	res := make(chan error)
	go func() { res <- g.Run() }()
	select {
	case err := <-res:
		if err != nil {
			t.Errorf("%v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestOne(t *testing.T) {
	myError := errors.New("foobar")
	var g rungroup.Group
	g.Add(func() error { return myError }, func(error) {})
	res := make(chan error)
	go func() { res <- g.Run() }()
	select {
	case err := <-res:

		if want, have := myError, err; !strings.Contains(have.Error(), want.Error()) {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}

func TestMany(t *testing.T) {
	interrupt := errors.New("interrupt")
	var g rungroup.Group
	g.Add(func() error { return interrupt }, func(error) {})
	cancel := make(chan struct{})
	g.Add(func() error { <-cancel; return nil }, func(error) { close(cancel) })
	res := make(chan error)
	go func() { res <- g.Run() }()
	select {
	case err := <-res:
		if want, have := interrupt, err; !strings.Contains(have.Error(), want.Error()) {
			t.Errorf("want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("timeout")
	}
}

func TestHalted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	g := rungroup.NewNamedGroup(ctx, "test")

	long := func(ctx context.Context) error { time.Sleep(1100 * time.Millisecond); return nil }
	short := func(ctx context.Context) error { return nil }

	g.AddWithContextNamed("Long", long)
	g.AddWithContextNamed("Short", short)

	go cancel()
	_ = g.Run()
}
