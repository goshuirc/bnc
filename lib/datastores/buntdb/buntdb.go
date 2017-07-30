package bncDataStoreBuntdb

import "github.com/goshuirc/bnc/lib"

type DataStoreBuntdb struct {
}

func (ds *DataStoreBuntdb) AddUser(id string, username string) (*ircbnc.User, error) {
	// TODO: Add the user to storage and return its User object
	return &ircbnc.User{}, nil
}

func (ds *DataStoreBuntdb) GetUserById(id string) *ircbnc.User {
	return &ircbnc.User{}
}

func (ds *DataStoreBuntdb) SaveUser(*ircbnc.User) error {
	return nil
}

func (ds *DataStoreBuntdb) AuthUser(username string, password string) *ircbnc.User {
	return ds.GetUserById(username)
}

func (ds *DataStoreBuntdb) SetUserPassword(userId string, newPassord string) {
	// TODO: Hash + store the new password
}

func (ds *DataStoreBuntdb) GetUserNetworks(userId string) {
	// TODO: Return a slice of network objects of some kind
}

func (ds *DataStoreBuntdb) AddUserNetwork(userId string, netName string) {
	// TODO: Return a new network object of some kind
}

func (ds *DataStoreBuntdb) SaveNetwork(userId string, netName string) {
	// TODO: Return a new network object of some kind
}
