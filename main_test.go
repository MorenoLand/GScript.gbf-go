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

func TestNewObjectKeepsNamedConstructorsAndAnonymousObjects(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "GuiTest", kind: "string"}},
		{addr: 1, op: opPushString, operand: &operand{str: "GuiWindowCtrl", kind: "string"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
		{addr: 4, op: opPushVariable, operand: &operand{str: "remotecontrol"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "options"}},
		{addr: 6, op: opAccessMember},
		{addr: 7, op: opPushVariable, operand: &operand{str: "unknown_object"}},
		{addr: 8, op: opPushString, operand: &operand{str: "TStaticVar", kind: "string"}},
		{addr: 9, op: opNewObject},
		{addr: 10, op: opAssign},
	}, 0, 11, 0)

	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `new GuiWindowCtrl("GuiTest");`) {
		t.Fatalf("named constructor was not preserved:\n%s", got)
	}
	if !strings.Contains(got, `remotecontrol.options = new TStaticVar();`) {
		t.Fatalf("anonymous object construction was not recovered:\n%s", got)
	}
}

func TestRecoverFormatAssignmentTarget(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushArray},
		{addr: 1, op: opPushVariable, operand: &operand{str: "ShopGlobal_ItemDescription"}},
		{addr: 2, op: opPushVariable, operand: &operand{str: "text"}},
		{addr: 3, op: opAccessMember},
		{addr: 4, op: opPushVariable, operand: &operand{str: "item"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "timeleft"}},
		{addr: 6, op: opAccessMember},
		{addr: 7, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 8, op: opLE},
		{addr: 9, op: opShortCircuitOr},
		{addr: 10, op: opPushString, operand: &operand{str: "%s", kind: "string"}},
		{addr: 11, op: opFormat},
		{addr: 12, op: opAssign},
	}, 0, 13, 0)

	got := strings.Join(lines, "\n")
	want := `ShopGlobal_ItemDescription.text = format(item.timeleft <= 0, "%s");`
	if got != want {
		t.Fatalf("format assignment:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnonymousGuiObjectDoesNotLeakUnknownObject(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "unknown_object"}},
		{addr: 1, op: opPushString, operand: &operand{str: "GuiBitmapCtrl", kind: "string"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
		{addr: 4, op: opConvertToObject},
		{addr: 5, op: opWith, operand: &operand{number: 9, kind: "number"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "bitmap"}},
		{addr: 7, op: opPushString, operand: &operand{str: "icon.png", kind: "string"}},
		{addr: 8, op: opAssign},
		{addr: 9, op: opPushArray},
		{addr: 10, op: opPushVariable, operand: &operand{str: "unknown_object"}},
		{addr: 11, op: opPushVariable, operand: &operand{str: "addcontrol"}},
		{addr: 12, op: opCall},
		{addr: 13, op: opPop},
	}, 0, 14, 0)

	got := strings.Join(lines, "\n")
	if strings.Contains(got, "unknown_object") || !strings.Contains(got, "temp.object") {
		t.Fatalf("anonymous gui object leaked placeholder:\n%s", got)
	}
}

func TestEmbeddedSyntheticFunctionRanges(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "parent"}},
		{addr: 1, op: opJmp, operand: &operand{number: 6, kind: "number"}},
		{addr: 2, op: opPushArray},
		{addr: 3, op: opEndParams},
		{addr: 4, op: opFunctionStart},
		{addr: 5, op: opRet},
		{addr: 6, op: opPushVariable, operand: &operand{str: "after"}},
		{addr: 7, op: opRet},
	}
	funcs := []functionDef{
		{name: "parent", addr: 0, bodyStart: 0},
		{name: "public.function_357_2", addr: 2, bodyStart: 5},
	}
	ranges := buildFunctionRanges(funcs, code)
	if ranges[0].end != len(code) {
		t.Fatalf("parent ended at %d, want %d", ranges[0].end, len(code))
	}
	if ranges[1].end != 6 {
		t.Fatalf("synthetic ended at %d, want 6", ranges[1].end)
	}
	if !skipsEmbeddedFunction(nestedFunctionRanges(ranges[0], ranges), 2, 6) {
		t.Fatalf("parent did not recognize embedded function jump")
	}
}

func TestDispatchSelectorUsesRecoveredRegisters(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opGetRegister, operand: &operand{number: 1, kind: "number"}},
		{addr: 1, op: opConvertToObject},
		{addr: 2, op: opPushVariable, operand: &operand{str: "title"}},
		{addr: 3, op: opAccessMember},
		{addr: 4, op: opCopy},
	}
	state := newDecompileState()
	state.registers[1] = expr{text: "obj"}

	selector, pos, ok := dispatchSelector(code, 0, state)
	if !ok || selector != "obj.title" || pos != 4 {
		t.Fatalf("dispatch selector = %q, %d, %v; want obj.title, 4, true", selector, pos, ok)
	}
}

func TestAssignmentStyleGuiConstructorOpensWithBlock(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "FileBrowser_Screen"}},
		{addr: 1, op: opPushString, operand: &operand{str: "GuiWindowCtrl", kind: "string"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
		{addr: 4, op: opConvertToObject},
		{addr: 5, op: opWith, operand: &operand{number: 9, kind: "number"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "text"}},
		{addr: 7, op: opPushString, operand: &operand{str: "File Browser", kind: "string"}},
		{addr: 8, op: opAssign},
	}, 0, 9, 0)

	got := strings.Join(lines, "\n")
	want := "new GuiWindowCtrl(\"FileBrowser_Screen\") {\n    text = \"File Browser\";\n}"
	if got != want {
		t.Fatalf("constructor block:\n%s\nwant:\n%s", got, want)
	}
}
