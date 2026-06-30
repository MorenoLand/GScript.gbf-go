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

func TestRecoverNewMultiDimArrayAssignmentTarget(t *testing.T) {
	for _, tc := range []struct {
		rhs  string
		want string
	}{
		{`new [this.board][new [this.boardwidth]][this.boardheight]`, `this.board = new [this.boardwidth][this.boardheight];`},
		{`new [new [this.fieldcorners][new [this.boardwidth + 1]][this.boardwidth + 1]][2]`, `this.fieldcorners = new [this.boardwidth + 1][this.boardwidth + 1][2];`},
	} {
		lhs, rhs, ok := recoverNewMultiDimAssignment(expr{text: "/* missing */"}, expr{text: tc.rhs})
		if !ok {
			t.Fatalf("did not recover %s", tc.rhs)
		}
		got := lhs.text + " = " + rhs.text + ";"
		if got != tc.want {
			t.Fatalf("multidim assignment:\n%s\nwant:\n%s", got, tc.want)
		}
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

func TestAndroidReferenceLowOpcodeRecovery(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "obj"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "name"}},
		{addr: 2, op: opPushString, operand: &operand{str: "value", kind: "string"}},
		{addr: 3, op: opAssignMember},
		{addr: 4, op: opPushVariable, operand: &operand{str: "out"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "a"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "b"}},
		{addr: 7, op: opBoolAnd},
		{addr: 8, op: opPushVariable, operand: &operand{str: "c"}},
		{addr: 9, op: opBoolOr},
		{addr: 10, op: opPushString, operand: &operand{str: "x", kind: "string"}},
		{addr: 11, op: opAppend},
		{addr: 12, op: opAssign},
	}, 0, 13, 0)

	got := strings.Join(lines, "\n")
	want := "obj.name = \"value\";\nout = a && b || c @ \"x\";"
	if got != want {
		t.Fatalf("android low opcode recovery:\n%s\nwant:\n%s", got, want)
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

func TestFunctionSignatureMovesVisibilityBeforeFunction(t *testing.T) {
	got := functionSignature("public.onActionClientSide", []string{"dataname", "dataval"})
	want := "public function onActionClientSide(dataname, dataval)"
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

func TestRecoverProfileCloneBlocks(t *testing.T) {
	lines := recoverProfileCloneBlocks([]string{
		`    "Game_ABCBrick_BigTextProfile" = "Game_ABCBrick_BackProfile";`,
		`    fontcolor = {0, 0, 0};`,
		`    fontsize = 24;`,
		`    addcontrol("Game_ABCBrick_BigTextProfile");`,
	})
	got := strings.Join(lines, "\n")
	want := "    new GuiControlProfile(\"Game_ABCBrick_BigTextProfile\") {\n      fontcolor = {0, 0, 0};\n      fontsize = 24;\n    }\n    addcontrol(\"Game_ABCBrick_BigTextProfile\");"
	if got != want {
		t.Fatalf("profile clone block:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverBareConstructorBlocks(t *testing.T) {
	lines := recoverBareConstructorBlocks([]string{
		`    new GuiControl(controlname);`,
		`    this.panel.gamescript = this;`,
		`    y = 0;`,
		`    x = y;`,
		`    new GuiStretchCtrl("Window") {`,
	})
	got := strings.Join(lines, "\n")
	want := "    new GuiControl(controlname) {\n      this.panel.gamescript = this;\n      y = 0;\n      x = y;\n    }\n    new GuiStretchCtrl(\"Window\") {"
	if got != want {
		t.Fatalf("bare constructor block:\n%s\nwant:\n%s", got, want)
	}
}

func TestRemoveDuplicateGotos(t *testing.T) {
	lines := removeDuplicateGotos([]string{
		"    goto label_331;",
		"    goto label_331;",
		"    if (x) goto label_331;",
		"    if (x) goto label_331;",
	})
	got := strings.Join(lines, "\n")
	want := "    goto label_331;\n    if (x) goto label_331;\n    if (x) goto label_331;"
	if got != want {
		t.Fatalf("duplicate goto cleanup:\n%s\nwant:\n%s", got, want)
	}
}

func TestRemoveRepeatedAssignmentRuns(t *testing.T) {
	lines := removeRepeatedAssignmentRuns([]string{
		"  y = 0;",
		"  x = y;",
		"  extent = parent.clientextent;",
		"  y = 0;",
		"  x = y;",
		"  extent = parent.clientextent;",
	})
	got := strings.Join(lines, "\n")
	want := "  y = 0;\n  x = y;\n  extent = parent.clientextent;"
	if got != want {
		t.Fatalf("assignment run cleanup:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardGotoGuards(t *testing.T) {
	lines := recoverForwardGotoGuards([]string{
		`    if (isObject("Game_ABCBrick_StartScreen_BackGround")) goto label_6486;`,
		`    Game_ABCBrick_StartScreen_BackGround.makeFirstResponder(true);`,
	})
	got := strings.Join(lines, "\n")
	want := "    if (!(isObject(\"Game_ABCBrick_StartScreen_BackGround\"))) {\n      Game_ABCBrick_StartScreen_BackGround.makeFirstResponder(true);\n    }"
	if got != want {
		t.Fatalf("forward goto guard:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForLoopWithAssignmentIncrement(t *testing.T) {
	lines, ok := recoverForLoop(
		[]string{"temp.i = 90;"},
		[]string{"    addcontrol(temp.i);", "    temp.i = temp.i + 30;", "    goto label_10;"},
		"temp.i <= 1200",
		20,
		0,
	)
	if !ok {
		t.Fatalf("loop was not recovered")
	}
	got := strings.Join(lines, "\n")
	want := "for (temp.i = 90; temp.i <= 1200; temp.i += 30) {\n    addcontrol(temp.i);\n}"
	if got != want {
		t.Fatalf("assignment increment loop:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverWhileLoopWithSleep(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushTrue},
		{addr: 1, op: opJne, operand: &operand{number: 8, kind: "number"}},
		{addr: 2, op: opPushVariable, operand: &operand{str: "done"}},
		{addr: 3, op: opLogicalNot},
		{addr: 4, op: opJne, operand: &operand{number: 5, kind: "number"}},
		{addr: 5, op: opPushNumber, operand: &operand{float: "0.05", kind: "float"}},
		{addr: 6, op: opSleep},
		{addr: 7, op: opJmp, operand: &operand{number: 0, kind: "number"}},
	}, 0, 8, 0)
	got := strings.Join(lines, "\n")
	want := "while (true) {\n  if (!done) {\n    break;\n  }\n  sleep(0.05);\n}"
	if got != want {
		t.Fatalf("while sleep loop:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverWhileLoopRejectsOuterBackJump(t *testing.T) {
	_, ok := recoverWhileLoop([]string{
		"  sleep(0.05);",
		"  goto label_10;",
	}, "waiting", 30, 0)
	if ok {
		t.Fatalf("outer backjump was recovered as local while")
	}
}

func TestRecoverSleepLoopBlocks(t *testing.T) {
	lines := recoverSleepLoopBlocks([]string{
		"  if (true) {",
		"    temp.waiting = false;",
		"    if (!temp.waiting) {",
		"      sleep(0.05);",
		"      goto label_156;",
		"    }",
		"  }",
	})
	got := strings.Join(lines, "\n")
	want := "  while (true) {\n    temp.waiting = false;\n    if (!temp.waiting) {\n      break;\n    }\n    sleep(0.05);\n  }"
	if got != want {
		t.Fatalf("sleep loop block:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardDispatch(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 5, kind: "number"}},
		{addr: 1, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 2, op: opRet},
		{addr: 3, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 4, op: opRet},
		{addr: 5, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 6, op: opCopy},
		{addr: 7, op: opPushString, operand: &operand{str: "one", kind: "string"}},
		{addr: 8, op: opEqual},
		{addr: 9, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 10, op: opJmp, operand: &operand{number: 3, kind: "number"}},
		{addr: 11, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	want := "if (mode == \"one\") {\n  return \"a\";\n}\nelse {\n  return \"b\";\n}"
	if got != want {
		t.Fatalf("forward dispatch:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardDispatchMultipleCases(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 7, kind: "number"}},
		{addr: 1, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 2, op: opRet},
		{addr: 3, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 4, op: opRet},
		{addr: 5, op: opPushString, operand: &operand{str: "c", kind: "string"}},
		{addr: 6, op: opRet},
		{addr: 7, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 8, op: opCopy},
		{addr: 9, op: opPushString, operand: &operand{str: "one", kind: "string"}},
		{addr: 10, op: opEqual},
		{addr: 11, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 12, op: opCopy},
		{addr: 13, op: opPushString, operand: &operand{str: "two", kind: "string"}},
		{addr: 14, op: opEqual},
		{addr: 15, op: opJeq, operand: &operand{number: 3, kind: "number"}},
		{addr: 16, op: opJmp, operand: &operand{number: 5, kind: "number"}},
		{addr: 17, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	want := "if (mode == \"one\") {\n  return \"a\";\n}\nelse if (mode == \"two\") {\n  return \"b\";\n}\nelse {\n  return \"c\";\n}"
	if got != want {
		t.Fatalf("forward multi dispatch:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardDispatchCaseAfterTable(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 3, kind: "number"}},
		{addr: 1, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 2, op: opRet},
		{addr: 3, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 4, op: opCopy},
		{addr: 5, op: opPushString, operand: &operand{str: "one", kind: "string"}},
		{addr: 6, op: opEqual},
		{addr: 7, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 8, op: opCopy},
		{addr: 9, op: opPushString, operand: &operand{str: "two", kind: "string"}},
		{addr: 10, op: opEqual},
		{addr: 11, op: opJeq, operand: &operand{number: 14, kind: "number"}},
		{addr: 12, op: opPop},
		{addr: 13, op: opRet},
		{addr: 14, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 15, op: opRet},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	want := "if (mode == \"one\") {\n  return \"a\";\n}\nelse if (mode == \"two\") {\n  return \"b\";\n}"
	if got != want {
		t.Fatalf("forward dispatch after table:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardDispatchKeepsLoopBodyWhole(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 11, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "i"}},
		{addr: 2, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opPushVariable, operand: &operand{str: "i"}},
		{addr: 5, op: opPushNumber, operand: &operand{number: 3, kind: "number"}},
		{addr: 6, op: opLessThan},
		{addr: 7, op: opJne, operand: &operand{number: 15, kind: "number"}},
		{addr: 8, op: opPushVariable, operand: &operand{str: "i"}},
		{addr: 9, op: opInc},
		{addr: 10, op: opJmp, operand: &operand{number: 4, kind: "number"}},
		{addr: 11, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 12, op: opCopy},
		{addr: 13, op: opPushString, operand: &operand{str: "loop", kind: "string"}},
		{addr: 14, op: opEqual},
		{addr: 15, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 16, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") {
		t.Fatalf("loop dispatch leaked goto:\n%s", got)
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

func TestDispatchSelectorSupportsFunctionCalls(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushArray},
		{addr: 1, op: opPushVariable, operand: &operand{str: "getPremiumOption"}},
		{addr: 2, op: opCall},
		{addr: 3, op: opCopy},
	}
	selector, pos, ok := dispatchSelector(code, 0, newDecompileState())
	if !ok || selector != "getPremiumOption()" || pos != 3 {
		t.Fatalf("dispatch selector = %q, %d, %v; want getPremiumOption(), 3, true", selector, pos, ok)
	}
}

func TestDispatchSelectorSupportsThisO(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opThisO},
		{addr: 1, op: opPushVariable, operand: &operand{str: "photobuttonsource"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opCopy},
	}
	selector, pos, ok := dispatchSelector(code, 0, newDecompileState())
	if !ok || selector != "thiso.photobuttonsource" || pos != 3 {
		t.Fatalf("dispatch selector = %q, %d, %v; want thiso.photobuttonsource, 3, true", selector, pos, ok)
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
	want := "new GuiWindowCtrl(\"FileBrowser_Screen\") {\n  text = \"File Browser\";\n}"
	if got != want {
		t.Fatalf("constructor block:\n%s\nwant:\n%s", got, want)
	}
}

func TestAssignmentStyleGuiConstructorKeepsVariableNameArgument(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "controlname"}},
		{addr: 1, op: opPushString, operand: &operand{str: "GuiControl", kind: "string"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
		{addr: 4, op: opConvertToObject},
		{addr: 5, op: opWith, operand: &operand{number: 9, kind: "number"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "alpha"}},
		{addr: 7, op: opPushNumber, operand: &operand{number: 10, kind: "number"}},
		{addr: 8, op: opAssign},
	}, 0, 9, 0)

	got := strings.Join(lines, "\n")
	want := "new GuiControl(controlname) {\n  alpha = 10;\n}"
	if got != want {
		t.Fatalf("constructor variable arg:\n%s\nwant:\n%s", got, want)
	}
}
