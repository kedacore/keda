package beanstalk

import (
	"time"
)

// TubeSet represents a set of tubes on the server connected to by Conn.
// Name names the tubes represented.
type TubeSet struct {
	Conn *Conn
	Name map[string]bool
}

// NewTubeSet returns a new TubeSet representing the given names.
func NewTubeSet(c *Conn, name ...string) *TubeSet {
	ts := &TubeSet{c, make(map[string]bool)}
	for _, s := range name {
		ts.Name[s] = true
	}
	return ts
}

// Reserve reserves and returns a job from one of the tubes in t. If no
// job is available before time timeout has passed, Reserve returns a
// ConnError recording ErrTimeout.
//
// Typically, a client will reserve a job, perform some work, then delete
// the job with Conn.Delete.
func (t *TubeSet) Reserve(timeout time.Duration) (id uint64, body []byte, err error) {
	r, err := t.Conn.cmd(nil, t, nil, "reserve-with-timeout", dur(timeout))
	if err != nil {
		return 0, nil, err
	}
	body, err = t.Conn.readResp(r, true, "RESERVED %d", &id)
	if err != nil {
		return 0, nil, err
	}
	return id, body, nil
}
