// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2014-2015 Edmund Huber
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

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
		Storage      map[string]string
		Listeners    []string
		TLSListeners map[string]*TLSListenConfig `yaml:"tls-listeners"`
		Logging      map[string]map[string]string
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

	if len(config.Bouncer.Listeners) == 0 {
		return nil, errors.New("No listeners are defined")
	}
	return config, nil
}
