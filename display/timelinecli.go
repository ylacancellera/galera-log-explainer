package display

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	// regular tabwriter do not work with color, this is a forked versions that ignores color special characters
	"github.com/Ladicle/tabwriter"
	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

// TimelineCLI print a timeline to the terminal using tabulated format
// It will print header and footers, and dequeue the timeline chronologically
func TimelineCLI(timeline types.Timeline, verbosity types.Verbosity) {

	// to hold the current context for each node
	// "keys" is needed, because iterating over a map must give a different order each time
	// a slice keeps its order
	keys, currentContext := initKeysContext(timeline)           // currentcontext to follow when important thing changed
	latestContext := timeline.GetLatestUpdatedContextsByNodes() // so that we have fully updated context when we print
	lastContext := map[string]types.LogCtx{}                    // just to follow when important thing changed

	w := tabwriter.NewWriter(os.Stdout, 8, 8, 3, ' ', tabwriter.DiscardEmptyColumns)
	defer w.Flush()

	// header
	fmt.Fprintln(w, headerNodes(keys))
	fmt.Fprintln(w, headerFilePath(keys, currentContext))
	fmt.Fprintln(w, headerIP(keys, latestContext))
	fmt.Fprintln(w, headerName(keys, latestContext))
	fmt.Fprintln(w, separator(keys))

	var (
		args      []string // stuff to print
		linecount int
	)

	// as long as there is a next event to print
	for nextNodes := timeline.IterateNode(); len(nextNodes) != 0; nextNodes = timeline.IterateNode() {

		// Date column
		//formattedDate, tmpLastLayout := dateBlock(nextDate, lastDate, timeline[nextNodes[0]][0].DateLayout, lastLayout)
		date := timeline[nextNodes[0]][0].Date
		if date != nil {
			args = []string{date.DisplayTime}
		} else {
			args = []string{""}
		}

		displayedValue := 0

		// node values
		for _, node := range keys {

			if !utils.SliceContains(nextNodes, node) {
				// if there are no events, having a | is needed for tabwriter
				// A few color can also help highlighting how the node is doing
				args = append(args, utils.PaintForState("| ", currentContext[node].State))
				continue
			}
			nl := timeline[node][0]
			lastContext[node] = currentContext[node]
			currentContext[node] = nl.Ctx

			timeline.Dequeue(node)

			if verbosity > nl.Verbosity && nl.Msg != nil {
				args = append(args, nl.Msg(latestContext[node]))
				displayedValue++
			} else {
				args = append(args, utils.PaintForState("| ", nl.Ctx.State))
			}
		}

		if sep := transitionSeparator(keys, lastContext, currentContext); sep != "" {
			// reset current context, so that we avoid duplicating transitions
			// lastContext/currentContext is only useful for that anyway
			lastContext = map[string]types.LogCtx{}
			for k, v := range currentContext {
				lastContext[k] = v
			}
			// print transition
			fmt.Fprintln(w, sep)
		}

		// If line is not filled with default placeholder values
		if displayedValue == 0 {
			continue

		}

		// Print tabwriter line
		_, err := fmt.Fprintln(w, strings.Join(args, "\t")+"\t")
		if err != nil {
			log.Println("Failed to write a line", err)
		}
		linecount++
	}

	// footer
	// only having a header is not fast enough to read when there are too many lines
	if linecount >= 50 {
		fmt.Fprintln(w, separator(keys))
		fmt.Fprintln(w, headerNodes(keys))
		fmt.Fprintln(w, headerFilePath(keys, currentContext))
		fmt.Fprintln(w, headerIP(keys, currentContext))
		fmt.Fprintln(w, headerName(keys, currentContext))
	}
}

func initKeysContext(timeline types.Timeline) ([]string, map[string]types.LogCtx) {
	currentContext := map[string]types.LogCtx{}

	// keys will be used to access the timeline map with an ordered manner
	// without this, we would not print on the correct column as the order of a map is guaranteed to be random each time
	keys := make([]string, 0, len(timeline))
	for node := range timeline {
		keys = append(keys, node)
		if len(timeline[node]) > 0 {
			currentContext[node] = timeline[node][0].Ctx
		} else {
			// Avoid crashing, but not ideal: we could have a better default Ctx with filepath at least
			currentContext[node] = types.NewLogCtx()
		}
	}
	sort.Strings(keys)
	return keys, currentContext
}

func separator(keys []string) string {
	return " \t" + strings.Repeat(" \t", len(keys))
}

func headerNodes(keys []string) string {
	return "identifier\t" + strings.Join(keys, "\t") + "\t"
}

