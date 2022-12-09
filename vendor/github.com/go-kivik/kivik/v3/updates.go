package kivik

import (
	"context"
	"net/http"

	"github.com/go-kivik/kivik/v3/driver"
)

// DBUpdates provides access to database updates.
type DBUpdates struct {
	*iter
	updatesi driver.DBUpdates
}

// Next returns the next DBUpdate from the feed. This function will block
// until an event is received. If an error occurs, it will be returned and
// the feed closed. If the feed was closed normally, io.EOF will be returned
// when there are no more events in the buffer.
func (f *DBUpdates) Next() bool {
	return f.iter.Next()
}

// Close closes the feed. Any unread updates will still be accessible via
// Next().
func (f *DBUpdates) Close() error {
	return f.iter.Close()
}

// Err returns the error, if any, that was encountered during iteration. Err
// may be called after an explicit or implicit Close.
func (f *DBUpdates) Err() error {
	return f.iter.Err()
}

type updatesIterator struct{ driver.DBUpdates }

var _ iterator = &updatesIterator{}

func (r *updatesIterator) Next(i interface{}) error { return r.DBUpdates.Next(i.(*driver.DBUpdate)) }

func newDBUpdates(ctx context.Context, updatesi driver.DBUpdates) *DBUpdates {
	return &DBUpdates{
		iter:     newIterator(ctx, &updatesIterator{updatesi}, &driver.DBUpdate{}),
		updatesi: updatesi,
	}
}

// DBName returns the database name for the current update.
func (f *DBUpdates) DBName() string {
	runlock, err := f.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return f.curVal.(*driver.DBUpdate).DBName
}

// Type returns the type of the current update.
func (f *DBUpdates) Type() string {
	runlock, err := f.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return f.curVal.(*driver.DBUpdate).Type
}

// Seq returns the update sequence of the current update.
func (f *DBUpdates) Seq() string {
	runlock, err := f.rlock()
	if err != nil {
		return ""
	}
	defer runlock()
	return f.curVal.(*driver.DBUpdate).Seq
}

// DBUpdates begins polling for database updates.
func (c *Client) DBUpdates(ctx context.Context) (*DBUpdates, error) {
	updater, ok := c.driverClient.(driver.DBUpdater)
	if !ok {
		return nil, &Error{HTTPStatus: http.StatusNotImplemented, Message: "kivik: driver does not implement DBUpdater"}
	}
	updatesi, err := updater.DBUpdates(ctx)
	if err != nil {
		return nil, err
	}
	return newDBUpdates(context.Background(), updatesi), nil
}
