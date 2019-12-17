package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Map struct {
	Height     int          `hcl:"height,optional"`
	Width      int          `hcl:"width,optional"`
	Margin     int          `hcl:"margin,optional"`
	ShowGuides bool         `hcl:"show_guides,optional"`
	Nodes      []*Node      `hcl:"node,block"`
	Connectors []*Connector `hcl:"connector,block"`
}

type Node struct {
	ID          string `hcl:"id,label"`
	Label       string `hcl:"label"`
	Description string `hcl:"description,optional"`
	X           int
	Y           int
	Visibility  int    `hcl:"visibility"`
	Evolution   string `hcl:"evolution,optional"`
	EvolutionX  int    `hcl:"x,optional"`
	Fill        string `hcl:"fill,optional"`
	Color       string `hcl:"color,optional"`
}

type Connector struct {
	Label string `hcl:"label,optional"`
	From  string `hcl:"from"`
	To    string `hcl:"to"`
	Color string `hcl:"color,optional"`
	Type  string `hcl:"type,optional"`
}

func ParseFile(filename string) (*Map, error) {
	var m Map
	err := hclsimple.DecodeFile(filename, nil, &m)
	if err != nil {
		return nil, err
	}
	ApplyDefaults(&m)
	return &m, nil
}

type Defaults struct {
	Height, Width, Margin int
	Node                  struct {
		Evolution  string
		EvolutionX int
		Fill       string
		Color      string
	}
	Connector struct {
		Color string
		Type  string
	}
}

var DEFAULTS = Defaults{
	Height: 768,
	Width:  1280,
	Margin: 40,
	Node: struct {
		Evolution  string
		EvolutionX int
		Fill       string
		Color      string
	}{
		Evolution:  "product",
		EvolutionX: 1,
		Fill:       "black",
		Color:      "black",
	},
	Connector: struct {
		Color, Type string
	}{
		Color: "black",
		Type:  "normal",
	},
}

func ApplyDefaults(m *Map) {
	if m.Height == 0 {
		m.Height = DEFAULTS.Height
	}
	if m.Width == 0 {
		m.Width = DEFAULTS.Width
	}
	if m.Margin == 0 {
		m.Margin = DEFAULTS.Margin
	}
	for _, node := range m.Nodes {
		if node.Evolution == "" {
			node.Evolution = DEFAULTS.Node.Evolution
		}
		if node.EvolutionX == 0 {
			node.EvolutionX = DEFAULTS.Node.EvolutionX
		}
		if node.Fill == "" {
			node.Fill = DEFAULTS.Node.Fill
		}
		if node.Color == "" {
			node.Fill = DEFAULTS.Node.Fill
		}
	}
	for _, connector := range m.Connectors {
		if connector.Color == "" {
			connector.Color = DEFAULTS.Connector.Color
		}
		if connector.Type == "" {
			connector.Type = DEFAULTS.Connector.Type
		}
	}
}

func (m *Map) RenderSVG(w io.Writer) {
	canvas := svg.New(w)

	mapGrid := Grid{
		XQuarterLength: (m.Width - m.Margin*4) / 4,
		Genesis:        0,
		Custom:         (m.Width - m.Margin*4) / 4,
		Product:        (m.Width - m.Margin*4) * 2 / 4,
		Commodity:      (m.Width - m.Margin*4) * 3 / 4,
		YLength:        m.Height - m.Margin*4,
		Visible:        0,
	}

	canvas.Start(m.Width, m.Height)
	canvas.Gstyle("font-family:sans-serif")
	grid(canvas, m.Margin, m.Width, m.Height, m.ShowGuides)
	canvas.Translate(m.Margin*2, m.Height-m.Margin*2)
	canvas.Marker("connector-arrow", 17, 3, 12, 10, `orient="auto"`)
	canvas.Path("M0,0 L0,6 L12,3 z")
	canvas.MarkerEnd()
	canvas.Marker("connector-inertia", 0, 10, 20, 40, `orient="auto"`)
	canvas.Path("M-5,20 L-5,-20 L5,-20 L5,20")
	canvas.MarkerEnd()

	maxGenesis, maxCustom, maxProduct, maxCommodity := 0, 0, 0, 0
	maxY := 0
	for _, n := range m.Nodes {
		if n.Evolution == "genesis" && n.EvolutionX > maxGenesis {
			maxGenesis = n.EvolutionX
		}
		if n.Evolution == "custom" && n.EvolutionX > maxCustom {
			maxCustom = n.EvolutionX
		}
		if n.Evolution == "product" && n.EvolutionX > maxProduct {
			maxProduct = n.EvolutionX
		}
		if n.Evolution == "commodity" && n.EvolutionX > maxCommodity {
			maxCommodity = n.EvolutionX
		}
		if n.Visibility > maxY {
			maxY = n.Visibility
		}
	}
	for _, n := range m.Nodes {
		NodeXY(n, mapGrid, maxGenesis, maxCustom, maxProduct, maxCommodity, maxY)
	}
	for _, c := range m.Connectors {
		var a, b *Node
		for _, n := range m.Nodes {
			if n.ID == c.From {
				a = n
			}
			if n.ID == c.To {
				b = n
			}
			if a != nil && b != nil {
				break
			}
		}
		if a == nil || b == nil {
			log.Printf("ERROR: couldn't find node '%s'\n", c.From)
			continue
		}
		if b == nil {
			log.Printf("ERROR: couldn't find node '%s'\n", c.To)
			continue
		}
		connect(canvas, c, a, b)
	}
	for _, n := range m.Nodes {
		DrawNode(canvas, n)
	}
	canvas.Gend()
	canvas.Gend()
	canvas.End()
}

