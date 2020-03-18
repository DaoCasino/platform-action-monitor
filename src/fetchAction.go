package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"strings"
)

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
	sql := fmt.Sprintf("SELECT act_data FROM chain.action_trace WHERE %s ORDER BY receipt_global_sequence ASC LIMIT %d", where, count)
	scraperLog.Debug("prepareSql", zap.String("sql", sql))
	return sql
}

func fetchActionData(db *pgx.Conn, offset string, filter *DatabaseFilters) (data []byte, err error) {
	whereParams := []string{fmt.Sprintf("receipt_global_sequence = %s", offset)}
	whereParams = filterParams(whereParams, filter)

	sql := prepareSql(whereParams, 1)
	err = db.QueryRow(context.Background(), sql).Scan(&data)
	return
}

func fetchAllActionData(db *pgx.Conn, offset string, count uint, filter *DatabaseFilters) ([][]byte, error) {
	whereParams := []string{fmt.Sprintf("receipt_global_sequence >= %s", offset)}
	whereParams = filterParams(whereParams, filter)

	sql := prepareSql(whereParams, count)

	rows, _ := db.Query(context.Background(), sql)

	result := make([][]byte, 0)

	for rows.Next() {
		var data []byte
		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	return result, rows.Err()
}
