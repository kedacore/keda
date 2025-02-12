// Package collector collects output lines grouped by id and provides ways to
// retrieve and merge output ordered by the time each line was added.
package collector

import (
	"sort"
	"time"
)

// line is a single line of output captured at some point in time.
type line struct {
	Timestamp time.Time
	Text      string
}

// Output stores output lines grouped by id. Output can be retrieved for one or
// more ids and output for different ids can be merged together, while
// preserving their insertion original order based on the time it was
// collected.
// Output also tracks the active id, so you can append output without providing
// an id.
type Output struct {
	m  map[int][]line
	id int // active id
}

// New returns a new output collector.
func New() *Output {
	return &Output{m: make(map[int][]line)}
}

// Clear deletes all output for the given id.
func (o *Output) Clear(id int) {
	delete(o.m, id)
}

// Append appends the given line of text to the output of the currently active
// id.
func (o *Output) Append(text string) {
	o.m[o.id] = append(o.m[o.id], line{time.Now(), text})
}

// AppendToID appends the given line of text to the output of the given id.
func (o *Output) AppendToID(id int, text string) {
	o.m[id] = append(o.m[id], line{time.Now(), text})
}

// Contains returns true if any output lines were collected for the given id.
func (o *Output) Contains(id int) bool {
	return len(o.m[id]) > 0
}

// Get returns the output lines for the given id.
func (o *Output) Get(id int) []string {
	var lines []string
	for _, line := range o.m[id] {
		lines = append(lines, line.Text)
	}
	return lines
}

// GetAll returns the output lines for all ids sorted by the collection
// timestamp of each line of output.
func (o *Output) GetAll(ids ...int) []string {
	var output []line
	for _, id := range ids {
		output = append(output, o.m[id]...)
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i].Timestamp.Before(output[j].Timestamp)
	})
	var lines []string
	for _, line := range output {
		lines = append(lines, line.Text)
	}
	return lines
}

// Merge merges the output lines from fromID into intoID, and sorts the output
// by the collection timestamp of each line of output.
func (o *Output) Merge(fromID, intoID int) {
	var merged []line
	for _, id := range []int{fromID, intoID} {
		merged = append(merged, o.m[id]...)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp.Before(merged[j].Timestamp)
	})
	o.m[intoID] = merged
	delete(o.m, fromID)
}

// SetActiveID sets the active id. Text appended to this output will be
// associated with the active id.
func (o *Output) SetActiveID(id int) {
	o.id = id
}
