// This file is based on 'config.go' from Oragono/Ergonomadic
// it is modified by Daniel Oaks <daniel@danieloaks.net>
// covered by the MIT license in the LICENSE.ergonomadic file

package ircbnc

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// TLSListenConfig defines configuration options for listening on TLS
type TLSListenConfig struct {
	Cert string
	Key  string
}

// Config returns the TLS certificate assicated with this TLSListenConfig
func (conf *TLSListenConfig) Config() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		return nil, errors.New("tls cert+key: invalid pair")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, err
}

// Config defines a configuration file for GoshuBNC
type Config struct {
	Bouncer struct {
		DatabasePath string `yaml:"database-path"`
		Listeners    []string
		TLSListeners map[string]*TLSListenConfig `yaml:"tls-listeners"`
	}
}

// TLSListeners returns a map of tls.Config objects from our config
func (conf *Config) TLSListeners() map[string]*tls.Config {
	tlsListeners := make(map[string]*tls.Config)
	for s, tlsListenersConf := range conf.Bouncer.TLSListeners {
		config, err := tlsListenersConf.Config()
		if err != nil {
			log.Fatal(err)
		}
		tlsListeners[s] = config
	}
	return tlsListeners
}

// LoadConfig returns a Config instance
func LoadConfig(filename string) (config *Config, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	if config.Bouncer.DatabasePath == "" {
		return nil, errors.New("Database path is missing")
	}
	if len(config.Bouncer.Listeners) == 0 {
		return nil, errors.New("No listeners are defined")
	}
	return config, nil
}
