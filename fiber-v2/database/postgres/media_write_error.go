package postgres

import (
	"context"
	"fmt"
	"io"
)

type mediaWriteStage uint8

const (
	mediaWriteDependency mediaWriteStage = iota
	mediaWriteInvalidInput
	mediaWriteBegin
	mediaWriteInsert
	mediaWriteScan
	mediaWriteSelectedRow
	mediaWriteCommit
)

type mediaWriteError struct {
	stage      mediaWriteStage
	contextErr error // Only a standard context sentinel, never the backend cause.
}

func newMediaWriteError(stage mediaWriteStage, _ error, ctx context.Context) error {
	err := &mediaWriteError{stage: stage}
	switch {
	case ctx != nil && ctx.Err() == context.Canceled:
		err.contextErr = context.Canceled
	case ctx != nil && ctx.Err() == context.DeadlineExceeded:
		err.contextErr = context.DeadlineExceeded
	}
	return err
}

func (e *mediaWriteError) Error() string {
	return "media could not be inserted: " + e.stageText()
}

// Is preserves only context classification without exposing a backend chain.
func (e *mediaWriteError) Is(target error) bool {
	return (target == context.Canceled || target == context.DeadlineExceeded) && e.contextErr == target
}

// Format keeps every standard error rendering on the same safe text boundary.
func (e *mediaWriteError) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.Error())
}

func (e *mediaWriteError) stageText() string {
	switch e.stage {
	case mediaWriteDependency:
		return "dependency"
	case mediaWriteInvalidInput:
		return "invalid input"
	case mediaWriteBegin:
		return "begin"
	case mediaWriteInsert:
		return "insert"
	case mediaWriteScan:
		return "returning scan"
	case mediaWriteSelectedRow:
		return "selected row"
	case mediaWriteCommit:
		return "commit"
	default:
		return "failure"
	}
}
