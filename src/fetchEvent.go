package main

import (
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

func fetchEvent(conn *pgx.Conn, offset uint64) (*Event, error) {
	filter := config.db.filter
	rows, err := fetchActionData(conn, offset, &filter)
	switch err {
	case nil:
		// ok
		if event, err := abiDecoder.Decode(rows.actData); err == nil {
			event.Offset = rows.offset
			return event, nil
		}
	case pgx.ErrNoRows:
		scraperLog.Debug("no act_data with filter",
			zap.Stringp("act_name", filter.actName),
			zap.Stringp("act_account", filter.actAccount),
		)
	default:
		scraperLog.Error("handleNotify SQL error", zap.Error(err))
	}

	return nil, err
}

func fetchAllEvents(conn *pgx.Conn, offset uint64, count uint) ([]*Event, error) {
	filter := config.db.filter
	events := make([]*Event, 0)
	dataset, err := fetchAllActionData(conn, offset, count, &filter)
	if err != nil {
		return nil, err
	}

	for _, data := range dataset {
		if event, err := abiDecoder.Decode(data.actData); err == nil {
			event.Offset = data.offset
			events = append(events, event)
		}
	}

	return events, nil
}
