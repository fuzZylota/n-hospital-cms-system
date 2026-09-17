package users

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	errUserEditForbidden   = errors.New("user edit is not permitted")
	errUserEditActorLookup = errors.New("user edit actor lookup failed")
	errInvalidUserEditUID  = errors.New("invalid user edit uid")
	errInvalidUserProfile  = errors.New("invalid user profile")
	userEmailPattern       = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

type userEditActor struct {
	UID      string
	Role     string
	IsActive bool
}

type userEditActorLookup func(uid string) (userEditActor, error)

type userEditPolicy struct {
	IsAdmin           bool
	UpdatePermissions bool
}

type userEditDecision struct {
	Policy    userEditPolicy
	TargetUID int
	Fields    map[string][]string
}

type branchPermissionChange struct {
	TargetUID int
	BranchID  int
	CanView   *bool
	CanDelete *bool
}

func isSelfProfileField(field string) bool {
	switch field {
	case "name", "surname", "email", "phone", "timezone":
		return true
	default:
		return false
	}
}

func isUserEditProtocolField(field string) bool {
	return field == "uid"
}

func normalizeUserEditFields(fields map[string][]string) (map[string][]string, error) {
	normalized := make(map[string][]string, len(fields))
	for field, values := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field == "" {
			return nil, fmt.Errorf("%w: empty field name", errUserEditForbidden)
		}
		if _, exists := normalized[field]; exists {
			return nil, fmt.Errorf("%w: duplicate field %s", errUserEditForbidden, field)
		}
		normalized[field] = append([]string(nil), values...)
	}
	return normalized, nil
}

func submittedFieldValue(fields map[string][]string, field string) (string, bool) {
	values, submitted := fields[field]
	if !submitted || len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func isValidPersonName(value string) bool {
	trimmedValue := strings.TrimSpace(value)
	runeCount := utf8.RuneCountInString(trimmedValue)
	if runeCount < 2 || runeCount > 255 {
		return false
	}

	letterCount := 0
	previousWasSeparator := false
	for index, character := range []rune(trimmedValue) {
		if unicode.IsLetter(character) {
			letterCount++
			previousWasSeparator = false
			continue
		}

		isSeparator := character == ' ' || character == '\'' || character == '’' || character == '-'
		if !isSeparator || index == 0 || index == runeCount-1 || previousWasSeparator {
			return false
		}
		previousWasSeparator = true
	}

	return letterCount >= 2
}

func isValidPhone(value string) bool {
	trimmedValue := strings.TrimSpace(value)
	digitCount := 0
	for index, character := range trimmedValue {
		switch {
		case character >= '0' && character <= '9':
			digitCount++
		case character == ' ' || character == '(' || character == ')' || character == '-':
		case character == '+' && index == 0:
		default:
			return false
		}
	}

	return digitCount >= 10 && digitCount <= 15
}

func validateSelfProfileFields(fields map[string][]string) error {
	for _, field := range [...]string{"name", "surname", "email", "phone", "timezone"} {
		value, submitted := submittedFieldValue(fields, field)
		if !submitted {
			if _, present := fields[field]; present {
				return fmt.Errorf("%w: %s must have exactly one value", errInvalidUserProfile, field)
			}
			continue
		}

		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return fmt.Errorf("%w: %s cannot be empty", errInvalidUserProfile, field)
		}

		switch field {
		case "name", "surname":
			if !isValidPersonName(value) {
				return fmt.Errorf("%w: invalid %s", errInvalidUserProfile, field)
			}
		case "email":
			if len(value) > 255 || !userEmailPattern.MatchString(value) {
				return fmt.Errorf("%w: invalid email", errInvalidUserProfile)
			}
		case "phone":
			if utf8.RuneCountInString(trimmedValue) > 255 || !isValidPhone(value) {
				return fmt.Errorf("%w: invalid phone", errInvalidUserProfile)
			}
		}
	}
	return nil
}