// Grid -
type Grid struct {
	XQuarterLength int
	YLength        int
	Genesis        int
	Custom         int
	Product        int
	Commodity      int
	Visible        int
}

func NodeXY(n *Node, mapGrid Grid, maxGenesis, maxCustom, maxProduct, maxCommodity, maxY int) {
	switch n.Evolution {
	case "genesis":
		n.X = mapGrid.Genesis + mapGrid.XQuarterLength/(maxGenesis+1)*n.EvolutionX
	case "custom":
		n.X = mapGrid.Custom + mapGrid.XQuarterLength/(maxCustom+1)*n.EvolutionX
	case "product":
		n.X = mapGrid.Product + mapGrid.XQuarterLength/(maxProduct+1)*n.EvolutionX
	case "commodity":
		n.X = mapGrid.Commodity + mapGrid.XQuarterLength/(maxCommodity+1)*n.EvolutionX
	}

	n.Y = -mapGrid.YLength / (maxY + 1) * (maxY + 1 - n.Visibility)
}

// DrawNode -
func DrawNode(canvas *svg.SVG, n *Node) {
	nodeFontSize := 9
	canvas.Gstyle("text-shadow: 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white")
	if n.Description != "" {
		canvas.Title(n.Description)
	} else {
		canvas.Title(n.Label)
	}
	canvas.Circle(n.X, n.Y, 5, fmt.Sprintf("fill:%s;stroke:%s", n.Fill, n.Color))
	// canvas.Text(n.X+10, n.Y+3, n.Label, fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black;text-shadow: -1px 0 white, 0 1px white, 1px 0 white, 0 -1px white", nodeFontSize))
	// canvas.Gstyle("text-shadow: -1px 0 white, 0 1px white, 1px 0 white, 0 -1px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white, 0 0 3px white")
	canvas.Textlines(n.X+8, n.Y+10, strings.Split(n.Label, "\n"), nodeFontSize, nodeFontSize+3, "black", "left")
	canvas.Gend()

}

func connect(canvas *svg.SVG, c *Connector, a, b *Node) {
	nodeFontSize := 9

	// Calculate midpoints
	x := a.X + (b.X-a.X)/2
	if a.X > b.X {
		x = b.X + (a.X-b.X)/2
	}
	y := a.Y + (b.Y-a.Y)/2
	if a.Y > b.Y {
		y = b.Y + (a.Y-b.Y)/2
	}

	// canvas.Def()
	// canvas.Path(fmt.Sprintf("M %d,%d %d,%d %d,%d", a.X, a.Y, x, y, b.X, b.Y), fmt.Sprintf(`id="%d"`, connectID))
	// canvas.DefEnd()
	switch c.Type {
	case "normal":
		canvas.Path(fmt.Sprintf("M %d,%d %d,%d", a.X, a.Y, b.X, b.Y),
			fmt.Sprintf(`id="%s-%s"`, a.ID, b.ID),
			fmt.Sprintf(`fill:none;stroke:%s;opacity:0.2`, c.Color))
	case "bold":
		canvas.Path(fmt.Sprintf("M %d,%d %d,%d", a.X, a.Y, b.X, b.Y), fmt.Sprintf(`fill:none;stroke:%s;opacity:0.8`, c.Color))
	case "change":
		canvas.Path(fmt.Sprintf("M %d,%d %d,%d %d,%d", a.X, a.Y, x, y, b.X, b.Y),
			fmt.Sprintf(`fill:white;stroke:%[1]s;opacity:0.6;stroke-dasharray:6,6;marker-end:url(#connector-arrow)`, c.Color))
	case "change-inertia":
		canvas.Path(fmt.Sprintf("M %d,%d %d,%d %d,%d", a.X, a.Y, x, y, b.X, b.Y),
			fmt.Sprintf(`fill:white;stroke:%[1]s;opacity:0.6;stroke-dasharray:6,6;marker-mid:url(#connector-inertia);marker-end:url(#connector-arrow)`, c.Color))
	}
	x += 8
	y += 10

	// if strings.Contains(c.Label, "\n") {
	canvas.Textlines(x, y, strings.Split(c.Label, "\n"), nodeFontSize, nodeFontSize+3, "black", "left")
	// } else {
	// 	s.Textpath(c.Label, fmt.Sprintf("#%d", connectID), `x="10" y="-5"`, fmt.Sprintf("text-align:left;font-size:%dpx;fill:black", nodeFontSize))
	// }
}

