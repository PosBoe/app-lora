package core

import (
	"crypto/rand"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordChars = 8
	// maxPasswordBytes is the bcrypt input limit.
	maxPasswordBytes = 72
	// temporaryPasswordChars is the length of a generated temporary password.
	temporaryPasswordChars = 16
)

// temporaryAlphabet leaves out characters that look alike: 0/O, 1/l/I.
const temporaryAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var loginPattern = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

// passwords hashes and checks passwords with bcrypt.
type passwords struct {
	cost int
	// dummyHash is compared when a login is unknown, so an unknown login
	// takes as long as a wrong password.
	dummyHash []byte
}

func newPasswords() (*passwords, error) {
	return newPasswordsWithCost(bcrypt.DefaultCost)
}

// newPasswordsWithCost lets a test use bcrypt.MinCost.
func newPasswordsWithCost(cost int) (*passwords, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), cost)
	if err != nil {
		return nil, err
	}
	return &passwords{cost: cost, dummyHash: dummy}, nil
}

// hash checks the password rules and returns the bcrypt hash.
func (p *passwords) hash(password string) (string, error) {
	if utf8.RuneCountInString(password) < minPasswordChars {
		return "", errPasswordTooShort
	}
	if len(password) > maxPasswordBytes {
		return "", errPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// matches reports whether password matches hash. An empty hash compares the
// dummy hash and reports false.
func (p *passwords) matches(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(p.dummyHash, []byte(password))
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// newTemporaryPassword returns a random password over temporaryAlphabet.
func newTemporaryPassword() string {
	return randomString(temporaryPasswordChars, temporaryAlphabet)
}

func randomString(n int, alphabet string) string {
	// The largest multiple of len(alphabet) below 256 keeps the bytes
	// uniform: a byte above it is drawn again.
	limit := 256 - 256%len(alphabet)
	var b strings.Builder
	buf := make([]byte, 1)
	for b.Len() < n {
		_, _ = rand.Read(buf)
		if int(buf[0]) >= limit {
			continue
		}
		b.WriteByte(alphabet[int(buf[0])%len(alphabet)])
	}
	return b.String()
}

// normalizeLogin trims and lowers a login and checks its characters.
func normalizeLogin(login string) (string, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if !loginPattern.MatchString(login) {
		return "", errLoginInvalid
	}
	return login, nil
}
