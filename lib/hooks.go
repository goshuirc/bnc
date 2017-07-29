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

var HookIrcClientRawName = "irc.client.raw"

type HookIrcClientRaw struct {
	Listener *Listener
	Raw      string
	Message  ircmsg.IrcMessage
	Halt     bool
}
