package bncDataStoreBuntdb

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/tidwall/buntdb"
)

type DataStore struct {
	ircbnc.DataStoreInterface
	Db      *buntdb.DB
	Manager *ircbnc.Manager
	salt    []byte
}

func (ds *DataStore) Init(manager *ircbnc.Manager) error {
	ds.Manager = manager

	var err error

	db, err := buntdb.Open(manager.Config.Bouncer.DatabasePath)
	if err != nil {
		return errors.New("Could not open DB:" + err.Error())
	}

	ds.Db = db

	err = ds.LoadSalt()
	if err != nil {
		return errors.New("Could not init DB:" + err.Error())
	}
	return nil
}

func (ds *DataStore) LoadSalt() error {
	err := ds.Db.View(func(tx *buntdb.Tx) error {
		saltString, err := tx.Get(KeySalt)
		if err != nil {
			return fmt.Errorf("Could not get salt string: %s", err.Error())
		}

		ds.salt, err = base64.StdEncoding.DecodeString(saltString)
		if err != nil {
			return fmt.Errorf("Could not decode b64'd salt: %s", err.Error())
		}

		return nil
	})

	return err
}

func (ds *DataStore) GetAllUsers() []*ircbnc.User {
	userIds := []string{}
	users := []*ircbnc.User{}

	ds.Db.View(func(tx *buntdb.Tx) error {
		tx.DescendKeys("user.info  *", func(key, value string) bool {
			userIds = append(userIds, strings.TrimPrefix(key, "user.info "))
			return true
		})

		// Iterate through the user IDs and generate our user objects
		for _, userId := range userIds {
			user, err := ds.loadUser(tx, userId)
			if err != nil {
				log.Println("Error loading user " + userId)
				continue
			}
			users = append(users, user)
		}

		return nil
	})

	return users
}

func (ds *DataStore) AddUser(id string, username string) (*ircbnc.User, error) {
	// TODO: Add the user to storage and return its User object
	return &ircbnc.User{}, nil
}

func (ds *DataStore) GetUserById(id string) *ircbnc.User {
	return &ircbnc.User{}
}

func (ds *DataStore) SaveUser(*ircbnc.User) error {
	return nil
}

func (ds *DataStore) AuthUser(username string, password string) (*ircbnc.User, error) {
	// Todo: actually password checking
	return ds.GetUserById(username), nil
}

func (ds *DataStore) SetUserPassword(userId string, newPassord string) {
	// TODO: Hash + store the new password
}

func (ds *DataStore) GetUserNetworks(userId string) {
	// TODO: Return a slice of network objects of some kind
}

func (ds *DataStore) AddUserNetwork(userId string, netName string) {
	// TODO: Return a new network object of some kind
}

func (ds *DataStore) SaveNetwork(userId string, netName string) {
	// TODO: Return a new network object of some kind
}

func (ds *DataStore) loadUser(tx *buntdb.Tx, userId string) (*ircbnc.User, error) {
	user := ircbnc.NewUser(ds.Manager)

	user.ID = userId
	user.Name = userId //TODO(dan): Store Name and ID separately in the future if we want to

	infoString, err := tx.Get(fmt.Sprintf(KeyUserInfo, userId))
	if err != nil {
		return nil, fmt.Errorf("Could not load user (loading user info from db): %s", err.Error())
	}
	ui := &UserInfo{}
	err = json.Unmarshal([]byte(infoString), ui)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (unmarshalling user info from db): %s", err.Error())
	}

	user.Salt, err = base64.StdEncoding.DecodeString(ui.EncodedSalt)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (decoding salt): %s", err.Error())
	}

	//TODO(dan): Make the below both have the same named fields
	user.HashedPassword, err = base64.StdEncoding.DecodeString(ui.EncodedPasswordHash)
	if err != nil {
		return nil, fmt.Errorf("Could not load user (decoding password): %s", err.Error())
	}
	user.DefaultNick = ui.DefaultNick
	user.DefaultFbNick = ui.DefaultNickFallback
	user.DefaultUser = ui.DefaultUsername
	user.DefaultReal = ui.DefaultRealname

	ds.loadUserConnections(user)

	return user, nil
}

func (ds *DataStore) loadUserConnections(user *ircbnc.User) {
	ds.Db.View(func(tx *buntdb.Tx) error {
		tx.DescendKeys(fmt.Sprintf("user.server.info %s *", user.ID), func(key, value string) bool {
			name := strings.TrimPrefix(key, fmt.Sprintf("user.server.info %s ", user.ID))

			sc, err := loadServerConnection(name, user, tx)
			if err != nil {
				log.Printf("Could not load user network: %s", err.Error())
				return false
			}

			user.Networks[name] = sc

			return true
		})

		return nil
	})
}

// LoadServerConnection loads the given server connection from our database.
func loadServerConnection(name string, user *ircbnc.User, tx *buntdb.Tx) (*ircbnc.ServerConnection, error) {
	sc := ircbnc.NewServerConnection()
	sc.Name = name
	sc.User = user

	// load general info
	scInfo := &ServerConnectionInfo{}
	scInfoString, err := tx.Get(fmt.Sprintf(KeyServerConnectionInfo, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc details from db): %s", err.Error())
	}

	err = json.Unmarshal([]byte(scInfoString), scInfo)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc details): %s", err.Error())
	}

	sc.Nickname = scInfo.Nickname
	sc.FbNickname = scInfo.NicknameFallback
	sc.Username = scInfo.Username
	sc.Realname = scInfo.Realname
	sc.Password = scInfo.ConnectPassword

	// set default values
	if sc.Nickname == "" {
		sc.Nickname = user.DefaultNick
	}
	if sc.FbNickname == "" {
		sc.FbNickname = user.DefaultFbNick
	}
	if sc.Username == "" {
		sc.Username = user.DefaultUser
	}
	if sc.Realname == "" {
		sc.Realname = user.DefaultReal
	}

	// load channels
	scChannelString, err := tx.Get(fmt.Sprintf(KeyServerConnectionChannels, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc channels from db): %s", err.Error())
	}

	scChans := &ircbnc.ServerConnectionChannels{}
	err = json.Unmarshal([]byte(scChannelString), scChans)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc channels): %s", err.Error())
	}

	sc.Channels = make(map[string]ircbnc.ServerConnectionChannel)
	for _, channel := range *scChans {
		sc.Channels[channel.Name] = ircbnc.ServerConnectionChannel{
			Name:   channel.Name,
			Key:    channel.Key,
			UseKey: channel.UseKey,
		}

	}

	// load addresses
	scAddressesString, err := tx.Get(fmt.Sprintf(KeyServerConnectionAddresses, user.ID, name))
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (getting sc addresses from db): %s", err.Error())
	}

	scAddresses := &ircbnc.ServerConnectionAddresses{}
	err = json.Unmarshal([]byte(scAddressesString), scAddresses)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc addresses): %s", err.Error())
	}

	// check port number and add addresses
	for _, address := range *scAddresses {
		if address.Port < 1 || address.Port > 65535 {
			return nil, fmt.Errorf("Could not create new ServerConnection (port %d is not valid)", address.Port)
		}

		sc.Addresses = append(sc.Addresses, address)
	}

	return sc, nil
}
