package main

import (
	"context"
	"fmt"
	// "github.com/jackc/pgx/v4"
	"strings"
)

type ActionTraceRows struct {
	actData []byte
	offset  uint64
}

type SqlQuery struct {
	key   []string
	value []interface{}
}

func newSqlQuery(filter *DatabaseFilters) *SqlQuery {
	s := &SqlQuery{make([]string, 0), make([]interface{}, 0)}

	if filter.actAccount != nil {
		s.append("act_account=", *filter.actAccount)
	}
	if filter.actName != nil {
		s.append("act_name=", *filter.actName)
	}

	return s
}

func (s *SqlQuery) append(key string, value interface{}) {
	s.value = append(s.value, value)
	s.key = append(s.key, fmt.Sprintf("%s$%d", key, len(s.value)))
}

func (s *SqlQuery) get() (string, []interface{}) {
	where := strings.Join(s.key, " AND ")
	sql := fmt.Sprintf("SELECT act_data, receipt_global_sequence AS offset FROM chain.action_trace WHERE %s ORDER BY receipt_global_sequence ASC", where)

	return sql, s.value
}

func fetchActionData(ctx context.Context, db DatabaseConnect, offset uint64, filter *DatabaseFilters) (*ActionTraceRows, error) {
	s := newSqlQuery(filter)
	s.append("receipt_global_sequence=", offset)

	sql, args := s.get()
	rows := new(ActionTraceRows)

	err := db.QueryRow(ctx, sql, args...).Scan(&rows.actData, &rows.offset)
	return rows, err
}

func fetchAllActionData(ctx context.Context, db DatabaseConnect, offset uint64, count uint, filter *DatabaseFilters) ([]*ActionTraceRows, error) {
	s := newSqlQuery(filter)
	s.append("receipt_global_sequence >=", offset)

	sql, args := s.get()

	if count != 0 { // TODO: not tested
		args = append(args, count)
		sql += fmt.Sprintf(" LIMIT $%d", len(args))
	}

	rows, _ := db.Query(ctx, sql, args...)
	defer rows.Close()

	result := make([]*ActionTraceRows, 0, count)

	for rows.Next() {
		data := new(ActionTraceRows)
		err := rows.Scan(&data.actData, &data.offset)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	return result, rows.Err()
}
