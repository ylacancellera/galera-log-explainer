package regex

import (
	"testing"

	"github.com/ylacancellera/galera-log-explainer/types"
	"github.com/ylacancellera/galera-log-explainer/utils"
)

func TestRegexShutdownComplete(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2021-12-14T15:51:03.437968Z 0 [System] [MY-010910] [Server] /usr/sbin/mysqld: Shutdown complete (mysqld 8.0.23-14.1)  Percona XtraDB Cluster (GPL), Release rel14, Revision d3b9a1d, WSREP version 26.4.3.",
			expectedState: "CLOSED",
			expectedOut:   "shutdown complete",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexShutdownComplete"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexTerminated(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2023-03-20 17:03:22 140430087788288 [Note] WSREP: /opt/rh-mariadb102/root/usr/libexec/mysqld: Terminated.",
			expectedState: "CLOSED",
			expectedOut:   "terminated",
		},
		{
			log:           "2023-02-07T14:12:48.465651Z 8 [Note] WSREP: /usr/sbin/mysqld: Terminated.",
			expectedState: "CLOSED",
			expectedOut:   "terminated",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexTerminated"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexShutdownSignal(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2023-01-04T22:43:43.569686Z 0 [Note] [MY-000000] [WSREP] Received shutdown signal. Will sleep for 10 secs before initiating shutdown. pxc_maint_mode switched to SHUTDOWN",
			expectedState: "CLOSED",
			expectedOut:   "received shutdown",
		},
		{
			log:           "2023-03-20 16:28:06 139688443508480 [Note] /opt/rh-mariadb102/root/usr/libexec/mysqld (unknown): Normal shutdown",
			expectedState: "CLOSED",
			expectedOut:   "received shutdown",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexShutdownSignal"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexAborting(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2022-09-03T14:07:54.586014Z 0 [ERROR] [MY-010119] [Server] Aborting",
			expectedState: "CLOSED",
			expectedOut:   "ABORTING",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexAborting"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexWsrepLoad(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2022-10-10T19:10:22.018687Z 0 [Note] [MY-000000] [Galera] wsrep_load(): loading provider library '/usr/lib64/galera4/libgalera_smm.so'",
			expectedState: "OPEN",
			expectedOut:   "started(cluster)",
		},
		{
			log:           "2022-09-12T21:12:42.971174Z 0 [Note] [MY-000000] [Galera] wsrep_load(): loading provider library 'none'",
			expectedState: "OPEN",
			expectedOut:   "started(standalone)",
		},

		{
			log:           "2023-03-20 16:14:19 140557650536640 [Note] WSREP: wsrep_load(): loading provider library '/opt/rh-mariadb102/root/usr/lib64/galera/libgalera_smm.so'",
			expectedState: "OPEN",
			expectedOut:   "started(cluster)",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexWsrepLoad"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexWsrepRecovery(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2023-02-23T03:13:33.574136Z 3 [Note] [MY-000000] [Galera] Recovered position from storage: 7780bb61-87cf-11eb-b53b-6a7c64b0fee3:23506640",
			expectedState: "RECOVERY",
			expectedOut:   "wsrep recovery",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexWsrepRecovery"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexUnknownConf(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedOut string }{
		{
			log:         "2023-03-20T23:09:24.045425-05:00 0 [ERROR] unknown variable 'validate_password_length=8'",
			expectedOut: "unknown variable: validate_password_length=8",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexUnknownConf"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if msg != test.expectedOut {
			t.Errorf("out: %s, expected: %s", msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexAssertionFailure(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2023-01-09T18:20:20.669186Z 0 [ERROR] [MY-013183] [InnoDB] Assertion failure: btr0cur.cc:296:btr_page_get_prev(get_block->frame, mtr) == page_get_page_no(page) thread 139538894652992",
			expectedState: "CLOSED",
			expectedOut:   "ASSERTION FAILURE",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexAssertionFailure"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}

func TestRegexBindAddressAlreadyUsed(t *testing.T) {
	utils.SkipColor = true

	tests := []struct{ log, expectedState, expectedOut string }{
		{
			log:           "2023-05-06  5:06:12 47285568576576 [ERROR] WSREP: failed to open gcomm backend connection: 98: error while trying to listen 'tcp://0.0.0.0:4567?socket.non_blocking=1', asio error 'bind: Address already in use': 98 (Address already in use)",
			expectedState: "CLOSED",
			expectedOut:   "bind address already used",
		},
	}

	for _, test := range tests {
		ctx := types.NewLogCtx()
		ctx, displayer := EventsMap["RegexBindAddressAlreadyUsed"].Handle(ctx, test.log)
		msg := displayer(ctx)
		if ctx.State != test.expectedState || msg != test.expectedOut {
			t.Errorf("state: %s, expected: %s, out: %s, expected: %s", ctx.State, test.expectedState, msg, test.expectedOut)
			t.Fail()
		}
	}
}
