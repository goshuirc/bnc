// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import (
	"fmt"

	"github.com/DanielOaks/girc-go/ircmsg"
)

// nickHandler handles the NICK command.
func nickHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	fmt.Println("NICK HANDLER")
	return true
}
