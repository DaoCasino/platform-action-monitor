package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"strings"
)

type ActionTraceRows struct {
	actData []byte
	offset  uint64
}

func filterParams(params []string, filter *DatabaseFilters) []string {
	if filter == nil {
		return params
	}

	if filter.actAccount != nil {
		params = append(params, fmt.Sprintf("act_account='%s'", *filter.actAccount))
	}
	if filter.actName != nil {
		params = append(params, fmt.Sprintf("act_name='%s'", *filter.actName))
	}

	return params
}

func prepareSql(whereParams []string, count uint) string {
	where := strings.Join(whereParams, " AND ")
	sql := fmt.Sprintf("SELECT act_data, receipt_global_sequence AS offset FROM chain.action_trace WHERE %s ORDER BY receipt_global_sequence ASC", where)
	if count != 0 {
		sql += fmt.Sprintf(" LIMIT %d", count)
	}
	// scraperLog.Debug("prepareSql", zap.String("sql", sql))
	return sql
}

func fetchActionData(db *pgx.Conn, offset string, filter *DatabaseFilters) (*ActionTraceRows, error) {
	whereParams := []string{fmt.Sprintf("receipt_global_sequence = %s", offset)}
	whereParams = filterParams(whereParams, filter)

	sql := prepareSql(whereParams, 1)
	rows := new(ActionTraceRows)

	err := db.QueryRow(context.Background(), sql).Scan(&rows.actData, &rows.offset)
	return rows, err
}

func fetchAllActionData(db *pgx.Conn, offset string, count uint, filter *DatabaseFilters) ([]*ActionTraceRows, error) {
	whereParams := []string{fmt.Sprintf("receipt_global_sequence >= %s", offset)}
	whereParams = filterParams(whereParams, filter)

	sql := prepareSql(whereParams, count)

	rows, _ := db.Query(context.Background(), sql)
	defer rows.Close() // TODO: !!! conn busy !!! не работает нихрена надо пул коннектов делать

	result := make([]*ActionTraceRows, 0)

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
