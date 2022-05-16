package mqtt

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/iotaledger/hive.go/basicauth"
)

// AuthAllowEveryone allows everyone, but without write permission.
type AuthAllowEveryone struct{}

// Authenticate returns true if a username and password are acceptable.
func (a *AuthAllowEveryone) Authenticate(user, password []byte) bool {
	return true
}

// ACL returns true if a user has access permissions to read or write on a topic.
func (a *AuthAllowEveryone) ACL(user []byte, topic string, write bool) bool {
	// clients are not allowed to write
	return !write
}

// AuthAllowBasicAuth allows users that authenticate with basic auth, but without write permission.
type AuthAllowBasicAuth struct {
	Salt  []byte
	Users map[string][]byte
}

func NewAuthAllowUsers(passwordSaltHex string, users map[string]string) (*AuthAllowBasicAuth, error) {

	if len(passwordSaltHex) != 64 {
		return nil, errors.New("password salt must be 64 (hex encoded) in length")
	}

	passwordSalt, err := hex.DecodeString(passwordSaltHex)
	if err != nil {
		return nil, fmt.Errorf("parsing password salt failed: %w", err)
	}

	usersWithHashedPasswords := make(map[string][]byte)
	for user, passwordHashHex := range users {
		if len(passwordHashHex) != 64 {
			return nil, fmt.Errorf("password hash for user %s must be 64 (hex encoded scrypt hash) in length", user)
		}

		password, err := hex.DecodeString(passwordHashHex)
		if err != nil {
			return nil, fmt.Errorf("parsing password hash for user %s failed: %w", user, err)
		}

		usersWithHashedPasswords[user] = password
	}

	return &AuthAllowBasicAuth{
		Users: usersWithHashedPasswords,
		Salt:  passwordSalt,
	}, nil
}

// Authenticate returns true if a username and password are acceptable.
func (a *AuthAllowBasicAuth) Authenticate(user, password []byte) bool {
	// If the user exists in the auth users map, and the password is correct,
	// then they can connect to the server.
	if hashedPassword, exists := a.Users[string(user)]; exists {

		// error is ignored because it returns false in case it can't be derived
		if valid, _ := basicauth.VerifyPassword(password, a.Salt, hashedPassword); valid {
			return true
		}
	}

	return false
}

// ACL returns true if a user has access permissions to read or write on a topic.
func (a *AuthAllowBasicAuth) ACL(user []byte, topic string, write bool) bool {
	// clients are not allowed to write
	return !write
}
