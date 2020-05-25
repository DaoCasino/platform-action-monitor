package monitor

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
)

var errUserNotExists = errors.New("user not exist")

func isUserExists(ctx context.Context, db DatabaseConnect, token string) (bool, error) {
	id := 0
	err := db.QueryRow(ctx, "SELECT id FROM monitor.users WHERE token=$1 LIMIT 1", token).Scan(&id)

	switch err {
	case nil:
		return true, nil
	case pgx.ErrNoRows:
		return false, nil
	}

	return false, err
}

func checkToken(parentContext context.Context, token string) error {
	if config.skipTokenCheck { // fot unit-test
		return nil
	}

	conn, err := sharedPool.Acquire(parentContext)
	if err != nil {
		return fmt.Errorf("shared pool acquire connection error: %s", err)
	}

	_, err = conn.Exec(parentContext, fmt.Sprintf("SET ROLE %s", config.sharedDatabase.role))
	if err != nil {
		return fmt.Errorf("shared set role error: %s", err)
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
