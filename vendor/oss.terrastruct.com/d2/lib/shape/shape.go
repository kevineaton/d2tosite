package shape

import (
	"math"

	"oss.terrastruct.com/d2/lib/geo"
)

const (
	SQUARE_TYPE        = "Square"
	REAL_SQUARE_TYPE   = "RealSquare"
	PARALLELOGRAM_TYPE = "Parallelogram"
	DOCUMENT_TYPE      = "Document"
	CYLINDER_TYPE      = "Cylinder"
	QUEUE_TYPE         = "Queue"
	PAGE_TYPE          = "Page"
	PACKAGE_TYPE       = "Package"
	STEP_TYPE          = "Step"
	CALLOUT_TYPE       = "Callout"
	STORED_DATA_TYPE   = "StoredData"
	PERSON_TYPE        = "Person"
	DIAMOND_TYPE       = "Diamond"
	OVAL_TYPE          = "Oval"
	CIRCLE_TYPE        = "Circle"
	HEXAGON_TYPE       = "Hexagon"
	CLOUD_TYPE         = "Cloud"

	TABLE_TYPE = "Table"
	CLASS_TYPE = "Class"
	TEXT_TYPE  = "Text"
	CODE_TYPE  = "Code"
	IMAGE_TYPE = "Image"
)

type Shape interface {
	Is(shape string) bool
	GetType() string

	AspectRatio1() bool
	IsRectangular() bool

	GetBox() *geo.Box
	GetInnerBox() *geo.Box

	// placing a rectangle of the given size and padding inside the shape, return the position relative to the shape's TopLeft
	GetInsidePlacement(width, height, padding float64) geo.Point

	GetDimensionsToFit(width, height, padding float64) (float64, float64)

	// Perimeter returns a slice of geo.Intersectables that together constitute the shape border
	Perimeter() []geo.Intersectable

	GetSVGPathData() []string
}

type baseShape struct {
	Type string
	Box  *geo.Box
}

func (s baseShape) Is(shapeType string) bool {
	return s.Type == shapeType
}

func (s baseShape) GetType() string {
	return s.Type
}

func (s baseShape) AspectRatio1() bool {
	return false
}

func (s baseShape) IsRectangular() bool {
	return false
}

func (s baseShape) GetBox() *geo.Box {
	return s.Box
}

func (s baseShape) GetInnerBox() *geo.Box {
	return s.Box
}

func (s baseShape) GetInsidePlacement(_, _, padding float64) geo.Point {
	return *geo.NewPoint(s.Box.TopLeft.X+padding, s.Box.TopLeft.Y+padding)
}

func (s baseShape) GetInnerTopLeft(_, _, padding float64) geo.Point {
	return *geo.NewPoint(s.Box.TopLeft.X+padding, s.Box.TopLeft.Y+padding)
}

func (s baseShape) GetDimensionsToFit(width, height, padding float64) (float64, float64) {
	return width + padding*2, height + padding*2
}

func (s baseShape) Perimeter() []geo.Intersectable {
	return nil
}

func (s baseShape) GetSVGPathData() []string {
	return nil
}

func NewShape(shapeType string, box *geo.Box) Shape {
	switch shapeType {
	case CALLOUT_TYPE:
		return NewCallout(box)
	case CIRCLE_TYPE:
		return NewCircle(box)
	case CLASS_TYPE:
		return NewClass(box)
	case CLOUD_TYPE:
		return NewCloud(box)
	case CODE_TYPE:
		return NewCode(box)
	case CYLINDER_TYPE:
		return NewCylinder(box)
	case DIAMOND_TYPE:
		return NewDiamond(box)
	case DOCUMENT_TYPE:
		return NewDocument(box)
	case HEXAGON_TYPE:
		return NewHexagon(box)
	case IMAGE_TYPE:
		return NewImage(box)
	case OVAL_TYPE:
		return NewOval(box)
	case PACKAGE_TYPE:
		return NewPackage(box)
	case PAGE_TYPE:
		return NewPage(box)
	case PARALLELOGRAM_TYPE:
		return NewParallelogram(box)
	case PERSON_TYPE:
		return NewPerson(box)
	case QUEUE_TYPE:
		return NewQueue(box)
	case REAL_SQUARE_TYPE:
		return NewRealSquare(box)
	case STEP_TYPE:
		return NewStep(box)
	case STORED_DATA_TYPE:
		return NewStoredData(box)
	case SQUARE_TYPE:
		return NewSquare(box)
	case TABLE_TYPE:
		return NewTable(box)
	case TEXT_TYPE:
		return NewText(box)

	default:
		return shapeSquare{
			baseShape: &baseShape{
				Type: shapeType,
				Box:  box,
			},
		}
	}
}

// TraceToShapeBorder takes the point on the rectangular border
// r here is the point on rectangular border
// p is the prev point (used to calculate slope)
// s is the point on the actual shape border that'll be returned
//
//      p
//      │
//      │
//      ▼
// ┌────r─────────────────────────┐
// │                              │
// │    │                         │
// │    │      xxxxxxxx           │
// │    ▼  xxxxx       xxxx       │
// │    sxxx               xx     │
// │   x                    xx    │
// │  xx                     xx   │
// │  x                      xx   │
// │  xx                   xxx    │
// │   xxxx             xxxx      │
// └──────xxxxxxxxxxxxxx──────────┘
func TraceToShapeBorder(shape Shape, rectBorderPoint, prevPoint *geo.Point) *geo.Point {
	if shape.Is("") || shape.IsRectangular() {
		return rectBorderPoint
	}

	// We want to extend the line all the way through to the other end of the shape to get the intersections
	scaleSize := shape.GetBox().Width
	if prevPoint.X == rectBorderPoint.X {
		scaleSize = shape.GetBox().Height
	}
	vector := prevPoint.VectorTo(rectBorderPoint)
	vector = vector.AddLength(scaleSize)
	extendedSegment := geo.Segment{Start: prevPoint, End: prevPoint.AddVector(vector)}

	closestD := math.Inf(1)
	closestPoint := rectBorderPoint

	for _, perimeterSegment := range shape.Perimeter() {
		for _, intersectingPoint := range perimeterSegment.Intersections(extendedSegment) {
			d := geo.EuclideanDistance(rectBorderPoint.X, rectBorderPoint.Y, intersectingPoint.X, intersectingPoint.Y)
			if d < closestD {
				closestD = d
				closestPoint = intersectingPoint
			}
		}
	}

	return geo.NewPoint(math.Round(closestPoint.X), math.Round(closestPoint.Y))
}
