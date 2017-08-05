package ircbnc

type DataStoreInterface interface {
	Init(*Manager) error
	AddUser(id string, username string) (*User, error)
	GetAllUsers() []*User
	GetUserById(id string) *User
	SaveUser(*User) error
	AuthUser(username string, password string) (*User, error)
	SetUserPassword(userId string, newPassord string)
	GetUserNetworks(userId string)
	AddUserNetwork(userId string, netName string)
	SaveNetwork(userId string, netName string)
}
