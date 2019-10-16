package nfdb

import (
	"testing"

	"gotest.tools/assert"
)

func TestSQL(t *testing.T) {
	verify := func(in, out string) {
		actual := SQLCleanIDList(in)
		assert.Equal(t, out, actual)
	}

	verify("", "()")
	verify("1", "(1)")
	verify("1,2", "(1,2)")
	verify("a,2", "(2)")
	verify("a", "()")
	verify("1,,3", "(1,3)")
	verify(",", "()")
}
