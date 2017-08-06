package ircbnc

type DataStoreInterface interface {
	Init(manager *Manager) error
	Setup() error
	GetAllUsers() []*User
	GetUserById(id string) *User
	SaveUser(*User) error
	SetUserPassword(user *User, newPassword string)
	AuthUser(username string, password string) (*User, error)
	GetUserNetworks(userId string)
	SaveConnection(connection *ServerConnection) error
}
