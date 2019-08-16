package gorm

import (
	"strings"
)

// UniqueViolation is true when database unique index is violated
func UniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// NotFound is true when there was no appropriate record
func NotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found")
}

// Deadlock is true when deadlock was detected
func Deadlock(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "deadlock")
}
