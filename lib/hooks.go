package ircbnc

import (
	"github.com/goshuirc/irc-go/ircmsg"
)

type HookEmitter struct {
	Registered map[string][]func(interface{})
}

func MakeHookEmitter() HookEmitter {
	return HookEmitter{
		Registered: make(map[string][]func(interface{})),
	}
}

func (hooks *HookEmitter) Dispatch(hookName string, data interface{}) {
	callbacks, _ := hooks.Registered[hookName]
	for _, p := range callbacks {
		p(data)
	}
}

func (hooks *HookEmitter) Register(hookName string, p func(interface{})) {
	_, exists := hooks.Registered[hookName]
	if !exists {
		hooks.Registered[hookName] = make([]func(interface{}), 0)
	}

	hooks.Registered[hookName] = append(hooks.Registered[hookName], p)
}

/**
 * Hooks that are dispatched throughout the core.
 * Components and plugins will listen out for these hooks
 * to extend the core functionality.
 */

var HookIrcRawName = "irc.raw"

type HookIrcRaw struct {
	Listener   *Listener
	FromServer bool
	FromClient bool
	Raw        string
	Message    ircmsg.IrcMessage
	Halt       bool
}

var HookNewListenerName = "listener.new"

type HookNewListener struct {
	Listener *Listener
	Halt     bool
}

var HookListenerCloseName = "listener.close"

type HookListenerClose struct {
	Listener *Listener
}
