package orb

import (
	"fmt"
)

// Clone will make a deep copy of the geometry.
func Clone(g Geometry) Geometry {
	if g == nil {
		return nil
	}

	switch g := g.(type) {
	case Point:
		return g
	case MultiPoint:
		if g == nil {
			return nil
		}
		return g.Clone()
	case LineString:
		if g == nil {
			return nil
		}
		return g.Clone()
	case MultiLineString:
		if g == nil {
			return nil
		}
		return g.Clone()
	case Ring:
		if g == nil {
			return nil
		}
		return g.Clone()
	case Polygon:
		if g == nil {
			return nil
		}
		return g.Clone()
	case MultiPolygon:
		if g == nil {
			return nil
		}
		return g.Clone()
	case Collection:
		if g == nil {
			return nil
		}
		return g.Clone()
	case Bound:
		return g
	}

	panic(fmt.Sprintf("geometry type not supported: %T", g))
}