func headerFilePath(keys []string, ctxs map[string]types.LogCtx) string {
	header := "current path\t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok {
			header += ctx.FilePath + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

func headerIP(keys []string, ctxs map[string]types.LogCtx) string {
	header := "last known ip\t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok && len(ctx.OwnIPs) > 0 {
			header += ctx.OwnIPs[len(ctx.OwnIPs)-1] + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

func headerName(keys []string, ctxs map[string]types.LogCtx) string {
	header := "last known name\t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok && len(ctx.OwnNames) > 0 {
			header += ctx.OwnNames[len(ctx.OwnNames)-1] + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

type transition struct {
	s1, s2, changeType string
	ok                 bool
	summary            transitionSummary
}

type transitions struct {
	tests             []*transition
	transitionToPrint []*transition
	numberFound       int
}

type transitionSummary [4]string

const NumberOfPossibleTransition = 3

const RowPerTransitions = 4

// transactionSeparator is useful to highligh a change of context
// example, changing file
//   mysqld.log.2
//    (file path)
//           V
//   mysqld.log.1
// or a change of ip, node name, ...
func transitionSeparator(keys []string, oldctxs, ctxs map[string]types.LogCtx) string {

	ts := map[string]*transitions{}

	for _, node := range keys {
		ctx, ok1 := ctxs[node]
		oldctx, ok2 := oldctxs[node]

		ts[node] = &transitions{tests: []*transition{}}
		if ok1 && ok2 {
			ts[node].tests = append(ts[node].tests, &transition{s1: oldctx.FilePath, s2: ctx.FilePath, changeType: "file path"})

			if len(oldctx.OwnNames) > 0 && len(ctx.OwnNames) > 0 {
				ts[node].tests = append(ts[node].tests, &transition{s1: oldctx.OwnNames[len(oldctx.OwnNames)-1], s2: ctx.OwnNames[len(ctx.OwnNames)-1], changeType: "node name"})
			}
			if len(oldctx.OwnIPs) > 0 && len(ctx.OwnIPs) > 0 {
				ts[node].tests = append(ts[node].tests, &transition{s1: oldctx.OwnIPs[len(oldctx.OwnIPs)-1], s2: ctx.OwnIPs[len(ctx.OwnIPs)-1], changeType: "node ip"})
			}

		}

		ts[node].fillEmptyTransition()
		ts[node].iterate()
	}

	highestStackOfTransitions := 0

	for _, node := range keys {
		if ts[node].numberFound > highestStackOfTransitions {
			highestStackOfTransitions = ts[node].numberFound
		}
	}
	for _, node := range keys {
		ts[node].stackPrioritizeFound(highestStackOfTransitions)
	}

	out := "\t"
	for i := 0; i < highestStackOfTransitions; i++ {
		for row := 0; row < RowPerTransitions; row++ {
			for _, node := range keys {
				out += ts[node].transitionToPrint[i].summary[row]
			}
			if !(i == highestStackOfTransitions-1 && row == RowPerTransitions-1) { // unless last row
				out += "\n\t"
			}
		}
	}

	if out == "\t" {
		return ""
	}
	return out
}

func (ts *transitions) iterate() {

	for _, test := range ts.tests {

		test.summarizeIfDifferent()
		if test.ok {
			ts.numberFound++
		}
	}

}
func (ts *transitions) stackPrioritizeFound(height int) {
	for i, test := range ts.tests {
		// if at the right height
		if len(ts.tests)-i+len(ts.transitionToPrint) == height {
			ts.transitionToPrint = append(ts.transitionToPrint, ts.tests[i:]...)
		}
		if test.ok {
			ts.transitionToPrint = append(ts.transitionToPrint, test)
		}
	}
}

func (ts *transitions) fillEmptyTransition() {
	if len(ts.tests) == NumberOfPossibleTransition {
		return
	}
	for i := len(ts.tests); i < NumberOfPossibleTransition; i++ {
		ts.tests = append(ts.tests, &transition{s1: "", s2: "", changeType: ""})
	}

}

func (t *transition) summarizeIfDifferent() {
	if t.s1 != t.s2 {
		t.summary = [4]string{utils.Paint(utils.BrightBlueText, t.s1), utils.Paint(utils.BlueText, "("+t.changeType+")"), utils.Paint(utils.BrightBlueText, " V "), utils.Paint(utils.BrightBlueText, t.s2)}
		t.ok = true
	}
	for i := range t.summary {
		t.summary[i] = t.summary[i] + "\t"
	}
	return
}
