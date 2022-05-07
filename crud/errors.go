package crud

import "errors"

// Errors returned by CRUD
var (
	ErrNotFound         = errors.New("not found")
	ErrFound            = errors.New("found")
	ErrUpdateNotApplied = errors.New("update not applied")
)
