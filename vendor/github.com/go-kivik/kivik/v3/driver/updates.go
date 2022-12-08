package driver

import "context"

// DBUpdate represents a database update event.
type DBUpdate struct {
	DBName string `json:"db_name"`
	Type   string `json:"type"`
	Seq    string `json:"seq"`
}

// DBUpdates is a DBUpdates iterator.
type DBUpdates interface {
	// Next is called to populate DBUpdate with the values of the next update in
	// the feed.
	//
	// Next should return io.EOF when the feed is closed normally.
	Next(*DBUpdate) error
	// Close closes the iterator.
	Close() error
}

// DBUpdater is an optional interface that may be implemented by a Client to
// provide access to the DB Updates feed.
type DBUpdater interface {
	// DBUpdates must return a DBUpdate iterator. The context, or the iterator's
	// Close method, may be used to close the iterator.
	DBUpdates(context.Context) (DBUpdates, error)
}
