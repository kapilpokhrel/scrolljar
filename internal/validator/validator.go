// Package validator is responsible for validating the user json request
package validator

import (
	"regexp"
	"slices"
	"strings"

	"github.com/kapilpokhrel/scrolljar/internal/api/spec"
)

var EmailReg = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Validator spec.ValidationError

func New() *Validator {
	return &Validator{Errors: make([]spec.FieldError, 0)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(e spec.FieldError) {
	v.Errors = append(v.Errors, e)
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(spec.FieldError{
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
