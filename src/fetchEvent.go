package main

import (
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"log"
)

//
type FetchEvent struct { // TODO: надо для расшифровки данных из бд
	abi    *AbiDecoder
	conn   *pgx.Conn
	filter *DatabaseFilters
}

func newFetchEvent(registry *Registry) *FetchEvent {
	abi := registry.get("abiDecoder").(*AbiDecoder)
	conn := registry.get("db").(*pgx.Conn)
	config := registry.get("config").(*Config)

	return &FetchEvent{abi, conn, &config.db.filter}
}

func (f *FetchEvent) fetch(offset string) (*Event, error) {
	rows, err := fetchActionData(f.conn, offset, f.filter)
	switch err {
	case nil:
		// ok
		if event, err := f.abi.Decode(rows.actData); err == nil {
			// event.Offset = rows.offset TODO: !!!! надо!!
			return event, nil
		}
	case pgx.ErrNoRows:
		scraperLog.Debug("no act_data with filter",
			zap.Stringp("act_name", f.filter.actName),
			zap.Stringp("act_account", f.filter.actAccount),
		)
	default:
		scraperLog.Error("handleNotify SQL error", zap.Error(err))
	}

	return nil, err
}

func (f *FetchEvent) fetchAll(offset string, count uint) ([]*Event, error) {
	events := make([]*Event, 0)
	dataset, err := fetchAllActionData(f.conn, offset, count, f.filter)
	if err != nil {
		return nil, err
	}

	log.Printf("%+v", dataset)

	for _, data := range dataset {
		if event, err := f.abi.Decode(data.actData); err == nil {
			// event.Offset = data.offset  TODO: !!!! надо
			events = append(events, event)
		}
	}

	return events, nil
}
