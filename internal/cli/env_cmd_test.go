package cli

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestParseSetArgs(t *testing.T) {
	tests := map[string]struct {
		args []string
		want []string
	}{
		"single pair":           {[]string{"APP_ENV=local"}, []string{"APP_ENV=local"}},
		"multi pairs":           {[]string{"A=1", "B=2", "C=3"}, []string{"A=1", "B=2", "C=3"}},
		"empty value":           {[]string{"A="}, []string{"A="}},
		"legacy name value":     {[]string{"APP_ENV", "local"}, []string{"APP_ENV=local"}},
		"legacy value with eq":  {[]string{"APP_OPTS", "A=1"}, []string{"APP_OPTS=A=1"}},
		"single bare name":      {[]string{"APP_ENV"}, []string{"APP_ENV"}},
		"three args not legacy": {[]string{"A", "B", "C"}, []string{"A", "B", "C"}},
		"first arg has eq":      {[]string{"A=1", "B"}, []string{"A=1", "B"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Eq(t, test.want, parseSetArgs(test.args))
		})
	}
}
