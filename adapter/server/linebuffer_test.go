package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestLineBufferKeepsMostRecentLines(t *testing.T) {
	b := &lineBuffer{size: 3}

	for i := 1; i <= 2; i++ {
		b.Append(fmt.Sprintf("%d\n", i))
	}
	var out strings.Builder
	b.WriteTo(&out)
	if out.String() != "1\n2\n" {
		t.Errorf("unexpected log %q", out.String())
	}

	for i := 3; i <= 7; i++ {
		b.Append(fmt.Sprintf("%d\n", i))
	}
	out.Reset()
	b.WriteTo(&out)
	if out.String() != "5\n6\n7\n" {
		t.Errorf("unexpected log %q", out.String())
	}
}
