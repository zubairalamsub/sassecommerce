package service

import (
	"errors"
	"strings"
)

// MinPasswordLength is enforced both here (service layer) and in the request
// model bindings so the policy holds even for callers that bypass the HTTP
// handlers.
const MinPasswordLength = 12

// commonPasswords is a denylist of passwords that satisfy the length rule but
// are still trivially guessable (dictionary words with digit suffixes,
// keyboard walks, repeated patterns). Matched case-insensitively.
var commonPasswords = map[string]struct{}{
	"password1234":      {},
	"password12345":     {},
	"password123456":    {},
	"passw0rd1234":      {},
	"123456789012":      {},
	"1234567890123":     {},
	"12345678901234":    {},
	"qwertyuiop12":      {},
	"qwertyuiopas":      {},
	"qwertyuiopasdf":    {},
	"qwerty123456":      {},
	"1q2w3e4r5t6y":      {},
	"administrator":     {},
	"adminadmin123":     {},
	"welcome12345":      {},
	"iloveyou1234":      {},
	"letmeinplease":     {},
	"changeme12345":     {},
	"defaultpassword":   {},
	"corporatepassword": {},
	"summer2024!!":      {},
	"password!2345":     {},
	"abc123abc123":      {},
	"111111111111":      {},
	"000000000000":      {},
	"aaaaaaaaaaaa":      {},
}

var (
	errPasswordTooShort = errors.New("password must be at least 12 characters")
	errPasswordCommon   = errors.New("password is too common — choose a less guessable one")
)

// validatePasswordPolicy rejects passwords that are too short or on the
// common-password denylist. Applied on registration and password
// change/reset; existing hashes are unaffected so current users can still
// log in.
func validatePasswordPolicy(password string) error {
	if len(password) < MinPasswordLength {
		return errPasswordTooShort
	}
	if _, common := commonPasswords[strings.ToLower(password)]; common {
		return errPasswordCommon
	}
	return nil
}