func authorizeUserEdit(actorUID string, actorRole string, targetUID string, fields map[string][]string) (userEditPolicy, error) {
	if actorUID == "" || targetUID == "" {
		return userEditPolicy{}, errUserEditForbidden
	}

	submittedUIDs, uidSubmitted := fields["uid"]
	if uidSubmitted {
		if len(submittedUIDs) != 1 || submittedUIDs[0] != targetUID {
			return userEditPolicy{}, errUserEditForbidden
		}
	}

	permissionIntent := false
	for field := range fields {
		if strings.HasPrefix(field, "perm_view_") || strings.HasPrefix(field, "perm_delete_") {
			permissionIntent = true
		}
	}

	if actorRole == "admin" {
		return userEditPolicy{IsAdmin: true, UpdatePermissions: permissionIntent}, nil
	}

	if actorUID != targetUID {
		return userEditPolicy{}, errUserEditForbidden
	}
	if !uidSubmitted {
		return userEditPolicy{}, errUserEditForbidden
	}

	for field := range fields {
		if isSelfProfileField(field) {
			continue
		}
		if isUserEditProtocolField(field) {
			continue
		}
		return userEditPolicy{}, fmt.Errorf("%w: field %s is not allowed", errUserEditForbidden, field)
	}

	return userEditPolicy{}, nil
}

func decideUserEditRequest(authenticatedUID string, targetUID string, fields map[string][]string, lookup userEditActorLookup) (userEditDecision, error) {
	normalizedFields, err := normalizeUserEditFields(fields)
	if err != nil {
		return userEditDecision{}, err
	}

	targetUIDInt, err := strconv.Atoi(targetUID)
	if err != nil || targetUIDInt <= 0 {
		return userEditDecision{}, errInvalidUserEditUID
	}

	actor, err := lookup(authenticatedUID)
	if err != nil {
		return userEditDecision{}, fmt.Errorf("%w: %v", errUserEditActorLookup, err)
	}
	if actor.UID == "" || !actor.IsActive {
		return userEditDecision{}, errUserEditForbidden
	}

	policy, err := authorizeUserEdit(actor.UID, actor.Role, targetUID, normalizedFields)
	if err != nil {
		return userEditDecision{}, err
	}
	if err = validateSelfProfileFields(normalizedFields); err != nil {
		return userEditDecision{}, err
	}

	return userEditDecision{Policy: policy, TargetUID: targetUIDInt, Fields: normalizedFields}, nil
}

func parseSubmittedBool(values []string) (bool, error) {
	if len(values) == 0 {
		return false, errors.New("missing boolean value")
	}

	result := false
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "on":
			result = true
		case "0", "false", "off", "":
		default:
			return false, fmt.Errorf("invalid boolean value %q", value)
		}
	}

	return result, nil
}

func parseBranchPermissionChanges(targetUID int, fields map[string][]string) ([]branchPermissionChange, error) {
	changesByBranch := make(map[int]*branchPermissionChange)

	for field, values := range fields {
		permissionName := ""
		branchIDText := ""
		switch {
		case strings.HasPrefix(field, "perm_view_"):
			permissionName = "view"
			branchIDText = strings.TrimPrefix(field, "perm_view_")
		case strings.HasPrefix(field, "perm_delete_"):
			permissionName = "delete"
			branchIDText = strings.TrimPrefix(field, "perm_delete_")
		default:
			continue
		}

		branchID, err := strconv.Atoi(branchIDText)
		if err != nil || branchID <= 0 {
			return nil, fmt.Errorf("invalid branch permission field %q", field)
		}
		permissionValue, err := parseSubmittedBool(values)
		if err != nil {
			return nil, fmt.Errorf("invalid branch permission field %q: %w", field, err)
		}

		change, ok := changesByBranch[branchID]
		if !ok {
			change = &branchPermissionChange{TargetUID: targetUID, BranchID: branchID}
			changesByBranch[branchID] = change
		}
		valueCopy := permissionValue
		if permissionName == "view" {
			change.CanView = &valueCopy
		} else {
			change.CanDelete = &valueCopy
		}
	}

	branchIDs := make([]int, 0, len(changesByBranch))
	for branchID := range changesByBranch {
		branchIDs = append(branchIDs, branchID)
	}
	sort.Ints(branchIDs)

	changes := make([]branchPermissionChange, 0, len(branchIDs))
	for _, branchID := range branchIDs {
		changes = append(changes, *changesByBranch[branchID])
	}
	return changes, nil
}
