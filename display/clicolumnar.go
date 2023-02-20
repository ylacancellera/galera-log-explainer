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

// DisplayColumnar is the main function to print
// It will print header and footers, and dequeue the timeline chronologically
func DisplayColumnar(timeline types.Timeline, verbosity types.Verbosity) {
	var args []string

	// to hold the current context for each node
	keys, currentContext := initKeysContext(timeline)
	latestContext := timeline.GetLatestUpdatedContextsByNodes()
	lastContext := map[string]types.LogCtx{}

	w := tabwriter.NewWriter(os.Stdout, 8, 8, 3, ' ', tabwriter.AlignRight)
	defer w.Flush()

	// header
	fmt.Fprintln(w, headerNodes(keys))
	fmt.Fprintln(w, headerFilePath(keys, currentContext))
	fmt.Fprintln(w, headerIP(keys, currentContext))
	fmt.Fprintln(w, separator(keys))

	// as long as there is a next event to print
	for nextNodes := iterateNode(timeline); len(nextNodes) != 0; nextNodes = iterateNode(timeline) {

		// Date column
		//formattedDate, tmpLastLayout := dateBlock(nextDate, lastDate, timeline[nextNodes[0]][0].DateLayout, lastLayout)
		date := timeline[nextNodes[0]][0].Date

		args = []string{date.DisplayTime}
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

			// dequeue the events
			if len(timeline[node]) > 0 {
				timeline[node] = timeline[node][1:]
			}

			if verbosity > nl.Verbosity && nl.Msg != nil {
				args = append(args, nl.Msg(latestContext[node]))
				displayedValue++
			} else {
				args = append(args, utils.PaintForState("| ", nl.Ctx.State))
			}
		}

		if sep := fileTransitionSeparator(keys, lastContext, currentContext); sep != "" {
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
	}

	// footer
	// only having a header is not fast enough to read when there are too many lines
	fmt.Fprintln(w, separator(keys))
	fmt.Fprintln(w, headerNodes(keys))
	fmt.Fprintln(w, headerFilePath(keys, currentContext))
	fmt.Fprintln(w, headerIP(keys, currentContext))
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
	header := "path\t"
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
	header := "ip\t"
	for _, node := range keys {
		if ctx, ok := ctxs[node]; ok && len(ctx.OwnIPs) > 0 {
			header += ctx.OwnIPs[len(ctx.OwnIPs)-1] + "\t"
		} else {
			header += " \t"
		}
	}
	return header
}

func fileTransitionSeparator(keys []string, oldctxs, ctxs map[string]types.LogCtx) string {
	sep1 := " \t"
	sep2 := " \t"
	sep3 := " \t"
	found := false
	for _, node := range keys {
		ctx, ok1 := ctxs[node]
		oldctx, ok2 := oldctxs[node]
		if ok1 && ok2 && ctx.FilePath != oldctx.FilePath {
			sep1 += utils.Paint(utils.BrightBlueText, oldctx.FilePath) + "\t"
			sep2 += utils.Paint(utils.BrightBlueText, " V ") + "\t"
			sep3 += utils.Paint(utils.BrightBlueText, ctx.FilePath) + "\t"
			found = true
		} else {
			sep1 += " \t"
			sep2 += " \t"
			sep3 += " \t"
		}
	}
	if !found {
		return ""
	}
	return sep1 + "\n" + sep2 + "\n" + sep3

}
