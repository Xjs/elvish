package cliedit

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/elves/elvish/cli"
	"github.com/elves/elvish/cli/clitypes"
	"github.com/elves/elvish/cli/cliutil"
	"github.com/elves/elvish/edit/eddefs"
	"github.com/elves/elvish/edit/ui"
	"github.com/elves/elvish/eval"
	"github.com/elves/elvish/eval/vals"
)

// TODO(xiaq): Move the implementation into this package.

// A specialized map type for key bindings.
type bindingMap = eddefs.BindingMap

// An empty binding map. It is useful for building binding maps.
var emptyBindingMap = eddefs.EmptyBindingMap

type mapBinding struct {
	ev   *eval.Evaler
	maps []*bindingMap
}

func newMapBinding(ev *eval.Evaler, maps ...*bindingMap) cli.Binding {
	return mapBinding{ev, maps}
}

func (b mapBinding) KeyHandler(k ui.Key) cli.KeyHandler {
	f := indexLayeredBindings(k, b.maps...)
	// TODO: Make this fallback part of GetOrDefault after moving BindingMap
	// into this package.
	if f == nil {
		return unbound
	}
	return func(ev cli.KeyEvent) {
		callBinding(ev.App(), b.ev, f, ev)
	}
}

func unbound(ev cli.KeyEvent) {
	ev.State().AddNote("Unbound: " + ev.Key().String())
}

// Indexes a series of layered bindings. Returns nil if none of the bindings
// have the required key or a default.
func indexLayeredBindings(k ui.Key, bindings ...*bindingMap) eval.Callable {
	for _, binding := range bindings {
		if binding.HasKey(k) {
			return binding.GetKey(k)
		}
	}
	for _, binding := range bindings {
		if binding.HasKey(ui.Default) {
			return binding.GetKey(ui.Default)
		}
	}
	return nil
}

var bindingSource = eval.NewInternalSource("[editor binding]")

// TODO: callBinding should pass KeyEvent to the function being called.

func callBinding(nt notifier, ev *eval.Evaler, f eval.Callable, event cli.KeyEvent) clitypes.HandlerAction {

	// TODO(xiaq): Use CallWithOutputCallback when it supports redirecting the
	// stderr port.
	notifyPort, cleanup := makeNotifyPort(nt.Notify)
	defer cleanup()
	ports := []*eval.Port{eval.DevNullClosedChan, notifyPort, notifyPort}
	frame := eval.NewTopFrame(ev, bindingSource, ports)

	err := frame.Call(f, []interface{}{event}, eval.NoOpts)

	if err != nil {
		if action, ok := eval.Cause(err).(cliutil.ActionError); ok {
			return clitypes.HandlerAction(action)
		}
		// TODO(xiaq): Make the stack trace available.
		nt.Notify("[binding error] " + err.Error())
	}
	return clitypes.NoAction
}

func makeNotifyPort(notify func(string)) (*eval.Port, func()) {
	ch := make(chan interface{})
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		// Relay value outputs
		for v := range ch {
			notify("[value out] " + vals.Repr(v, vals.NoPretty))
		}
		wg.Done()
	}()
	go func() {
		// Relay byte outputs
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if line != "" {
					notify("[bytes out] " + line)
				}
				if err != io.EOF {
					notify("[bytes error] " + err.Error())
				}
				break
			}
			notify("[bytes out] " + line[:len(line)-1])
		}
		wg.Done()
	}()
	port := &eval.Port{Chan: ch, File: w, CloseChan: true, CloseFile: true}
	cleanup := func() {
		port.Close()
		wg.Wait()
	}
	return port, cleanup
}
