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
