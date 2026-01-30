package orb

// LineString represents a set of points to be thought of as a polyline.
type LineString []Point

// GeoJSONType returns the GeoJSON type for the object.
func (ls LineString) GeoJSONType() string {
	return "LineString"
}

// Dimensions returns 1 because a LineString is a 1d object.
func (ls LineString) Dimensions() int {
	return 1
}

// Reverse will reverse the line string.
// This is done inplace, ie. it modifies the original data.
func (ls LineString) Reverse() {
	l := len(ls) - 1
	for i := 0; i <= l/2; i++ {
		ls[i], ls[l-i] = ls[l-i], ls[i]
	}
}

// Bound returns a rect around the line string. Uses rectangular coordinates.
func (ls LineString) Bound() Bound {
	return MultiPoint(ls).Bound()
}

// Equal compares two line strings. Returns true if lengths are the same
// and all points are Equal.
func (ls LineString) Equal(lineString LineString) bool {
	return MultiPoint(ls).Equal(MultiPoint(lineString))
}

// Clone returns a new copy of the line string.
func (ls LineString) Clone() LineString {
	ps := MultiPoint(ls)
	return LineString(ps.Clone())
}
