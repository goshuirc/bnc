// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package bncDataStoreBuntdb

import (
	"crypto/rand"

	"golang.org/x/crypto/bcrypt"
)

const newSaltLen = 30
const defaultPasswordCost = 14

// NewSalt returns a salt for crypto uses.
func NewSalt() []byte {
	salt := make([]byte, newSaltLen)
	rand.Read(salt)
	return salt
}

// assemblePassword returns an assembled slice of bytes for the given password details.
func assemblePassword(bncSalt []byte, specialSalt []byte, password string) []byte {
	var assembledPasswordBytes []byte
	assembledPasswordBytes = append(assembledPasswordBytes, bncSalt...)
	assembledPasswordBytes = append(assembledPasswordBytes, '-')
	assembledPasswordBytes = append(assembledPasswordBytes, specialSalt...)
	assembledPasswordBytes = append(assembledPasswordBytes, '-')
	assembledPasswordBytes = append(assembledPasswordBytes, []byte(password)...)
	return assembledPasswordBytes
}

// GenerateFromPassword takes our salts and encrypts the given password.
func GenerateFromPassword(bncSalt []byte, specialSalt []byte, password string) ([]byte, error) {
	assembledPasswordBytes := assemblePassword(bncSalt, specialSalt, password)
	return bcrypt.GenerateFromPassword(assembledPasswordBytes, defaultPasswordCost)
}

// CompareHashAndPassword compares an ircbnc hashed password with its possible plaintext equivalent.
// Returns nil on success, or an error on failure.
func CompareHashAndPassword(hashedPassword []byte, bncSalt []byte, specialSalt []byte, password string) bool {
	assembledPasswordBytes := assemblePassword(bncSalt, specialSalt, password)
	err := bcrypt.CompareHashAndPassword(hashedPassword, assembledPasswordBytes)
	if err == nil {
		return true
	}

	return false
}
