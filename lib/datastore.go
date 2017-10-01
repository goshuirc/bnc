// Copyright (c) 2017 Darren Whitlen <darren@kiwiirc.com>
// released under the MIT license

package ircbnc

type DataStoreInterface interface {
	Init(manager *Manager) error
	Setup() error
	GetAllUsers() []*User
	GetUserById(id string) *User
	GetUserByUsername(username string) *User
	SaveUser(*User) error
	SetUserPassword(user *User, newPassword string)
	AuthUser(username string, password string) (authedUserId string, authSuccess bool)
	GetUserNetworks(userId string)
	SaveConnection(connection *ServerConnection) error
	DelConnection(connection *ServerConnection) error
}
