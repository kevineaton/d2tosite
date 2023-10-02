package d2exporter

import (
	"context"
	"strconv"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2target"
	"oss.terrastruct.com/d2/d2themes"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
)

func Export(ctx context.Context, g *d2graph.Graph, themeID int64) (*d2target.Diagram, error) {
	theme := d2themescatalog.Find(themeID)

	diagram := d2target.NewDiagram()

	diagram.Shapes = make([]d2target.Shape, len(g.Objects))
	for i := range g.Objects {
		diagram.Shapes[i] = toShape(g.Objects[i], &theme)
	}

	diagram.Connections = make([]d2target.Connection, len(g.Edges))
	for i := range g.Edges {
		diagram.Connections[i] = toConnection(g.Edges[i], &theme)
	}

	return diagram, nil
}

func applyTheme(shape *d2target.Shape, obj *d2graph.Object, theme *d2themes.Theme) {
	shape.Stroke = obj.GetStroke(theme, shape.StrokeDash)
	shape.Fill = obj.GetFill(theme)
	if obj.Attributes.Shape.Value == d2target.ShapeText {
		shape.Color = theme.Colors.Neutrals.N1
	}
}

func applyStyles(shape *d2target.Shape, obj *d2graph.Object) {
	if obj.Attributes.Style.Opacity != nil {
		shape.Opacity, _ = strconv.ParseFloat(obj.Attributes.Style.Opacity.Value, 64)
	}
	if obj.Attributes.Style.StrokeDash != nil {
		shape.StrokeDash, _ = strconv.ParseFloat(obj.Attributes.Style.StrokeDash.Value, 64)
	}
	if obj.Attributes.Style.Fill != nil {
		shape.Fill = obj.Attributes.Style.Fill.Value
	} else if obj.Attributes.Shape.Value == d2target.ShapeText {
		shape.Fill = "transparent"
	}
	if obj.Attributes.Style.Stroke != nil {
		shape.Stroke = obj.Attributes.Style.Stroke.Value
	}
	if obj.Attributes.Style.StrokeWidth != nil {
		shape.StrokeWidth, _ = strconv.Atoi(obj.Attributes.Style.StrokeWidth.Value)
	}
	if obj.Attributes.Style.Shadow != nil {
		shape.Shadow, _ = strconv.ParseBool(obj.Attributes.Style.Shadow.Value)
	}
	if obj.Attributes.Style.ThreeDee != nil {
		shape.ThreeDee, _ = strconv.ParseBool(obj.Attributes.Style.ThreeDee.Value)
	}
	if obj.Attributes.Style.Multiple != nil {
		shape.Multiple, _ = strconv.ParseBool(obj.Attributes.Style.Multiple.Value)
	}
	if obj.Attributes.Style.BorderRadius != nil {
		shape.BorderRadius, _ = strconv.Atoi(obj.Attributes.Style.BorderRadius.Value)
	}

	if obj.Attributes.Style.FontColor != nil {
		shape.Color = obj.Attributes.Style.FontColor.Value
	}
	if obj.Attributes.Shape.Value != d2target.ShapeText {
		if obj.Attributes.Style.Italic != nil {
			shape.Italic, _ = strconv.ParseBool(obj.Attributes.Style.Italic.Value)
		}
		if obj.Attributes.Style.Bold != nil {
			shape.Bold, _ = strconv.ParseBool(obj.Attributes.Style.Bold.Value)
		}
		if obj.Attributes.Style.Underline != nil {
			shape.Underline, _ = strconv.ParseBool(obj.Attributes.Style.Underline.Value)
		}
		if obj.Attributes.Style.Font != nil {
			shape.FontFamily = obj.Attributes.Style.Font.Value
		}
	}
}

func toShape(obj *d2graph.Object, theme *d2themes.Theme) d2target.Shape {
	shape := d2target.BaseShape()
	shape.SetType(obj.Attributes.Shape.Value)
	shape.ID = obj.AbsID()
	shape.ZIndex = obj.ZIndex
	shape.Level = int(obj.Level())
	shape.Pos = d2target.NewPoint(int(obj.TopLeft.X), int(obj.TopLeft.Y))
	shape.Width = int(obj.Width)
	shape.Height = int(obj.Height)

	text := obj.Text()
	shape.Bold = text.IsBold
	shape.Italic = text.IsItalic
	shape.FontSize = text.FontSize

	if obj.IsSequenceDiagram() {
		shape.StrokeWidth = 0
	}

	if obj.IsSequenceDiagramGroup() {
		shape.StrokeWidth = 0
		shape.Blend = true
	}

	applyStyles(shape, obj)
	applyTheme(shape, obj, theme)
	shape.Color = text.GetColor(theme, shape.Italic)
	applyStyles(shape, obj)

	switch obj.Attributes.Shape.Value {
	case d2target.ShapeCode, d2target.ShapeText:
		shape.Language = obj.Attributes.Language
		shape.Label = obj.Attributes.Label.Value
	case d2target.ShapeClass:
		shape.Class = *obj.Class
		// The label is the header for classes and tables, which is set in client to be 4 px larger than the object's set font size
		shape.FontSize -= 4
	case d2target.ShapeSQLTable:
		shape.SQLTable = *obj.SQLTable
		shape.FontSize -= 4
	}
	shape.Label = text.Text
	shape.LabelWidth = text.Dimensions.Width
	shape.LabelHeight = text.Dimensions.Height
	if obj.LabelPosition != nil {
		shape.LabelPosition = *obj.LabelPosition
	}

	shape.Tooltip = obj.Attributes.Tooltip
	shape.Link = obj.Attributes.Link
	shape.Icon = obj.Attributes.Icon
	if obj.IconPosition != nil {
		shape.IconPosition = *obj.IconPosition
	}

	return *shape
}

