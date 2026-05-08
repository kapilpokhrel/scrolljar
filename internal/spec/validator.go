package spec

import (
	"regexp"
	"slices"
	"strings"
)

var EmailReg = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Validator ValidationError

func NewValidator() *Validator {
	return &Validator{Errors: make([]FieldError, 0)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(e FieldError) {
	v.Errors = append(v.Errors, e)
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(FieldError{
			Field: strings.Split(key, "."),
			Msg:   message,
		})
	}
}

func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func AllFunc[E any](s []E, fn func(E) bool) bool {
	for _, v := range s {
		if !fn(v) {
			return false
		}
	}
	return true
}
