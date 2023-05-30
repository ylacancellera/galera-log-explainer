package regex

import (
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/ylacancellera/galera-log-explainer/types"
)

func testActualGrepOnLog(t *testing.T, log string, regex *types.LogRegex) {

	f, err := ioutil.TempFile(t.TempDir(), "test_log")
	if err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}
	defer f.Sync()

	_, err = f.WriteString(log)
	if err != nil {
		t.Fatalf("failed to write in tmp file: %v", err)
	}
	m := types.RegexMap{"test": regex}

	out, err := exec.Command("grep", "-P", m.Compile()[0], f.Name()).Output()
	if err != nil {
		t.Fatalf("failed to grep in tmp file: %v, using: %s", err, regex.Regex.String())
	}
	if string(out) == "" {
		t.Errorf("empty results when grepping in tmp file: %v, using: %s", err, regex.Regex.String())
	}
}
