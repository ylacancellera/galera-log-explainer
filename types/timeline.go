package types

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

	// if t2 is an updated version of t1, or t1 an updated of t2, or t1=t2
	if t1[0].Date.Time.Equal(t2[0].Date.Time) {
		if t1[len(t1)-1].Date.Time.Before(t2[len(t2)-1].Date.Time) {
			return t2
		}
		return t1
	}
	if t1[0].Date.Time.Before(t2[0].Date.Time) {
		return append(t1, t2...)
	}
	return append(t2, t1...)
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
