package sixel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// from https://en.wikipedia.org/wiki/Sixel
func TestParsing(t *testing.T) {

	raw := `q
 #0;2;0;0;0#1;2;100;100;0#2;2;0;100;0
 #1~~@@vv@@~~@@~~$
 #2??}}GG}}??}}??-
 #1!14@`

	_, err := ParseString(raw)
	require.Nil(t, err)
}
