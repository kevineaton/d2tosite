package d2dagrelayout

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"cdr.dev/slog"
	"github.com/dop251/goja"

	"oss.terrastruct.com/util-go/xdefer"

	"oss.terrastruct.com/util-go/go2"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2target"
	"oss.terrastruct.com/d2/lib/geo"
	"oss.terrastruct.com/d2/lib/label"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/shape"
)

//go:embed setup.js
var setupJS string

//go:embed dagre.js
var dagreJS string

type DagreNode struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type DagreEdge struct {
	Points []*geo.Point `json:"points"`
}

type dagreGraphAttrs struct {
	// for a top to bottom graph: ranksep is y spacing, nodesep is x spacing, edgesep is x spacing
	ranksep int
	edgesep int
	nodesep int
	// graph direction: tb (top to bottom)| bt | lr | rl
	rankdir string
}

func Layout(ctx context.Context, g *d2graph.Graph) (err error) {
	defer xdefer.Errorf(&err, "failed to dagre layout")

	debugJS := false
	vm := goja.New()
	if _, err := vm.RunString(dagreJS); err != nil {
		return err
	}
	if _, err := vm.RunString(setupJS); err != nil {
		return err
	}

	rootAttrs := dagreGraphAttrs{
		ranksep: 100,
		edgesep: 40,
		nodesep: 60,
	}
	switch g.Root.Attributes.Direction.Value {
	case "down":
		rootAttrs.rankdir = "TB"
	case "right":
		rootAttrs.rankdir = "LR"
	case "left":
		rootAttrs.rankdir = "RL"
	case "up":
		rootAttrs.rankdir = "BT"
	default:
		rootAttrs.rankdir = "TB"
	}
	configJS := setGraphAttrs(rootAttrs)
	if _, err := vm.RunString(configJS); err != nil {
		return err
	}

	loadScript := ""
	idToObj := make(map[string]*d2graph.Object)
	for _, obj := range g.Objects {
		id := obj.AbsID()
		idToObj[id] = obj
		loadScript += generateAddNodeLine(id, int(obj.Width), int(obj.Height))
		if obj.Parent != g.Root {
			loadScript += generateAddParentLine(id, obj.Parent.AbsID())
		}
	}
	for _, edge := range g.Edges {
		// dagre doesn't work with edges to containers so we connect container edges to their first child instead (going all the way down)
		// we will chop the edge where it intersects the container border so it only shows the edge from the container
		src := edge.Src
		for len(src.Children) > 0 && src.Class == nil && src.SQLTable == nil {
			src = src.ChildrenArray[0]
		}
		dst := edge.Dst
		for len(dst.Children) > 0 && dst.Class == nil && dst.SQLTable == nil {
			dst = dst.ChildrenArray[0]
		}
		if edge.SrcArrow && !edge.DstArrow {
			// for `b <- a`, edge.Edge is `a -> b` and we expect this routing result
			src, dst = dst, src
		}
		loadScript += generateAddEdgeLine(src.AbsID(), dst.AbsID(), edge.AbsID())
	}

	if debugJS {
		log.Debug(ctx, "script", slog.F("all", setupJS+configJS+loadScript))
	}

	if _, err := vm.RunString(loadScript); err != nil {
		return err
	}

	if _, err := vm.RunString(`dagre.layout(g)`); err != nil {
		if debugJS {
			log.Warn(ctx, "layout error", slog.F("err", err))
		}
		return err
	}

	for i := range g.Objects {
		val, err := vm.RunString(fmt.Sprintf("JSON.stringify(g.node(g.nodes()[%d]))", i))
		if err != nil {
			return err
		}
		var dn DagreNode
		if err := json.Unmarshal([]byte(val.String()), &dn); err != nil {
			return err
		}
		if debugJS {
			log.Debug(ctx, "graph", slog.F("json", dn))
		}

		obj := idToObj[dn.ID]

		// dagre gives center of node
		obj.TopLeft = geo.NewPoint(math.Round(dn.X-dn.Width/2), math.Round(dn.Y-dn.Height/2))
		obj.Width = dn.Width
		obj.Height = dn.Height

		if obj.LabelWidth != nil && obj.LabelHeight != nil {
			if len(obj.ChildrenArray) > 0 {
				obj.LabelPosition = go2.Pointer(string(label.InsideTopCenter))
			} else if obj.Attributes.Shape.Value == d2target.ShapeImage || obj.Attributes.Icon != nil {
				obj.LabelPosition = go2.Pointer(string(label.OutsideTopCenter))
			} else {
				obj.LabelPosition = go2.Pointer(string(label.InsideMiddleCenter))
			}
		}
		if obj.Attributes.Icon != nil {
			obj.IconPosition = go2.Pointer(string(label.InsideMiddleCenter))
		}
	}

	for i, edge := range g.Edges {
		val, err := vm.RunString(fmt.Sprintf("JSON.stringify(g.edge(g.edges()[%d]))", i))
		if err != nil {
			return err
		}
		var de DagreEdge
		if err := json.Unmarshal([]byte(val.String()), &de); err != nil {
			return err
		}
		if debugJS {
			log.Debug(ctx, "graph", slog.F("json", de))
		}

		points := make([]*geo.Point, len(de.Points))
		for i := range de.Points {
			if edge.SrcArrow && !edge.DstArrow {
				points[len(de.Points)-i-1] = de.Points[i].Copy()
			} else {
				points[i] = de.Points[i].Copy()
			}
		}

		startIndex, endIndex := 0, len(points)-1
		start, end := points[startIndex], points[endIndex]

		// chop where edge crosses the source/target boxes since container edges were routed to a descendant
		if edge.Src != edge.Dst {
			for i := 1; i < len(points); i++ {
				segment := *geo.NewSegment(points[i-1], points[i])
				if intersections := edge.Src.Box.Intersections(segment); len(intersections) > 0 {
					start = intersections[0]
					startIndex = i - 1
				}

				if intersections := edge.Dst.Box.Intersections(segment); len(intersections) > 0 {
					end = intersections[0]
					endIndex = i
					break
				}
			}
		}

		srcShape := shape.NewShape(d2target.DSL_SHAPE_TO_SHAPE_TYPE[strings.ToLower(edge.Src.Attributes.Shape.Value)], edge.Src.Box)
		dstShape := shape.NewShape(d2target.DSL_SHAPE_TO_SHAPE_TYPE[strings.ToLower(edge.Dst.Attributes.Shape.Value)], edge.Dst.Box)

		// trace the edge to the specific shape's border
		points[startIndex] = shape.TraceToShapeBorder(srcShape, start, points[startIndex+1])
		points[endIndex] = shape.TraceToShapeBorder(dstShape, end, points[endIndex-1])
		points = points[startIndex : endIndex+1]

		// build a curved path from the dagre route
		vectors := make([]geo.Vector, 0, len(points)-1)
		for i := 1; i < len(points); i++ {
			vectors = append(vectors, points[i-1].VectorTo(points[i]))
		}

		path := make([]*geo.Point, 0)
		path = append(path, points[0])
		path = append(path, points[0].AddVector(vectors[0].Multiply(.8)))
		for i := 1; i < len(vectors)-2; i++ {
			p := points[i]
			v := vectors[i]
			path = append(path, p.AddVector(v.Multiply(.2)))
			path = append(path, p.AddVector(v.Multiply(.5)))
			path = append(path, p.AddVector(v.Multiply(.8)))
		}
		path = append(path, points[len(points)-2].AddVector(vectors[len(vectors)-1].Multiply(.2)))
		path = append(path, points[len(points)-1])

		edge.IsCurve = true
		edge.Route = path
		// compile needs to assign edge label positions
		if edge.Attributes.Label.Value != "" {
			edge.LabelPosition = go2.Pointer(string(label.InsideMiddleCenter))
		}
	}

	return nil
}

