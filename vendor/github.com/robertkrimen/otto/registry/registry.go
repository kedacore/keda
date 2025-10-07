// Package registry is an experimental package to facilitate altering the otto runtime via import.
//
// This interface can change at any time.
package registry

var registry []*Entry = make([]*Entry, 0)

// Entry represents a registry entry.
type Entry struct {
	source func() string
	active bool
}

// newEntry returns a new Entry for source.
func newEntry(source func() string) *Entry {
	return &Entry{
		active: true,
		source: source,
	}
}

// Enable enables the entry.
func (e *Entry) Enable() {
	e.active = true
}

// Disable disables the entry.
func (e *Entry) Disable() {
	e.active = false
}

// Source returns the source of the entry.
func (e Entry) Source() string {
	return e.source()
}

// Apply applies callback to all registry entries.
func Apply(callback func(Entry)) {
	for _, entry := range registry {
		if !entry.active {
			continue
		}
		callback(*entry)
	}
}

// Register registers a new Entry for source.
func Register(source func() string) *Entry {
	entry := newEntry(source)
	registry = append(registry, entry)
	return entry
}
