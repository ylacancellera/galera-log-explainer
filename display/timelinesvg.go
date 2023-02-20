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

	rectY       = 70
	rectX       = 300
	rectExpandY = 30
	rectstyle   = "style=\"stroke:grey; fill:white; stroke-width:1; cursor:crosshair\" "

	roundRY = 10
	roundRX = 10

	stepY = rectY + 50
	stepX = rectX + 50

	textSpacingX       = 10
	textSpacingY       = 15
	textSizePerChar    = 5
	textMainStyle      = " style=\"font-size:18px; font-weight:bold\""
	textRegexTypeStyle = " style=\"font-size:15px; fill:grey\""

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

	canvas.Group(n.typeID("group"), fmt.Sprintf("transform=\"translate(%d,%d)\"", n.x, n.y))
	canvas.Roundrect(0, 0, rectX, rectY, roundRX, roundRY, n.extras(rectstyle, n.onclick(), n.typeID("rect"), "data-expanded=\"no\""))
	canvas.Line(0, 16, rectX, 16, linestyle)

	n.text(canvas, string(n.li.RegexType), 14, n.extras(textRegexTypeStyle, n.onclick()))

	n.text(canvas, n.li.Msg(n.latestCtx), 40, n.extras(textMainStyle, n.onclick()))
	n.text(canvas, n.li.Log, 65, n.extras("visibility=\"hidden\"", n.typeID("detail")))

	canvas.Gend()
}

func (n *svgnode) text(canvas *svg.SVG, s string, y int, extras ...string) {
	canvas.Text(n.centerText(rectX, s), y, s, extras...)
}

func (n *svgnode) centerText(width int, s string) int {
	return (width / 2) - (len(s)*textSizePerChar + 10)
}

func (n *svgnode) extras(s ...string) string {
	return strings.Join(s, " ")
}

func (n *svgnode) typeID(t string) string {
	return fmt.Sprintf("id=\"%s%d\"", t, n.id)
}

func (n *svgnode) onclick() string {
	return fmt.Sprintf("onclick=\"expandBy(%d, %d)\"", n.id, rectExpandY)
}

func lineStartPointFromRectPos(x, y int) (int, int) {
	return int(x + (rectX / 2)), y + rectY
}

func TimelineSVG(timeline types.Timeline, verbosity types.Verbosity) {

	width := 3000
	height := initY
	for _, node := range timeline {
		height += len(node) * (rectY)
	}
	canvas := svg.New(os.Stdout)
	canvas.Start(width, height)

	canvas.Script("application/javascript", `

function expandBy(id, diff){
	diff = parseInt(diff)
	var elem = document.getElementById("rect"+id);
	var expanded = elem.getAttribute("data-expanded");
	if (expanded == "yes") {
		diff = -diff;
		elem.setAttribute("data-expanded", "no");
		detailsHide(id)
	} else {
		elem.setAttribute("data-expanded", "yes");
		detailsShow(id)
	}

	var height = elem.getAttribute("height");
	elem.setAttribute("height", parseInt(height)+diff);

	while (true) {
		id++
		var elem = document.getElementById("group"+id);
		if (elem == null) {
			return
		}
		groupMoveY(id, diff)
	}
}

function groupMoveY(id, diff){
	var elem = document.getElementById("group"+id);
	var xforms = elem.getAttribute("transform");
	var parts  = /translate\(\s*([^\s,)]+)[ ,]([^\s,)]+)/.exec(xforms);
	var firstX = parts[1], firstY = parts[2];

	var newY = parseInt(firstY) + parseInt(diff)

	elem.setAttribute("transform", "translate(" + firstX + "," + newY +")");
}

function detailsHide(id) {
	var elem = document.getElementById("detail"+id);
	elem.setAttribute("visibility", "hidden");
}

function detailsShow(id) {
	var elem = document.getElementById("detail"+id);
	elem.setAttribute("visibility", "visible");
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
