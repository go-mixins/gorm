package gorm

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// UniqueViolation is true when database unique index is violated
func UniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

// NotFound is true when there was no appropriate record
func NotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// Deadlock is true when deadlock was detected
func Deadlock(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "deadlock")
}
