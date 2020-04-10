package monitor

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

const (
	sqlFetchAction       = "SELECT action_trace.act_data, action_trace.receipt_global_sequence AS offset FROM chain.action_trace WHERE %s ORDER BY action_trace.receipt_global_sequence ASC"
	sqlFetchActions      = "SELECT action_trace.act_data, action_trace.receipt_global_sequence AS offset FROM chain.action_trace INNER JOIN chain.block_info ON block_info.block_num = action_trace.block_num WHERE %s ORDER BY action_trace.receipt_global_sequence ASC"
	sqlWhereEventExpires = "block_info.timestamp > now() - interval '%s'"
	sqlWhereActAccount   = "action_trace.act_account="
	sqlWhereActName      = "action_trace.act_name="
	sqlWhereAnd          = " AND "
)

func newSqlQuery(filter *DatabaseFilters) *SqlQuery {
	s := &SqlQuery{make([]string, 0), make([]interface{}, 0)}

	if filter.actAccount != nil {
		s.append(sqlWhereActAccount, *filter.actAccount)
	}
	if filter.actName != nil {
		s.append(sqlWhereActName, *filter.actName)
	}

	return s
}

func (s *SqlQuery) append(key string, value interface{}) {
	s.value = append(s.value, value)
	s.key = append(s.key, fmt.Sprintf("%s$%d", key, len(s.value)))
}

func (s *SqlQuery) getRow() (string, []interface{}) {
	where := strings.Join(s.key, sqlWhereAnd)
	sql := fmt.Sprintf(sqlFetchAction, where)

	return sql, s.value
}

func (s *SqlQuery) getRows(eventExpires *string) (string, []interface{}) {
	if eventExpires != nil {
		s.key = append(s.key, fmt.Sprintf(sqlWhereEventExpires, *eventExpires))
	}

	where := strings.Join(s.key, sqlWhereAnd)
	sql := fmt.Sprintf(sqlFetchActions, where)
	return sql, s.value
}

func fetchActionData(ctx context.Context, db DatabaseConnect, offset uint64, filter *DatabaseFilters) (*ActionTraceRows, error) {
	s := newSqlQuery(filter)
	s.append("action_trace.receipt_global_sequence =", offset)

	sql, args := s.getRow()
	rows := new(ActionTraceRows)

	err := db.QueryRow(ctx, sql, args...).Scan(&rows.actData, &rows.offset)
	return rows, err
}

func fetchAllActionData(ctx context.Context, db DatabaseConnect, offset uint64, count uint, eventExpires *string, filter *DatabaseFilters) ([]*ActionTraceRows, error) {
	s := newSqlQuery(filter)
	s.append("action_trace.receipt_global_sequence >=", offset)
	sql, args := s.getRows(eventExpires)

	if count != 0 {
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
