package log

import (
	"log"

	"github.com/goph/logur"
)

// NewErrorStandardLogger returns a new standard logger logging on error level.
func NewErrorStandardLogger(logger logur.Logger) *log.Logger {
	return logur.NewErrorStandardLogger(logger, "", 0)
}
