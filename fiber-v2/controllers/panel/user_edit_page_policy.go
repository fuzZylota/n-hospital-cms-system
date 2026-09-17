package panel

import (
	"errors"
	"fmt"
	"strconv"
)

var (
	errUserEditPageForbidden   = errors.New("user edit page is not permitted")
	errUserEditPageActorLookup = errors.New("user edit page actor lookup failed")
	errInvalidUserEditPageUID  = errors.New("invalid user edit page uid")
)

type userEditPageActor struct {
	UID      string
	Role     string
	IsActive bool
}

type userEditPageActorLookup func(uid string) (userEditPageActor, error)

func authorizeUserEditPage(authenticatedUID string, targetUID string, lookup userEditPageActorLookup) (userEditPageActor, error) {
	targetUIDInt, err := strconv.Atoi(targetUID)
	if authenticatedUID == "" || err != nil || targetUIDInt <= 0 {
		return userEditPageActor{}, errInvalidUserEditPageUID
	}

	actor, err := lookup(authenticatedUID)
	if err != nil {
		return userEditPageActor{}, fmt.Errorf("%w: %v", errUserEditPageActorLookup, err)
	}
	if actor.UID == "" || !actor.IsActive {
		return userEditPageActor{}, errUserEditPageForbidden
	}
	if actor.Role != "admin" && actor.UID != targetUID {
		return userEditPageActor{}, errUserEditPageForbidden
	}

	return actor, nil
}