func setGraphAttrs(attrs dagreGraphAttrs) string {
	return fmt.Sprintf(`g.setGraph({
  ranksep: %d,
  edgesep: %d,
  nodesep: %d,
  rankdir: "%s",
});
`,
		attrs.ranksep,
		attrs.edgesep,
		attrs.nodesep,
		attrs.rankdir,
	)
}

func escapeID(id string) string {
	// fixes \\
	id = strings.ReplaceAll(id, "\\", `\\`)
	// replaces \n with \\n whenever \n is not preceded by \ (does not replace \\n)
	re := regexp.MustCompile(`[^\\]\n`)
	id = re.ReplaceAllString(id, `\\n`)
	// avoid an unescaped \r becoming a \n in the layout result
	id = strings.ReplaceAll(id, "\r", `\r`)
	return id
}

func generateAddNodeLine(id string, width, height int) string {
	id = escapeID(id)
	return fmt.Sprintf("g.setNode(`%s`, { id: `%s`, width: %d, height: %d });\n", id, id, width, height)
}

func generateAddParentLine(childID, parentID string) string {
	return fmt.Sprintf("g.setParent(`%s`, `%s`);\n", escapeID(childID), escapeID(parentID))
}

func generateAddEdgeLine(fromID, toID, edgeID string) string {
	// in dagre v is from, w is to, name is to uniquely identify
	return fmt.Sprintf("g.setEdge({v:`%s`, w:`%s`, name:`%s` });\n", escapeID(fromID), escapeID(toID), escapeID(edgeID))
}
