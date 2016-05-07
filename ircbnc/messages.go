// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

// Message represents an internal message passed by ircbnc.
type Message struct {
	Type MessageType
	Verb MessageVerb
	Info map[MessageInfoKey]interface{}
}

// MessageType represents the type of message it is.
type MessageType int

const (
	// LineMT represents an IRC line Message Type
	LineMT MessageType = iota
)

// MessageVerb represents the verb (i.e. the specific command, etc) of a message.
type MessageVerb int

const (
	// NoMV represents no Message Verb.
	NoMV MessageVerb = iota
)

// MessageInfoKey represents a key in the Info attribute of a Message.
type MessageInfoKey int

const (
	// LineIK represents an IRC line message info key
	LineIK MessageInfoKey = iota
)
