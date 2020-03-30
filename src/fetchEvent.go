package main

import (
	"context"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

func fetchEvent(ctx context.Context, conn DatabaseConnect, offset uint64) (*Event, error) {
	filter := config.db.filter
	rows, err := fetchActionData(ctx, conn, offset, &filter)
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

func fetchAllEvents(ctx context.Context, conn DatabaseConnect, offset uint64, count uint) ([]*Event, error) {
	filter := config.db.filter

	dataset, err := fetchAllActionData(ctx, conn, offset, count, &filter)
	if err != nil {
		return nil, err
	}

	events := make([]*Event, 0, len(dataset))
	for _, data := range dataset {
		if event, err := abiDecoder.Decode(data.actData); err == nil {
			event.Offset = data.offset
			events = append(events, event)
		}
	}

	return events, nil
}
