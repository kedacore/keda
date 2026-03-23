package orb

// MultiPolygon is a set of polygons.
type MultiPolygon []Polygon

// GeoJSONType returns the GeoJSON type for the object.
func (mp MultiPolygon) GeoJSONType() string {
	return "MultiPolygon"
}

// Dimensions returns 2 because a MultiPolygon is a 2d object.
func (mp MultiPolygon) Dimensions() int {
	return 2
}

// Bound returns a bound around the multi-polygon.
func (mp MultiPolygon) Bound() Bound {
	if len(mp) == 0 {
		return emptyBound
	}
	bound := mp[0].Bound()
	for i := 1; i < len(mp); i++ {
		bound = bound.Union(mp[i].Bound())
	}

	return bound
}

// Equal compares two multi-polygons.
func (mp MultiPolygon) Equal(multiPolygon MultiPolygon) bool {
	if len(mp) != len(multiPolygon) {
		return false
	}

	for i, p := range mp {
		if !p.Equal(multiPolygon[i]) {
			return false
		}
	}

	return true
}

// Clone returns a new deep copy of the multi-polygon.
func (mp MultiPolygon) Clone() MultiPolygon {
	if mp == nil {
		return nil
	}

	nmp := make(MultiPolygon, 0, len(mp))
	for _, p := range mp {
		nmp = append(nmp, p.Clone())
	}

	return nmp
}
