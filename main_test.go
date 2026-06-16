package main

import (
	"strings"
	"testing"
)

func mustHexBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := parseHexBytes(s)
	if err != nil {
		t.Fatalf("parse test hex: %v", err)
	}
	return b
}

func TestDecompileDataSkipsWeaponRecordPrefix(t *testing.T) {
	raw := mustHexBytes(t, `
		0000 0001 0000 0004 0000 0000 0000 0002
		0000 0000 0000 0003 0000 0033 6372 6561
		7465 6400 7669 6e65 7332 2e70 6e67 0073
		6574 696d 6700 6472 6177 6f76 6572 706c
		6179 6572 0064 6f6e 7462 6c6f 636b 0000
		0000 0400 0000 1c16 f000 2104 f310 1715
		f001 16f0 0206 2017 16f0 0306 2017 16f0
		0406 20
	`)
	wrapped := append([]byte(" ?weapon,-ScriptedRC,1,#\x8e\x9a[o'\x9f=\x8es"), raw...)

	out, err := decompileData(wrapped)
	if err != nil {
		t.Fatalf("decompile wrapped bytecode: %v", err)
	}
	for _, want := range []string{
		"if (created)",
		`setimg("vines2.png");`,
		"drawoverplayer();",
		"dontblock();",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}
