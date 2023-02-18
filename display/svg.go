package display

import (
	"fmt"
	"os"
	"strings"

	svg "github.com/ajstarks/svgo"
	"github.com/ylacancellera/galera-log-explainer/types"
)

const (
	linestyle = "stroke:black; stroke-width:1"

	initY = 100
	initX = 280

	rectY     = 100
	rectX     = 300
	rectstyle = "style=\"stroke:grey; fill:white; stroke-width:1; cursor:crosshair\" "

	roundRY = 10
	roundRX = 10

	stepY = rectY + 50
	stepX = rectX + 50

	textSpacingX = 10
	textSpacingY = 15

	timestampX = 5

	headerY           = 20
	headerArrowInitY  = 5
	headerArrowHeight = 30
	headerArrowDepthY = 45
	headerArrowStyle  = "stroke:black; fill:white; stroke-width:3"
)

var (
	headerArrowYs = []int{headerArrowInitY, headerArrowInitY, headerArrowInitY, headerArrowInitY + headerArrowHeight, headerArrowDepthY + headerArrowHeight, headerArrowInitY + headerArrowHeight}
)

type svgnode struct {
	x, y      int
	id        uint64
	prevNode  *svgnode
	nodeident string
	li        *types.LogInfo
	latestCtx types.LogCtx
}

func (n *svgnode) draw(canvas *svg.SVG) {

	canvas.Text(timestampX, n.y+int(rectY/2), n.li.Date.DisplayTime)

	canvas.Roundrect(n.x, n.y, rectX, rectY, roundRX, roundRY, n.extras())
	y := n.y + (2 * textSpacingY)
	canvas.Text(n.x+textSpacingX, y, "type: "+string(n.li.RegexType))

	y += (2 * textSpacingY)
	canvas.Text(n.x+textSpacingX, y, n.li.Msg(n.latestCtx))

	y += (2 * textSpacingY)
	canvas.Text(n.x+textSpacingX, y, "click for details")
}

func (n *svgnode) extras() string {
	return strings.Join([]string{rectstyle, n.onclick(), n.dynamicMetadatas()}, " ")
}

func (n *svgnode) dynamicMetadatas() string {
	return fmt.Sprintf("id=\"%d\"", n.id)
}

func (n *svgnode) onclick() string {
	return fmt.Sprintf("onclick=\"scale(%d)\"", n.id)
}

func lineStartPointFromRectPos(x, y int) (int, int) {
	return int(x + (rectX / 2)), y + rectY
}

func Svg(timeline types.Timeline, verbosity types.Verbosity) {

	width := 3000
	height := initY
	for _, node := range timeline {
		height += len(node) * (rectY)
	}
	canvas := svg.New(os.Stdout)
	canvas.Start(width, height)

	canvas.Script("application/javascript", `

function scale(id){
	var elem = document.getElementById(id);
	let add = 20;
	var height = elem.getAttribute("height");
	elem.setAttribute("height", parseInt(height)+add);
	while (true) {
		id++
		var elem = document.getElementById(id);
		if (elem == null) {
			return
		}
		var y = elem.getAttribute("y");
		elem.setAttribute("y", parseInt(y)+add);
	}
}
`)

	latestCtxs := timeline.GetLatestUpdatedContextsByNodes()

	relativeX := map[string]int{}
	curSvgnodes := map[string]*svgnode{}
	x := initX
	for node := range timeline {
		curSvgnodes[node] = nil
		relativeX[node] = x
		canvas.Polygon([]int{x, x + (rectX / 2), x + rectX, x + rectX, x + (rectX / 2), x}, headerArrowYs, headerArrowStyle)
		canvas.Text(x+4, headerY, node)
		x += stepX
	}

	y := initY
	var id uint64
	for nextNodes := iterateNode(timeline); len(nextNodes) != 0; nextNodes = iterateNode(timeline) {
		for _, node := range nextNodes {

			nl := &timeline[node][0]
			if verbosity > nl.Verbosity && nl.Msg != nil {
				n := &svgnode{id: id, x: relativeX[node], y: y, li: nl, prevNode: curSvgnodes[node], nodeident: node, latestCtx: latestCtxs[node]}
				id++

				n.draw(canvas)

				y += rectY
				curSvgnodes[node] = n
			}

			timeline[node] = timeline[node][1:]
		}
	}

	canvas.End()
}
