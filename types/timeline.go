package types

import "time"

// It should be kept already sorted by timestamp
type LocalTimeline []LogInfo

// "string" key is a node IP
type Timeline map[string]LocalTimeline

// MergeTimeline is helpful when log files are split by date, it can be useful to be able to merge content
// a "timeline" come from a log file. Log files that came from some node should not never have overlapping dates
func MergeTimeline(t1, t2 LocalTimeline) LocalTimeline {
	if len(t1) == 0 {
		return t2
	}
	if len(t2) == 0 {
		return t1
	}

	startt1 := t1[0].Date.Time
	startt2 := t2[0].Date.Time

	// just flip them, easier than adding too many nested conditions
	// t1: ---O----?--
	// t2: --O-----?--
	if startt1.After(startt2) {
		return MergeTimeline(t2, t1)
	}

	endt1 := t1[len(t2)-1].Date.Time
	endt2 := t2[len(t2)-1].Date.Time

	// if t2 is an updated version of t1, or t1 an updated of t2, or t1=t2
	// t1: --O-----?--
	// t2: --O-----?--
	if startt1.Equal(startt2) {
		// t2 > t1
		// t1: ---O---O----
		// t2: ---O-----O--
		if endt1.Before(endt2) {
			return t2
		}
		// t1: ---O-----O--
		// t2: ---O-----O--
		// or
		// t1: ---O-----O--
		// t2: ---O---O----
		return t1
	}

	// if t1 superseds t2
	// t1: --O-----O--
	// t2: ---O---O---
	// or
	// t1: --O-----O--
	// t2: ---O----O--
	if endt1.After(endt2) || endt1.Equal(endt2) {
		return t1
	}
	//return append(t1, t2...)

	// t1: --O----O----
	// t2: ----O----O--
	if endt1.After(startt2) {
		// t1: --O----O----
		// t2: ----OO--OO--
		//>t : --O----OOO-- won't try to get things between t1.end and t2.start
		// we assume they're identical, they're supposed to be from the same server
		return append(t1, CutTimelineAt(t2, endt1)...)
	}

	// t1: --O--O------
	// t2: ------O--O--
	return append(t1, t2...)
}

// CutTimelineAt returns a localtimeline with the 1st event starting
// right after the time sent as parameter
func CutTimelineAt(t LocalTimeline, at time.Time) LocalTimeline {
	var i int
	for i = 0; i < len(t); i++ {
		if t[i].Date.Time.After(at) {
			break
		}
	}

	return t[i:]
}

func (t *Timeline) GetLatestUpdatedContextsByNodes() map[string]LogCtx {
	updatedCtxs := map[string]LogCtx{}
	latestctxs := []LogCtx{}

	for key, localtimeline := range *t {
		if len(localtimeline) == 0 {
			updatedCtxs[key] = NewLogCtx()
			continue
		}
		latestctx := localtimeline[len(localtimeline)-1].Ctx
		latestctxs = append(latestctxs, latestctx)
		updatedCtxs[key] = latestctx
	}

	for _, ctx := range updatedCtxs {
		ctx.MergeMapsWith(latestctxs)
	}
	return updatedCtxs
}
