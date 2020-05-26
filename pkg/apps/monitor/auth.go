package monitor

import (
	"context"
	"errors"
	"fmt"
)

var errUserNotExists = errors.New("user not exist")

func isUserExists(ctx context.Context, db DatabaseConnect, token string) (bool, error) {
	cnt := 0
	err := db.QueryRow(ctx, "SELECT count(token) FROM monitor.users WHERE token = $1", token).Scan(&cnt)

	if err != nil {
		return false, err
	}

	return !(cnt == 0), nil
}

func checkToken(parentContext context.Context, token string) error {
	if config.skipTokenCheck { // for unit testing
		return nil
	}

	conn, err := sharedPool.Acquire(parentContext)
	if err != nil {
		return fmt.Errorf("shared pool acquire connection error: %s", err)
	}

	defer func() {
		conn.Release()
	}()

	var ok bool
	ok, err = isUserExists(parentContext, conn, token)
	if err != nil {
		return fmt.Errorf("shared query error: %s", err)
	}

	if !ok {
		return errUserNotExists
	}

	return nil
}
