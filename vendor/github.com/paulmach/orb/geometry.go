package orb

// Geometry is an interface that represents the shared attributes
// of a geometry.
type Geometry interface {
	GeoJSONType() string
	Dimensions() int // e.g. 0d, 1d, 2d
	Bound() Bound

	// requiring because sub package type switch over all possible types.
	private()
}

// compile time checks
var (
	_ Geometry = Point{}
	_ Geometry = MultiPoint{}
	_ Geometry = LineString{}
	_ Geometry = MultiLineString{}
	_ Geometry = Ring{}
	_ Geometry = Polygon{}
	_ Geometry = MultiPolygon{}
	_ Geometry = Bound{}

	_ Geometry = Collection{}
)

func (p Point) private()             {}
func (mp MultiPoint) private()       {}
func (ls LineString) private()       {}
func (mls MultiLineString) private() {}
func (r Ring) private()              {}
func (p Polygon) private()           {}
func (mp MultiPolygon) private()     {}
func (b Bound) private()             {}
func (c Collection) private()        {}

// AllGeometries lists all possible types and values that a geometry
// interface can be. It should be used only for testing to verify
// functions that accept a Geometry will work in all cases.
var AllGeometries = []Geometry{
	nil,
	Point{},
	MultiPoint{},
	LineString{},
	MultiLineString{},
	Ring{},
	Polygon{},
	MultiPolygon{},
	Bound{},
	Collection{},

	// nil values
	MultiPoint(nil),
	LineString(nil),
	MultiLineString(nil),
	Ring(nil),
	Polygon(nil),
	MultiPolygon(nil),
	Collection(nil),

	// Collection of Collection
	Collection{Collection{Point{}}},
}

// A Collection is a collection of geometries that is also a Geometry.
type Collection []Geometry

// GeoJSONType returns the geometry collection type.
func (c Collection) GeoJSONType() string {
	return "GeometryCollection"
}

// Dimensions returns the max of the dimensions of the collection.
func (c Collection) Dimensions() int {
	max := -1
	for _, g := range c {
		if d := g.Dimensions(); d > max {
			max = d
		}
	}

	return max
}

// Bound returns the bounding box of all the Geometries combined.
func (c Collection) Bound() Bound {
	if len(c) == 0 {
		return emptyBound
	}

	var b Bound
	start := -1

	for i, g := range c {
		if g != nil {
			start = i
			b = g.Bound()
			break
		}
	}

	if start == -1 {
		return emptyBound
	}

	for i := start + 1; i < len(c); i++ {
		if c[i] == nil {
			continue
		}

		b = b.Union(c[i].Bound())
	}

	return b
}

// Equal compares two collections. Returns true if lengths are the same
// and all the sub geometries are the same and in the same order.
func (c Collection) Equal(collection Collection) bool {
	if len(c) != len(collection) {
		return false
	}

	for i, g := range c {
		if !Equal(g, collection[i]) {
			return false
		}
	}

	return true
}

// Clone returns a deep copy of the collection.
func (c Collection) Clone() Collection {
	if c == nil {
		return nil
	}

	nc := make(Collection, len(c))
	for i, g := range c {
		nc[i] = Clone(g)
	}

	return nc
}
