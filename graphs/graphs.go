//go:build js && wasm
// +build js,wasm

package graphs

import (
	"github.com/Nigel2392/jsext-framework/graphs/charts"
	"github.com/Nigel2392/jsext-framework/graphs/options"
	"github.com/Nigel2392/jsext/canvas"
)

// CreateGraph creates a graph based on the options provided.
func CreateGraph(Canvas canvas.Canvas, opts options.GraphOptions) {
	switch opts.Type {
	case options.Bar:
		charts.Bar(Canvas, opts)
	case options.Line:
		charts.Line(Canvas, opts)
	case options.Pie:
		charts.Pie(Canvas, opts, false)
	case options.Donut:
		charts.Pie(Canvas, opts, true)
	default:
		panic("Invalid Graph Type")
	}
}
