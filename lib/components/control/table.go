// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package bncComponentControl

import (
	"bytes"
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/olekukonko/tablewriter"
)

/**
 * Table writer outputs to a Writer interface but we need a string.
 * This simply wraps the tablewriter.Table interface with a buffer
 * and adds a few helper functions
 */

type Table struct {
	tablewriter.Table
	Out *bytes.Buffer
}

func NewTable() *Table {
	table := &Table{
		Out: new(bytes.Buffer),
	}
	table.Table = *tablewriter.NewWriter(table.Out)
	return table
}
func (table *Table) RenderToString() string {
	table.Render()
	return table.Out.String()
}
func (table *Table) RenderToListener(listener *ircbnc.Listener, prefix string, command string) {
	out := table.RenderToString()
	out = strings.Trim(out, "\n")
	for _, line := range strings.Split(out, "\n") {
		listener.Send(nil, prefix, command, listener.ClientNick, line)
	}
}
