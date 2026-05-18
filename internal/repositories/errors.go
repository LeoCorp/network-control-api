package repositories

import "errors"

var (
	ErrNotFound                = errors.New("record not found")
	ErrDuplicateEmail          = errors.New("email already exists")
	ErrDuplicateName           = errors.New("name already exists")
	ErrDuplicateActiveIncident = errors.New("active incident already exists for device")
)
