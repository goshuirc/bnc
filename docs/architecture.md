# GoshuBNC Architecture

This file roughly lays out the architecture of GoshuBNC and why certain decisions have been made.


## Manager

The Manager, quite simply, manages the different components, holds references to the database and keeps track of the users. Upon starting GoshuBNC, the Manager loads all the user information, creates the relevant ServerConnections and has them connect to the given addresses. It also creates the listening sockets, and makes Listeners for the incoming client connections that get established.


## Components

There are two primary components in GoshuBNC – Server Connections and Listeners.

- **Server Connections** are connections that get established to actual IRC servers. In other words, a ServerConnection connects to an IRC server.
- **Listeners** attach to and handle incoming client connections.

Basically, if an IRC client is connecting to GoshuIRC, they're talking to a Listener. If GoshuIRC is connecting out to an IRC server, it's using a ServerConnection.

### Server Connections

A ServerConnection is created for a specific user's network connection. Every user can have any number of networks. Every user+network can have one ServerConnection, that connects to one of the addresses the user supplies for that network.

### Listeners

A listener is created for every IRC client that connects in to GoshuBNC. Every listener belongs to a user. Every listener can 'listen' to one user+network (and gets events forwarded to it from the relevant ServerConnection in use at that time for the given user+network).
