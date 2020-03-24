package main

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
)

type DatabaseMock struct{}
type DatabaseMockRow struct{}

func (r *DatabaseMockRow) Scan(dest ...interface{}) error {
	return pgx.ErrNoRows
}

type DatabaseMockRows struct{}

func (r *DatabaseMockRows) Next() bool {
	return false
}
func (r *DatabaseMockRows) Close() {}
func (r *DatabaseMockRows) Err() error {
	return pgx.ErrNoRows
}
func (r *DatabaseMockRows) CommandTag() pgconn.CommandTag {
	return nil
}
func (r *DatabaseMockRows) FieldDescriptions() []pgproto3.FieldDescription {
	return nil
}
func (r *DatabaseMockRows) Scan(dest ...interface{}) error {
	return pgx.ErrNoRows
}
func (r *DatabaseMockRows) Values() ([]interface{}, error) {
	return nil, pgx.ErrNoRows
}
func (r *DatabaseMockRows) RawValues() [][]byte {
	return nil
}
func (m *DatabaseMock) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return new(DatabaseMockRow)
}
func (m *DatabaseMock) Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error) {
	return new(DatabaseMockRows), nil
}
