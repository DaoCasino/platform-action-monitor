package main

import (
	"context"
	"github.com/jackc/pgx/v4"
)

type DatabaseConnect interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
}
