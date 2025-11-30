package log_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

func Test_LogInfo(t *testing.T) {
	tests := map[string]struct {
		level     log.Level
		logMethod log.Level
		message   string
		expect    string
	}{
		"info at debug": {
			level:     log.DebugLvl,
			logMethod: log.InfoLvl,
			message:   "write me out...",
			expect:    `level=INFO msg="write me out..."`,
		},
		"error at debug": {
			level:     log.DebugLvl,
			logMethod: log.ErrorLvl,
			message:   "write me out...",
			expect:    `level=ERROR msg="write me out..."`,
		},
		"debug at debug": {
			level:     log.DebugLvl,
			logMethod: log.DebugLvl,
			message:   "write me out...",
			expect:    `level=DEBUG msg="write me out..."`,
		},
		"debug at info": {
			level:     log.InfoLvl,
			logMethod: log.DebugLvl,
			message:   "write me out...",
			expect:    ``,
		},
		"info at error": {
			level:     log.ErrorLvl,
			logMethod: log.InfoLvl,
			message:   "write me out...",
			expect:    ``,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// overwrite writer for testing
			buf := &bytes.Buffer{}
			// overwrite logger for testing without timestamp
			logger := log.New(buf)
			logger.SetLevel(tt.level)

			switch tt.logMethod {
			case "debug":
				logger.Debug(tt.message)
			case "info":
				logger.Info(tt.message)
			case "error":
				logger.Error(tt.message)
			}
			got := buf.String()
			if !strings.Contains(got, tt.expect) {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
			if len(tt.expect) == 0 && len(got) > 0 {
				t.Errorf(testutils.TestPhraseWithContext, "no output expected", got, tt.expect)
			}
		})
	}
}
