package spec

import (
	"errors"
	"strconv"
	"time"
)

type ExpiryDuration struct {
	Duration *time.Duration
}

var errInvalidExpiryFormat = errors.New("invalid expiry duration foramt")

func (d *ExpiryDuration) UnmarshalJSON(jsonValue []byte) error {
	val, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return errInvalidExpiryFormat
	}
	dur, err := time.ParseDuration(val)
	if err != nil {
		return errInvalidExpiryFormat
	}
	d.Duration = &dur
	return nil
}

func (input CreateJarInput) Validate(unlimitedExpiry bool) *Validator {
	v := NewValidator()
	v.Check(input.Expiry.Duration == nil || time.Duration(*input.Expiry.Duration) > time.Minute*5, "expiry", "expiry period must be greater than or equal to 5 minutes")
	v.Check(input.Access <= AccessPrivate, "access", "access type can be one of 0, 1")
	v.Check(input.Access == AccessPublic || len(input.Password) != 0, "password", "password can't be empty when access is private")
	v.Check(len(input.Scrolls) < 255, "scrolls", "no of scrolls can't be greater than 254")
	v.Check(AllFunc(input.Tags, func(tag string) bool {
		return len(tag) < 50
	}), "tags", "no tag can be of length greater than 50")

	DurYear := time.Hour * 25 * 365
	v.Check(unlimitedExpiry || input.Expiry.Duration == nil || *(input.Expiry.Duration) < DurYear, "expiry", "Duration of anonymouns jar must be less than a yaer")
	return v
}

func (input LoginInput) Validate() *Validator {
	v := NewValidator()
	v.Check(
		Matches(string(input.Email), EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)
	return v
}

func (input ActivationInput) Validate() *Validator {
	v := NewValidator()
	v.Check(
		len(input.Token) > 0,
		"token",
		"token must not be empty",
	)
	return v
}

func (input RegistrationInput) Validate() *Validator {
	v := NewValidator()
	v.Check(
		len(input.Username) > 0 && len(input.Username) <= 512,
		"username",
		"username must be withing 1-512 charcters",
	)
	v.Check(
		Matches(string(input.Email), EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)

	return v
}
