package domain

import "fmt"

type Error struct {
	Code    string
	Message string
	State   DossierState
	Version uint64
}

func (e *Error) Error() string        { return fmt.Sprintf("%s: %s", e.Code, e.Message) }
func Invalid(code, msg string) *Error { return &Error{Code: code, Message: msg} }
func (d *SurveyDossier) err(code, msg string) error {
	return &Error{Code: code, Message: msg, State: d.State, Version: d.Version}
}
