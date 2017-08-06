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

	dbPath, _ := manager.Config.Bouncer.Storage["database"]
	if dbPath == "" {
		return errors.New("No database file has been configured")
	}

	db, err := buntdb.Open(dbPath)
	if err != nil {
		return errors.New("Could not open database: " + err.Error())
	}

	ds.Db = db

	err = ds.LoadSalt()
	if err != nil {
		return errors.New("Could not initialize database: " + err.Error())
	}
	return nil
}

func (ds *DataStore) Setup() error {
	// generate bouncer salt
	bncSalt := NewSalt()
	encodedBncSalt := base64.StdEncoding.EncodeToString(bncSalt)
	err := ds.Db.Update(func(tx *buntdb.Tx) error {
		tx.Set(KeySalt, encodedBncSalt, nil)
		return nil
	})

	ds.salt = bncSalt

	return err
}

func (ds *DataStore) LoadSalt() error {
	err := ds.Db.View(func(tx *buntdb.Tx) error {
		saltString, err := tx.Get(KeySalt)
		if err != nil && err.Error() == "not found" {
			return nil
		} else if err != nil {
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
	println("Getting all users...")
	ds.Db.View(func(tx *buntdb.Tx) error {
		tx.DescendKeys("user.info *", func(key, value string) bool {
			println(key)
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

func (ds *DataStore) GetUserById(id string) *ircbnc.User {
	var user *ircbnc.User
	ds.Db.View(func(tx *buntdb.Tx) error {
		user, _ = ds.loadUser(tx, id)
		return nil
	})

	return user
}

func (ds *DataStore) SaveUser(user *ircbnc.User) error {
	// TODO: If ID isn't set, set the ID now.
	// An ID set = the user object was saved or retrieved from the db
	ui := UserInfo{}
	ui.ID = user.ID
	ui.Role = user.Role
	ui.EncodedSalt = base64.StdEncoding.EncodeToString(user.Salt)
	ui.EncodedPasswordHash = base64.StdEncoding.EncodeToString(user.HashedPassword)
	ui.DefaultNick = user.DefaultNick
	ui.DefaultNickFallback = user.DefaultFbNick
	ui.DefaultUsername = user.DefaultUser
	ui.DefaultRealname = user.DefaultReal

	uiBytes, err := json.Marshal(ui)
	if err != nil {
		return fmt.Errorf("Error marshalling user info: %s", err.Error())
	}
	uiString := string(uiBytes) //TODO(dan): Should we do this in a safer way?

	// User permissions
	upBytes, err := json.Marshal(user.Permissions)
	if err != nil {
		return fmt.Errorf("Error marshalling user permissions: %s", err.Error())
	}
	upString := string(upBytes) //TODO(dan): Should we do this in a safer way?

	updateErr := ds.Db.Update(func(tx *buntdb.Tx) error {
		var err error
		_, _, err = tx.Set(fmt.Sprintf(KeyUserInfo, ui.ID), uiString, nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf(KeyUserPermissions, ui.ID), upString, nil)
		if err != nil {
			return err
		}
		return nil
	})

	return updateErr
}

func (ds *DataStore) AuthUser(username string, password string) (string, bool) {
	user := ds.GetUserById(username)
	if user == nil {
		return "", false
	}

	passMatches := CompareHashAndPassword(user.HashedPassword, ds.salt, user.Salt, password)
	if !passMatches {
		return "", false
	}

	return username, true
}

func (ds *DataStore) SetUserPassword(user *ircbnc.User, newPassword string) {
	userSalt := NewSalt()
	passHash, _ := GenerateFromPassword(ds.salt, userSalt, newPassword)

	user.Salt = userSalt
	user.HashedPassword = passHash
}

func (ds *DataStore) GetUserNetworks(userId string) {
	// TODO: Return a slice of network objects of some kind
}

func (ds *DataStore) SaveConnection(connection *ircbnc.ServerConnection) error {
	// Store server info
	sc := ServerConnectionMapping{
		Name:             connection.Name,
		Enabled:          true,
		ConnectPassword:  connection.Password,
		Nickname:         connection.Nickname,
		NicknameFallback: connection.FbNickname,
		Username:         connection.Username,
		Realname:         connection.Realname,
	}
	scBytes, err := json.Marshal(sc)
	if err != nil {
		return fmt.Errorf("Error marshalling user info: %s", err.Error())
	}
	scString := string(scBytes) //TODO(dan): Should we do this in a safer way?

	// Store server addresses
	saBytes, err := json.Marshal(connection.Addresses)
	if err != nil {
		return fmt.Errorf("Error marshalling user permissions: %s", err.Error())
	}
	saString := string(saBytes) //TODO(dan): Should we do this in a safer way?

	// Store server channels (Convert the string map to a slice)
	scChannels := []ircbnc.ServerConnectionChannel{}
	for _, channel := range connection.Channels {
		scChannels = append(scChannels, channel)
	}
	scChanBytes, err := json.Marshal(scChannels)
	if err != nil {
		return fmt.Errorf("Error marshalling user permissions: %s", err.Error())
	}
	scChanString := string(scChanBytes) //TODO(dan): Should we do this in a safer way?

	saveErr := ds.Db.Update(func(tx *buntdb.Tx) error {
		var err error
		_, _, err = tx.Set(fmt.Sprintf(KeyServerConnectionInfo, connection.User.ID, connection.Name), scString, nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf(KeyServerConnectionAddresses, connection.User.ID, connection.Name), saString, nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf(KeyServerConnectionChannels, connection.User.ID, connection.Name), scChanString, nil)
		if err != nil {
			return err
		}
		return nil
	})

	return saveErr
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
	scInfo := &ServerConnectionMapping{}
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

	scChans := &[]ServerConnectionChannelMapping{}
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

	scAddresses := &[]ServerConnectionAddressMapping{}
	err = json.Unmarshal([]byte(scAddressesString), scAddresses)
	if err != nil {
		return nil, fmt.Errorf("Could not create new ServerConnection (unmarshalling sc addresses): %s", err.Error())
	}

	// check port number and add addresses
	for _, address := range *scAddresses {
		if address.Port < 1 || address.Port > 65535 {
			return nil, fmt.Errorf("Could not create new ServerConnection (port %d is not valid)", address.Port)
		}

		sc.Addresses = append(sc.Addresses, ircbnc.ServerConnectionAddress(address))
	}

	return sc, nil
}
