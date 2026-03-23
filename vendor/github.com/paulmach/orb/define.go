package orb

// EarthRadius is the radius of the earth in meters. It is used in geo distance calculations.
// To keep things consistent, this value matches WGS84 Web Mercator (EPSG:3857).
const EarthRadius = 6378137.0 // meters

// DefaultRoundingFactor is the default rounding factor used by the Round func.
var DefaultRoundingFactor = 1e6 // 6 decimal places

// Orientation defines the order of the points in a polygon
// or closed ring.
type Orientation int8

// Constants to define orientation.
// They follow the right hand rule for orientation.
const (
	// CCW stands for Counter Clock Wise
	CCW Orientation = 1

	// CW stands for Clock Wise
	CW Orientation = -1
)

// A DistanceFunc is a function that computes the distance between two points.
type DistanceFunc func(Point, Point) float64

// A Projection a function that moves a point from one space to another.
type Projection func(Point) Point

// Pointer is something that can be represented by a point.
type Pointer interface {
	Point() Point
}

// A Simplifier is something that can simplify geometry.
type Simplifier interface {
	Simplify(g Geometry) Geometry
	LineString(ls LineString) LineString
	MultiLineString(mls MultiLineString) MultiLineString
	Ring(r Ring) Ring
	Polygon(p Polygon) Polygon
	MultiPolygon(mp MultiPolygon) MultiPolygon
	Collection(c Collection) Collection
}
