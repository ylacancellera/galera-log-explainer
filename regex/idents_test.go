package regex

import (
	"testing"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func TestRegexMemberCount(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedOut string }{
		{
			log:         "  members(1):",
			expectedOut: "view member count: 1",
		},
	}

	for _, test := range tests {
		testActualGrepOnLog(t, test.log, IdentsMap["RegexMemberCount"])

		ctx := types.NewLogCtx()
		ctx, displayer := IdentsMap["RegexMemberCount"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if msg != test.expectedOut {
			t.Errorf("out: %s, expected: %s", msg, test.expectedOut)
			t.Fail()
		}
	}
}
