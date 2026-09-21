package postgres

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type readOperation uint8

const (
	readSiteOptions readOperation = iota
	readUploadPolicy
	readPasswordPolicy
	readMailDeliveryOptions
	readCaptchaVerificationOptions
	readOptionMediaReferences
	readOptionMediaMutationSnapshot
	readContactRequestWorkflowSnapshot
	readContactRequestResponseWorkflowSnapshot
	readUserStatus
)

type readStage uint8

const (
	readDependency readStage = iota
	readQuery
	readScan
	readRowsClose
	readCardinality
	readInvalidSelection
	readInvalidIdentifier
	readSelectedRow
)

type repositoryReadError struct {
	operation  readOperation
	stage      readStage
	contextErr error // Only a standard context sentinel, never the backend cause.
}

func newRepositoryReadError(operation readOperation, stage readStage, cause error) error {
	err := &repositoryReadError{operation: operation, stage: stage}
	switch {
	case errors.Is(cause, context.Canceled):
		err.contextErr = context.Canceled
	case errors.Is(cause, context.DeadlineExceeded):
		err.contextErr = context.DeadlineExceeded
	}
	return err
}

func (e *repositoryReadError) Error() string {
	return e.operationText() + " could not be read: " + e.stageText()
}

// Is preserves only context classification without exposing a backend chain.
func (e *repositoryReadError) Is(target error) bool {
	return (target == context.Canceled || target == context.DeadlineExceeded) && e.contextErr == target
}

// Format keeps every standard error rendering on the same safe text boundary.
func (e *repositoryReadError) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, e.Error())
}

func (e *repositoryReadError) operationText() string {
	switch e.operation {
	case readSiteOptions:
		return "site options"
	case readUploadPolicy:
		return "upload policy"
	case readPasswordPolicy:
		return "password policy"
	case readMailDeliveryOptions:
		return "mail delivery options"
	case readCaptchaVerificationOptions:
		return "captcha verification options"
	case readOptionMediaReferences:
		return "option media references"
	case readOptionMediaMutationSnapshot:
		return "option media mutation snapshot"
	case readContactRequestWorkflowSnapshot:
		return "contact request workflow snapshot"
	case readContactRequestResponseWorkflowSnapshot:
		return "contact request response workflow snapshot"
	case readUserStatus:
		return "user status"
	default:
		return "repository data"
	}
}

func (e *repositoryReadError) stageText() string {
	switch e.stage {
	case readDependency:
		return "dependency"
	case readQuery:
		return "query"
	case readScan:
		return "scan"
	case readRowsClose:
		return "rows/close"
	case readCardinality:
		return "cardinality"
	case readInvalidSelection:
		return "invalid selection"
	case readInvalidIdentifier:
		return "invalid identifier"
	case readSelectedRow:
		return "selected row"
	default:
		return "failure"
	}
}