func toConnection(edge *d2graph.Edge, theme *d2themes.Theme) d2target.Connection {
	connection := d2target.BaseConnection()
	connection.ID = edge.AbsID()
	connection.ZIndex = edge.ZIndex
	// edge.Edge.ID = go2.StringToIntHash(connection.ID)
	text := edge.Text()

	if edge.SrcArrow {
		connection.SrcArrow = d2target.TriangleArrowhead
		if edge.SrcArrowhead != nil {
			if edge.SrcArrowhead.Shape.Value != "" {
				filled := false
				if edge.SrcArrowhead.Style.Filled != nil {
					filled, _ = strconv.ParseBool(edge.SrcArrowhead.Style.Filled.Value)
				}
				connection.SrcArrow = d2target.ToArrowhead(edge.SrcArrowhead.Shape.Value, filled)
			}
		}
	}
	if edge.SrcArrowhead != nil {
		if edge.SrcArrowhead.Label.Value != "" {
			connection.SrcLabel = edge.SrcArrowhead.Label.Value
		}
	}
	if edge.DstArrow {
		connection.DstArrow = d2target.TriangleArrowhead
		if edge.DstArrowhead != nil {
			if edge.DstArrowhead.Shape.Value != "" {
				filled := false
				if edge.DstArrowhead.Style.Filled != nil {
					filled, _ = strconv.ParseBool(edge.DstArrowhead.Style.Filled.Value)
				}
				connection.DstArrow = d2target.ToArrowhead(edge.DstArrowhead.Shape.Value, filled)
			}
		}
	}
	if edge.DstArrowhead != nil {
		if edge.DstArrowhead.Label.Value != "" {
			connection.DstLabel = edge.DstArrowhead.Label.Value
		}
	}

	if edge.Attributes.Style.Opacity != nil {
		connection.Opacity, _ = strconv.ParseFloat(edge.Attributes.Style.Opacity.Value, 64)
	}

	if edge.Attributes.Style.StrokeDash != nil {
		connection.StrokeDash, _ = strconv.ParseFloat(edge.Attributes.Style.StrokeDash.Value, 64)
	}
	connection.Stroke = edge.GetStroke(theme, connection.StrokeDash)
	if edge.Attributes.Style.Stroke != nil {
		connection.Stroke = edge.Attributes.Style.Stroke.Value
	}

	if edge.Attributes.Style.StrokeWidth != nil {
		connection.StrokeWidth, _ = strconv.Atoi(edge.Attributes.Style.StrokeWidth.Value)
	}

	if edge.Attributes.Style.Fill != nil {
		connection.Fill = edge.Attributes.Style.Fill.Value
	}

	connection.FontSize = text.FontSize
	if edge.Attributes.Style.FontSize != nil {
		connection.FontSize, _ = strconv.Atoi(edge.Attributes.Style.FontSize.Value)
	}

	if edge.Attributes.Style.Animated != nil {
		connection.Animated, _ = strconv.ParseBool(edge.Attributes.Style.Animated.Value)
	}

	connection.Tooltip = edge.Attributes.Tooltip
	connection.Icon = edge.Attributes.Icon

	if edge.Attributes.Style.Italic != nil {
		connection.Italic, _ = strconv.ParseBool(edge.Attributes.Style.Italic.Value)
	}

	connection.Color = text.GetColor(theme, connection.Italic)
	if edge.Attributes.Style.FontColor != nil {
		connection.Color = edge.Attributes.Style.FontColor.Value
	}
	if edge.Attributes.Style.Bold != nil {
		connection.Bold, _ = strconv.ParseBool(edge.Attributes.Style.Bold.Value)
	}
	if edge.Attributes.Style.Font != nil {
		connection.FontFamily = edge.Attributes.Style.Font.Value
	}
	connection.Label = text.Text
	connection.LabelWidth = text.Dimensions.Width
	connection.LabelHeight = text.Dimensions.Height

	if edge.LabelPosition != nil {
		connection.LabelPosition = *edge.LabelPosition
	}
	if edge.LabelPercentage != nil {
		connection.LabelPercentage = *edge.LabelPercentage
	}
	connection.Route = edge.Route
	connection.IsCurve = edge.IsCurve

	connection.Src = edge.Src.AbsID()
	connection.Dst = edge.Dst.AbsID()

	return *connection
}