func grid(canvas *svg.SVG, margin, width, height int, showGuides bool) {
	// TODO: Variable font size based on width and height
	fontSize := 12

	// Grid
	//   X
	xLength := width - margin*4
	xZero := margin * 2
	xEnd := width - margin*2
	//   Y
	yLength := height - margin*4
	yZero := height - margin*2
	yEnd := margin * 2

	canvas.Rect(0, 0, width, height, "fill:white")

	if showGuides {
		// Limits Guide
		canvas.Line(0, height, width, height, "fill:none;stroke:red")
		canvas.Line(width, 0, width, height, "fill:none;stroke:red")
		// Margin guide
		canvas.Line(margin, height-margin, width-margin, height-margin, "fill:none;stroke:green")
		canvas.Line(width-margin, margin, width-margin, height-margin, "fill:none;stroke:green")
		canvas.Line(margin, margin, width-margin, margin, "fill:none;stroke:green")
		canvas.Line(margin, margin, margin, height-margin, "fill:none;stroke:green")

		canvas.Translate(xZero, yZero)
		canvas.Text(xLength-40, -yLength, fmt.Sprintf("%d,%d", xLength, yLength), fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:green", fontSize))
		canvas.Text(0, 0, fmt.Sprintf("%d,%d", xZero, yZero), fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:green", fontSize))
		canvas.Text(xLength/4, 0, fmt.Sprintf("%d,%d", xLength/4, 0), fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:green", fontSize))
		canvas.Text(xLength*2/4, 0, fmt.Sprintf("%d,%d", 2*margin+xLength*2/4, 0), fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:green", fontSize))
		canvas.Text(xLength*3/4, 0, fmt.Sprintf("%d,%d", 2*margin+xLength*3/4, 0), fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:green", fontSize))
		canvas.Gend()
	}

	// Grid
	canvas.Marker("arrow", 0, 3, 12, 10, `orient="auto"`)
	canvas.Path("M0,0 L0,6 L12,3 z", "fill:black")
	canvas.MarkerEnd()
	canvas.Line(xZero, yZero, xEnd, yZero, "fill:none;stroke:black;marker-end:url(#arrow)")
	canvas.Line(xZero, yZero, xZero, yEnd, "fill:blue;stroke:black;marker-end:url(#arrow)")

	canvas.Line(2*margin+xLength/4, yZero, 2*margin+xLength/4, yEnd, `fill:none;stroke:gray;stroke-dasharray:1,10`)
	canvas.Line(2*margin+xLength*2/4, yZero, 2*margin+xLength*2/4, yEnd, `fill:none;stroke:gray;stroke-dasharray:1,10`)
	canvas.Line(2*margin+xLength*3/4, yZero, 2*margin+xLength*3/4, yEnd, `fill:none;stroke:gray;stroke-dasharray:1,10`)

	// Text
	canvas.Text(xZero, height-margin, "Genesis", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black", fontSize))
	canvas.Text(2*margin+xLength/4, height-margin, "Custom", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black", fontSize))
	canvas.Text(2*margin+xLength*2/4, height-margin, "Product (+rental)", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black", fontSize))
	canvas.Text(2*margin+xLength*3/4, height-margin, "Commodity (+utility)", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black", fontSize))
	canvas.Text(xEnd-100, height-2*margin-5, "Evolution", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black;font-weight:bold;font-family:serif", fontSize+2))

	canvas.TranslateRotate(xZero, yZero, 270)
	canvas.Text(0, -5, "Invisible", fmt.Sprintf("text-anchor:top;font-size:%dpx;fill:black", fontSize))
	canvas.Text(yLength-50, -5, "Visible", fmt.Sprintf("text-anchor:top;font-size:%dpx;fill:black", fontSize))
	canvas.Text(yLength-100, fontSize+2+5, "Value Chain", fmt.Sprintf("text-anchor:left;font-size:%dpx;fill:black;font-weight:bold;font-family:serif", fontSize+2))
	canvas.Gend()
}
