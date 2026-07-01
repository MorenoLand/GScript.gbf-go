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

func TestRecoverProfileCloneWithBlock(t *testing.T) {
	lines := recoverProfileCloneBlocks([]string{
		`  "Game_BomberMad_TutorialInGameTextProfile" = "Game_BomberMad_TutorialTextProfile";`,
		`  with ("Game_BomberMad_TutorialInGameTextProfile") {`,
		`    fontcolor = "white";`,
		`    if (!player.languagedomain in {"fr", "de"}) {`,
		`      fonttype = "adventure";`,
		`    }`,
		`  }`,
		`  addcontrol("Game_BomberMad_TutorialInGameTextProfile");`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, `with ("Game_BomberMad_TutorialInGameTextProfile")`) || !strings.Contains(got, `new GuiControlProfile("Game_BomberMad_TutorialInGameTextProfile")`) || !strings.Contains(got, `fonttype = "adventure";`) {
		t.Fatalf("profile with block not recovered:\n%s", got)
	}
}

func TestRecoverForwardGotoGuardWrapsIfChain(t *testing.T) {
	lines := recoverForwardGotoGuards([]string{
		`      if (graalversion >= 5.222) goto label_9737;`,
		`      if (client.bombermovespeed >= 1) {`,
		`        this.display.movestep = 0.5;`,
		`      }`,
		`      else if (client.bombermovespeed <= -1) {`,
		`        this.display.movestep = 0.25;`,
		`      }`,
		`      else {`,
		`        this.display.movestep = 0.333333333;`,
		`      }`,
		`    }`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "if (!(graalversion >= 5.222)) {") || !strings.Contains(got, "else if (client.bombermovespeed <= -1)") {
		t.Fatalf("forward guard if-chain not recovered:\n%s", got)
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

func TestRecoverForwardGotoGuardsNestedSimpleStatement(t *testing.T) {
	lines := recoverForwardGotoGuards([]string{
		`    if (client.disablesounds) goto label_12000;`,
		`    if (this.game.gamename == "SoulBubbles") {`,
		`      if (goright) {`,
		`        if (!newmenu in {21, 16}) goto label_11986;`,
		`        play("soulbubbles_select.wav");`,
		`      }`,
		`      if (newmenu != 0) goto label_11986;`,
		`      play("soulbubbles_back.wav");`,
		`    }`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `if (!(!newmenu in {21, 16})) {`) || !strings.Contains(got, `if (!(newmenu != 0)) {`) {
		t.Fatalf("nested forward goto guard:\n%s", got)
	}
}

func TestRecoverForwardGotoGuardsReturnStatement(t *testing.T) {
	lines := recoverForwardGotoGuards([]string{
		`    if (temp.stacksize <= 0) goto label_6250;`,
		`    return 0;`,
	})
	got := strings.Join(lines, "\n")
	want := "    if (!(temp.stacksize <= 0)) {\n      return 0;\n    }"
	if got != want {
		t.Fatalf("return forward goto guard:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverForwardGotoGuardsUsesMatchingBlock(t *testing.T) {
	lines := recoverForwardGotoGuards([]string{
		`    if (thiso.current.background != 0) goto label_6329;`,
		`    if (!(thiso.current.background != "")) {`,
		`      return thiso.current.background;`,
		`    }`,
		`  }`,
		`  else if (panel == "Animate2") {`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `if (!(thiso.current.background != 0)) {`) || !strings.Contains(got, `else if (panel == "Animate2") {`) {
		t.Fatalf("matching block guard:\n%s", got)
	}
}

func TestRecoverForwardGotoGuardsFixedPoint(t *testing.T) {
	lines := recoverForwardGotoGuardsFixedPoint([]string{
		`  if (panel == "Disguise2") {`,
		`    if (thiso.current.background != 0) goto label_6329;`,
		`    if (!(thiso.current.background != "")) {`,
		`      return thiso.current.background;`,
		`    }`,
		`  }`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `if (!(thiso.current.background != 0)) {`) {
		t.Fatalf("fixed point guard:\n%s", got)
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

func TestRecoverForwardIfGotoLoops(t *testing.T) {
	lines := recoverForwardIfGotoLoops([]string{
		`  temp.i = 0;`,
		`  if (temp.i < this.items.size()) {`,
		`    addcontrol(temp.i);`,
		`    temp.i += 1;`,
		`    goto label_5107;`,
		`  }`,
	})
	got := strings.Join(lines, "\n")
	want := "  for (temp.i = 0; temp.i < this.items.size(); temp.i += 1) {\n    addcontrol(temp.i);\n  }"
	if got != want {
		t.Fatalf("forward if goto loop:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecoverLoopGotoContinues(t *testing.T) {
	lines := recoverLoopGotoContinues([]string{
		`  for (temp.i = 0; temp.i < 7; temp.i += 1) {`,
		`    if (temp.i == 0) {`,
		`      if (temp.look == this.game.characterindex) goto label_10723;`,
		`    }`,
		`    if (temp.index in this.playerstatsindices) {`,
		`      goto label_10723;`,
		`    }`,
		`  }`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "if (temp.look == this.game.characterindex) {\n        continue;\n      }") || !strings.Contains(got, "continue;") {
		t.Fatalf("loop goto continue:\n%s", got)
	}
}

func TestRecoverInvertedIfGotoLoops(t *testing.T) {
	lines := recoverInvertedIfGotoLoops([]string{
		`      temp.i = 0;`,
		`      if (data.taker == 2) {`,
		`      }`,
		`      if (!(temp.i < 1)) {`,
		`        temp.object = new GuiShowImgCtrl() {`,
		`          profile = Game_Cards_BackProfile;`,
		`        }`,
		`      }`,
		`      addcontrol(temp.object);`,
		`      temp.i += 1;`,
		`      goto label_6953;`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "for (temp.i = 0; temp.i < 1; temp.i += 1)") || !strings.Contains(got, "addcontrol(temp.object);") {
		t.Fatalf("inverted if goto loop:\n%s", got)
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

func TestRecoverWhileLoopWithConditionSetupBackEdge(t *testing.T) {
	body := []string{
		"  temp.check = getSurroundBlocks();",
		"  for (temp.i in temp.blocksurround) {",
		"  }",
		"  temp.checked.add(temp.checksurround);",
		"  goto label_2917;",
	}
	lines, ok := recoverWhileLoop(body, "temp.checked.size() != temp.blocksurround.size()", 2928, 0)
	got := strings.Join(lines, "\n")
	if !ok || strings.Contains(got, "goto label_2917") || !strings.Contains(got, "while (temp.checked.size() != temp.blocksurround.size())") {
		t.Fatalf("condition-setup while not recovered (%v):\n%s", ok, got)
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

func TestRecoverForwardDispatchComputedSelector(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 5, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "bee"}},
		{addr: 2, op: opCall},
		{addr: 3, op: opPop},
		{addr: 4, op: opJmp, operand: &operand{number: 21, kind: "number"}},
		{addr: 5, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 6, op: opPushNumber, operand: &operand{number: 6, kind: "number"}},
		{addr: 7, op: opRandom},
		{addr: 8, op: opConvertToFloat},
		{addr: 9, op: opInt},
		{addr: 10, op: opCopy},
		{addr: 11, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 12, op: opEqual},
		{addr: 13, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 14, op: opCopy},
		{addr: 15, op: opPushNumber, operand: &operand{number: 1, kind: "number"}},
		{addr: 16, op: opEqual},
		{addr: 17, op: opJeq, operand: &operand{number: 18, kind: "number"}},
		{addr: 18, op: opPushVariable, operand: &operand{str: "fly"}},
		{addr: 19, op: opCall},
		{addr: 20, op: opPop},
		{addr: 21, op: opRet},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "temp.switchvalue = int(random(0, 6));") || !strings.Contains(got, "if (temp.switchvalue == 0)") || !strings.Contains(got, "else if (temp.switchvalue == 1)") {
		t.Fatalf("computed selector dispatch:\n%s", got)
	}
}

func TestRecoverForwardDispatchInfixSelector(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 5, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "out"}},
		{addr: 2, op: opPushString, operand: &operand{str: "record", kind: "string"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "state"}},
		{addr: 6, op: opPushNumber, operand: &operand{number: 7, kind: "number"}},
		{addr: 7, op: opBitwiseAnd},
		{addr: 8, op: opCopy},
		{addr: 9, op: opPushNumber, operand: &operand{number: 2, kind: "number"}},
		{addr: 10, op: opEqual},
		{addr: 11, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 12, op: opPushVariable, operand: &operand{str: "out"}},
		{addr: 13, op: opPushString, operand: &operand{str: "idle", kind: "string"}},
		{addr: 14, op: opAssign},
		{addr: 15, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 16, op: opJmp, operand: &operand{number: 12, kind: "number"}},
		{addr: 17, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "if (state & 7 == 2)") || !strings.Contains(got, `out = "record";`) {
		t.Fatalf("infix selector dispatch:\n%s", got)
	}
}

func TestRecoverForwardDispatchSubstringSelector(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 5, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "x"}},
		{addr: 2, op: opPushString, operand: &operand{str: "Top", kind: "string"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opJmp, operand: &operand{number: 21, kind: "number"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "obj"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "buttonname"}},
		{addr: 7, op: opAccessMember},
		{addr: 8, op: opConvertToString},
		{addr: 9, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 10, op: opPushNumber, operand: &operand{number: 1, kind: "number"}},
		{addr: 11, op: opObjSubstring},
		{addr: 12, op: opCopy},
		{addr: 13, op: opPushString, operand: &operand{str: "T", kind: "string"}},
		{addr: 14, op: opEqual},
		{addr: 15, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 16, op: opPushVariable, operand: &operand{str: "x"}},
		{addr: 17, op: opPushString, operand: &operand{str: "Width", kind: "string"}},
		{addr: 18, op: opAssign},
		{addr: 19, op: opJmp, operand: &operand{number: 21, kind: "number"}},
		{addr: 20, op: opJmp, operand: &operand{number: 16, kind: "number"}},
		{addr: 21, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `temp.switchvalue = obj.buttonname.substring(0, 1);`) || !strings.Contains(got, `if (temp.switchvalue == "T")`) || !strings.Contains(got, `x = "Top";`) {
		t.Fatalf("substring selector dispatch:\n%s", got)
	}
}

func TestRecoverTailDispatchSkipsNoOpCases(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "open.wav", kind: "string"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "play"}},
		{addr: 2, op: opCall},
		{addr: 3, op: opPop},
		{addr: 4, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 5, op: opPushString, operand: &operand{str: "win.wav", kind: "string"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "play"}},
		{addr: 7, op: opCall},
		{addr: 8, op: opPop},
		{addr: 9, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 10, op: opPushVariable, operand: &operand{str: "sound"}},
		{addr: 11, op: opCopy},
		{addr: 12, op: opPushString, operand: &operand{str: "play", kind: "string"}},
		{addr: 13, op: opEqual},
		{addr: 14, op: opJeq, operand: &operand{number: 17, kind: "number"}},
		{addr: 15, op: opCopy},
		{addr: 16, op: opPushString, operand: &operand{str: "windowopen", kind: "string"}},
		{addr: 17, op: opEqual},
		{addr: 18, op: opJeq, operand: &operand{number: 0, kind: "number"}},
		{addr: 19, op: opCopy},
		{addr: 20, op: opPushString, operand: &operand{str: "win", kind: "string"}},
		{addr: 21, op: opEqual},
		{addr: 22, op: opJeq, operand: &operand{number: 5, kind: "number"}},
		{addr: 23, op: opPop},
		{addr: 24, op: opRet},
	}
	lines := decompileRange(code, 10, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `if (sound == "windowopen")`) || !strings.Contains(got, `else if (sound == "win")`) {
		t.Fatalf("tail dispatch no-op cases:\n%s", got)
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

func TestDispatchCommonEndUsesFirstCaseExit(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 10, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "a"}},
		{addr: 2, op: opJmp, operand: &operand{number: 20, kind: "number"}},
		{addr: 3, op: opJmp, operand: &operand{number: 30, kind: "number"}},
		{addr: 10, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 11, op: opCopy},
		{addr: 12, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 13, op: opEqual},
		{addr: 14, op: opJeq, operand: &operand{number: 1, kind: "number"}},
	}
	commonEnd, ok := dispatchCommonEnd(code, []dispatchCase{{condition: `mode == "a"`, target: 1}}, 10)
	if !ok || commonEnd != 20 {
		t.Fatalf("commonEnd = %d, %v; want 20, true", commonEnd, ok)
	}
}

func TestRecoverTailDispatchAtSelector(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 1, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 2, op: opAssign},
		{addr: 3, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 4, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 5, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 6, op: opAssign},
		{addr: 7, op: opJmp, operand: &operand{number: 17, kind: "number"}},
		{addr: 8, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 9, op: opCopy},
		{addr: 10, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 11, op: opEqual},
		{addr: 12, op: opJeq, operand: &operand{number: 0, kind: "number"}},
		{addr: 13, op: opCopy},
		{addr: 14, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 15, op: opEqual},
		{addr: 16, op: opJeq, operand: &operand{number: 4, kind: "number"}},
		{addr: 17, op: opPop},
	}
	lines := decompileRange(code, 8, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, `if (mode == "a")`) || !strings.Contains(got, `else if (mode == "b")`) {
		t.Fatalf("tail dispatch not recovered:\n%s", got)
	}
}

func TestRecoverBackwardDispatchSkipsLeadingDefaultCase(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 9, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 2, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opJmp, operand: &operand{number: 22, kind: "number"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 6, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 7, op: opAssign},
		{addr: 8, op: opJmp, operand: &operand{number: 22, kind: "number"}},
		{addr: 9, op: opPushVariable, operand: &operand{str: "mode"}},
		{addr: 10, op: opCopy},
		{addr: 11, op: opPushString, operand: &operand{str: "none", kind: "string"}},
		{addr: 12, op: opEqual},
		{addr: 13, op: opJeq, operand: &operand{number: 22, kind: "number"}},
		{addr: 14, op: opCopy},
		{addr: 15, op: opPushString, operand: &operand{str: "a", kind: "string"}},
		{addr: 16, op: opEqual},
		{addr: 17, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 18, op: opCopy},
		{addr: 19, op: opPushString, operand: &operand{str: "b", kind: "string"}},
		{addr: 20, op: opEqual},
		{addr: 21, op: opJeq, operand: &operand{number: 5, kind: "number"}},
		{addr: 22, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || strings.Contains(got, `"none"`) || !strings.Contains(got, `if (mode == "a")`) || !strings.Contains(got, `else if (mode == "b")`) {
		t.Fatalf("leading default dispatch not recovered:\n%s", got)
	}
}

func TestRecoverBackwardDispatchWithNumericCases(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 9, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 2, op: opPushString, operand: &operand{str: "six", kind: "string"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opJmp, operand: &operand{number: 18, kind: "number"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "result"}},
		{addr: 6, op: opPushString, operand: &operand{str: "nine", kind: "string"}},
		{addr: 7, op: opAssign},
		{addr: 8, op: opJmp, operand: &operand{number: 16, kind: "number"}},
		{addr: 9, op: opPushVariable, operand: &operand{str: "level"}},
		{addr: 10, op: opCopy},
		{addr: 11, op: opPushNumber, operand: &operand{number: 6, kind: "number"}},
		{addr: 12, op: opEqual},
		{addr: 13, op: opJeq, operand: &operand{number: 1, kind: "number"}},
		{addr: 14, op: opCopy},
		{addr: 15, op: opPushNumber, operand: &operand{number: 9, kind: "number"}},
		{addr: 16, op: opEqual},
		{addr: 17, op: opJeq, operand: &operand{number: 5, kind: "number"}},
		{addr: 18, op: opPop},
	}
	lines := decompileRange(code, 0, len(code), 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "goto label_") || !strings.Contains(got, "if (level == 6)") || !strings.Contains(got, "else if (level == 9)") {
		t.Fatalf("numeric backward dispatch not recovered:\n%s", got)
	}
}

func TestRecoverConditionalAssignmentChain(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opTemp},
		{addr: 1, op: opPushVariable, operand: &operand{str: "nextrow"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushVariable, operand: &operand{str: "score"}},
		{addr: 4, op: opPushNumber, operand: &operand{number: 10000, kind: "number"}},
		{addr: 5, op: opGE},
		{addr: 6, op: opJne, operand: &operand{number: 9, kind: "number"}},
		{addr: 7, op: opPushNumber, operand: &operand{number: 2, kind: "number"}},
		{addr: 8, op: opJmp, operand: &operand{number: 16, kind: "number"}},
		{addr: 9, op: opPushVariable, operand: &operand{str: "score"}},
		{addr: 10, op: opPushNumber, operand: &operand{number: 8000, kind: "number"}},
		{addr: 11, op: opGE},
		{addr: 12, op: opJne, operand: &operand{number: 15, kind: "number"}},
		{addr: 13, op: opPushNumber, operand: &operand{number: 3, kind: "number"}},
		{addr: 14, op: opJmp, operand: &operand{number: 16, kind: "number"}},
		{addr: 15, op: opPushNumber, operand: &operand{number: 7, kind: "number"}},
		{addr: 16, op: opAssign},
	}
	got := strings.Join(decompileRange(code, 0, len(code), 0), "\n")
	if strings.Contains(got, "goto label_") || strings.Contains(got, "if (score >= 10000) {\n}") || !strings.Contains(got, "temp.nextrow = 2;") || !strings.Contains(got, "else if (score >= 8000)") || !strings.Contains(got, "temp.nextrow = 7;") {
		t.Fatalf("conditional assignment chain not recovered:\n%s", got)
	}
}

func TestRecoverForLoopConvertsInternalBackEdgeToContinue(t *testing.T) {
	lines := []string{"i = 0;"}
	body := []string{
		"  if (skip) {",
		"    goto label_4;",
		"  }",
		"  use(i);",
		"  i += 1;",
		"  goto label_4;",
	}
	gotLines, ok := recoverForLoop(lines, body, "i < count", 5, 0)
	got := strings.Join(gotLines, "\n")
	if !ok || strings.Contains(got, "goto label_") || !strings.Contains(got, "continue;") {
		t.Fatalf("for-loop continue not recovered (%v):\n%s", ok, got)
	}
}

func TestJmpToRangeEndStopsBodyLeak(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 4, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "leaked"}},
		{addr: 2, op: opPushString, operand: &operand{str: "bad", kind: "string"}},
		{addr: 3, op: opAssign},
		{addr: 4, op: opPushVariable, operand: &operand{str: "kept"}},
		{addr: 5, op: opPushString, operand: &operand{str: "ok", kind: "string"}},
		{addr: 6, op: opAssign},
	}
	lines := decompileRange(code, 0, 4, 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "leaked") || strings.Contains(got, "goto label_") {
		t.Fatalf("range-end jump leaked body:\n%s", got)
	}
}

func TestForwardJumpPaddingIsSkipped(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opJmp, operand: &operand{number: 2, kind: "number"}},
		{addr: 1, op: opJmp, operand: &operand{number: 3, kind: "number"}},
		{addr: 2, op: opPushVariable, operand: &operand{str: "done"}},
		{addr: 3, op: opPop},
	}
	got := strings.Join(decompileRange(code, 0, len(code), 0), "\n")
	if strings.Contains(got, "goto label_") {
		t.Fatalf("jump padding emitted goto:\n%s", got)
	}
}

func TestDynamicMemberNameIsParenthesized(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opThis},
		{addr: 1, op: opPushString, operand: &operand{str: "game_board", kind: "string"}},
		{addr: 2, op: opPushVariable, operand: &operand{str: "board"}},
		{addr: 3, op: opJoin},
		{addr: 4, op: opPushString, operand: &operand{str: "_load", kind: "string"}},
		{addr: 5, op: opJoin},
		{addr: 6, op: opPushNumber, operand: &operand{number: 1, kind: "number"}},
		{addr: 7, op: opAssignMember},
	}
	got := strings.Join(decompileRange(code, 0, len(code), 0), "\n")
	if !strings.Contains(got, `this.("game_board" @ board @ "_load") = 1;`) {
		t.Fatalf("dynamic member not parenthesized:\n%s", got)
	}
}

func TestDynamicCallTargetIsParenthesized(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushArray},
		{addr: 1, op: opPushVariable, operand: &operand{str: "value"}},
		{addr: 2, op: opPushString, operand: &operand{str: "faceezSet", kind: "string"}},
		{addr: 3, op: opPushVariable, operand: &operand{str: "part"}},
		{addr: 4, op: opJoin},
		{addr: 5, op: opCall},
		{addr: 6, op: opPop},
	}
	got := strings.Join(decompileRange(code, 0, len(code), 0), "\n")
	if !strings.Contains(got, `("faceezSet" @ part)(value);`) {
		t.Fatalf("dynamic call target not parenthesized:\n%s", got)
	}
}

func TestRecoverGenericWithBlock(t *testing.T) {
	code := []instruction{
		{addr: 0, op: opPushNumber, operand: &operand{number: 200, kind: "number"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "findimg"}},
		{addr: 2, op: opCall},
		{addr: 3, op: opWith, operand: &operand{number: 11, kind: "number"}},
		{addr: 4, op: opPushVariable, operand: &operand{str: "emitter"}},
		{addr: 5, op: opWith, operand: &operand{number: 10, kind: "number"}},
		{addr: 6, op: opPushVariable, operand: &operand{str: "delaymin"}},
		{addr: 7, op: opPushNumber, operand: &operand{number: 1, kind: "number"}},
		{addr: 8, op: opAssign},
		{addr: 9, op: opWithEnd},
		{addr: 10, op: opWithEnd},
	}
	got := strings.Join(decompileRange(code, 0, len(code), 0), "\n")
	if !strings.Contains(got, "with (findimg(200)) {") || !strings.Contains(got, "with (emitter) {") || !strings.Contains(got, "delaymin = 1;") {
		t.Fatalf("generic with block not recovered:\n%s", got)
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

func TestAnonymousGuiAssignmentConstructorClosesWithSemicolon(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "gamepanel"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushVariable, operand: &operand{str: "unknown_object"}},
		{addr: 4, op: opPushString, operand: &operand{str: "GuiStretchCtrl", kind: "string"}},
		{addr: 5, op: opNewObject},
		{addr: 6, op: opAssign},
		{addr: 7, op: opConvertToObject},
		{addr: 8, op: opWith, operand: &operand{number: 12, kind: "number"}},
		{addr: 9, op: opPushVariable, operand: &operand{str: "clientwidth"}},
		{addr: 10, op: opPushNumber, operand: &operand{number: 660, kind: "number"}},
		{addr: 11, op: opAssign},
	}, 0, 12, 0)

	got := strings.Join(lines, "\n")
	want := "this.gamepanel = new GuiStretchCtrl() {\n  clientwidth = 660;\n};"
	if got != want {
		t.Fatalf("anonymous assignment constructor:\n%s\nwant:\n%s", got, want)
	}
}

func TestPendingObjectCallStatementIsEmitted(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "data"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushNumber, operand: &operand{number: 1, kind: "number"}},
		{addr: 4, op: opArrayAccess},
		{addr: 5, op: opPushVariable, operand: &operand{str: "block"}},
		{addr: 6, op: opEqual},
		{addr: 7, op: opJne, operand: &operand{number: 13, kind: "number"}},
		{addr: 8, op: opPushVariable, operand: &operand{str: "temp"}},
		{addr: 9, op: opPushVariable, operand: &operand{str: "surround"}},
		{addr: 10, op: opAccessMember},
		{addr: 11, op: opPushVariable, operand: &operand{str: "check_i"}},
		{addr: 12, op: opObjAddString},
	}, 0, 13, 0)

	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "temp.surround.add(check_i);") {
		t.Fatalf("pending object call:\n%s", got)
	}
}

func TestRecoverLoopEmptyIfContinues(t *testing.T) {
	lines := recoverLoopGotoContinues([]string{
		`  for (temp.i = 0; temp.i < this.count; temp.i += 1) {`,
		`    if (!(("Obj_" @ temp.i).visible)) {`,
		`    }`,
		`    doWork(temp.i);`,
		`  }`,
	})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "{\n    }") || !strings.Contains(got, "continue;") {
		t.Fatalf("empty if continue:\n%s", got)
	}
}

func TestRecoverTernaryAssignmentBranch(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "state"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "anibody"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushVariable, operand: &operand{str: "field"}},
		{addr: 4, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 5, op: opArrayAccess},
		{addr: 6, op: opConvertToFloat},
		{addr: 7, op: opJne, operand: &operand{number: 12, kind: "number"}},
		{addr: 8, op: opPushVariable, operand: &operand{str: "field"}},
		{addr: 9, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 10, op: opArrayAccess},
		{addr: 11, op: opJmp, operand: &operand{number: 16, kind: "number"}},
		{addr: 12, op: opPushVariable, operand: &operand{str: "defaults"}},
		{addr: 13, op: opPushString, operand: &operand{str: "Body"}},
		{addr: 14, op: opAccessMember},
		{addr: 15, op: opConvertToObject},
		{addr: 16, op: opAssign},
	}, 0, 17, 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "{\n}") || !strings.Contains(got, "state.anibody = field[0];") || !strings.Contains(got, "state.anibody = defaults.Body;") {
		t.Fatalf("ternary assignment:\n%s", got)
	}
}

func TestRecoverTernaryExpressionBranch(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "out"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "field"}},
		{addr: 2, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 3, op: opArrayAccess},
		{addr: 4, op: opJne, operand: &operand{number: 9, kind: "number"}},
		{addr: 5, op: opPushVariable, operand: &operand{str: "bitmap"}},
		{addr: 6, op: opPushString, operand: &operand{str: "Body"}},
		{addr: 7, op: opAccessMember},
		{addr: 8, op: opJmp, operand: &operand{number: 10, kind: "number"}},
		{addr: 9, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 10, op: opPushVariable, operand: &operand{str: "extra"}},
		{addr: 11, op: opBitwiseOr},
		{addr: 12, op: opAssign},
	}, 0, 13, 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "{\n}") || !strings.Contains(got, "out = (field[0] ? bitmap.Body : 0) | extra;") {
		t.Fatalf("ternary expression:\n%s", got)
	}
}

func TestRecoverSelfTernaryAssignmentBranch(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "state"}},
		{addr: 1, op: opPushString, operand: &operand{str: "facetransform"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushVariable, operand: &operand{str: "field"}},
		{addr: 4, op: opPushNumber, operand: &operand{number: 41, kind: "number"}},
		{addr: 5, op: opArrayAccess},
		{addr: 6, op: opJne, operand: &operand{number: 8, kind: "number"}},
		{addr: 7, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 8, op: opAssign},
	}, 0, 9, 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "{\n}") || !strings.Contains(got, "state.facetransform = field[41];") || !strings.Contains(got, "state.facetransform = 0;") {
		t.Fatalf("self ternary assignment:\n%s", got)
	}
}

func TestRecoverNestedTernaryAssignmentBranch(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "text"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "lang"}},
		{addr: 2, op: opPushString, operand: &operand{str: "en"}},
		{addr: 3, op: opEqual},
		{addr: 4, op: opJne, operand: &operand{number: 7, kind: "number"}},
		{addr: 5, op: opPushString, operand: &operand{str: "English"}},
		{addr: 6, op: opJmp, operand: &operand{number: 14, kind: "number"}},
		{addr: 7, op: opPushVariable, operand: &operand{str: "lang"}},
		{addr: 8, op: opPushString, operand: &operand{str: "de"}},
		{addr: 9, op: opEqual},
		{addr: 10, op: opJne, operand: &operand{number: 13, kind: "number"}},
		{addr: 11, op: opPushString, operand: &operand{str: "German"}},
		{addr: 12, op: opJmp, operand: &operand{number: 14, kind: "number"}},
		{addr: 13, op: opPushString, operand: &operand{str: "French"}},
		{addr: 14, op: opAssign},
	}, 0, 15, 0)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "{\n}") || !strings.Contains(got, `else if (lang == "de")`) || !strings.Contains(got, `text = "French";`) {
		t.Fatalf("nested ternary assignment:\n%s", got)
	}
}

func TestRecoverNonGuiConstructorTarget(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "SinglePlayerTable"}},
		{addr: 1, op: opPushString, operand: &operand{str: "TStaticVar"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
	}, 0, 4, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `SinglePlayerTable = new TStaticVar();`) || strings.Contains(got, `"SinglePlayerTable" = "TStaticVar";`) {
		t.Fatalf("non-gui constructor:\n%s", got)
	}
}

func TestConstructorAssignmentDoesNotAbsorbDifferentWithTarget(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "SinglePlayerTable"}},
		{addr: 1, op: opPushString, operand: &operand{str: "TStaticVar"}},
		{addr: 2, op: opNewObject},
		{addr: 3, op: opAssign},
		{addr: 4, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 5, op: opPushString, operand: &operand{str: "singletable"}},
		{addr: 6, op: opAccessMember},
		{addr: 7, op: opWith, operand: &operand{number: 11, kind: "number"}},
		{addr: 8, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 9, op: opPushString, operand: &operand{str: "_parent"}},
		{addr: 10, op: opAssignMember},
	}, 0, 11, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `SinglePlayerTable = new TStaticVar();`) || !strings.Contains(got, `with (this.singletable)`) || strings.Contains(got, `SinglePlayerTable = new TStaticVar() {`) {
		t.Fatalf("constructor absorbed unrelated with:\n%s", got)
	}
}

func TestNewObjectWithAssignmentReceiverKeepsReceiver(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 1, op: opPushString, operand: &operand{str: "singletable"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushString, operand: &operand{str: "SinglePlayerTable"}},
		{addr: 4, op: opNew},
		{addr: 5, op: opPushString, operand: &operand{str: "TStaticVar"}},
		{addr: 6, op: opNewObject},
		{addr: 7, op: opAssign},
	}, 0, 8, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `this.singletable = new TStaticVar("SinglePlayerTable");`) || strings.Contains(got, `SinglePlayerTable = new TStaticVar`) {
		t.Fatalf("new object receiver:\n%s", got)
	}
}

func TestNonGuiAssignmentConstructorKeepsExplicitWith(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 1, op: opPushString, operand: &operand{str: "singletable"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushString, operand: &operand{str: "SinglePlayerTable"}},
		{addr: 4, op: opNew},
		{addr: 5, op: opPushString, operand: &operand{str: "TStaticVar"}},
		{addr: 6, op: opNewObject},
		{addr: 7, op: opAssign},
		{addr: 8, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 9, op: opPushString, operand: &operand{str: "singletable"}},
		{addr: 10, op: opAccessMember},
		{addr: 11, op: opWith, operand: &operand{number: 15, kind: "number"}},
		{addr: 12, op: opPushVariable, operand: &operand{str: "this"}},
		{addr: 13, op: opPushString, operand: &operand{str: "_parent"}},
		{addr: 14, op: opAssignMember},
	}, 0, 15, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `this.singletable = new TStaticVar("SinglePlayerTable");`) || !strings.Contains(got, `with (this.singletable)`) || strings.Contains(got, `new TStaticVar("SinglePlayerTable") {`) {
		t.Fatalf("non-gui assignment constructor with:\n%s", got)
	}
}

func TestNamedGuiConstructionDropsRedundantAssignment(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "Accordion"}},
		{addr: 1, op: opPushString, operand: &operand{str: "Accordion"}},
		{addr: 2, op: opNew},
		{addr: 3, op: opPushString, operand: &operand{str: "GuiAccordionCtrl"}},
		{addr: 4, op: opNewObject},
		{addr: 5, op: opAssign},
	}, 0, 6, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `new GuiAccordionCtrl("Accordion");`) || strings.Contains(got, `"Accordion" = new`) {
		t.Fatalf("named gui constructor:\n%s", got)
	}
}

func TestDynamicNamedGuiConstructionDropsRedundantAssignment(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "Accordion_Panel"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "i"}},
		{addr: 2, op: opJoin},
		{addr: 3, op: opPushString, operand: &operand{str: "Accordion_Panel"}},
		{addr: 4, op: opPushVariable, operand: &operand{str: "i"}},
		{addr: 5, op: opJoin},
		{addr: 6, op: opNew},
		{addr: 7, op: opPushString, operand: &operand{str: "GuiControl"}},
		{addr: 8, op: opNewObject},
		{addr: 9, op: opAssign},
	}, 0, 10, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `new GuiControl("Accordion_Panel" @ i);`) || strings.Contains(got, `"Accordion_Panel" @ i = new`) {
		t.Fatalf("dynamic named gui constructor:\n%s", got)
	}
}

func TestNamedProfileCloneConstructionUsesGuiControlProfile(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushString, operand: &operand{str: "Game_Board_BigButtonProfile"}},
		{addr: 1, op: opPushString, operand: &operand{str: "Game_Board_BigButtonProfile"}},
		{addr: 2, op: opNew},
		{addr: 3, op: opPushString, operand: &operand{str: "Game_Board_ButtonProfile"}},
		{addr: 4, op: opNewObject},
		{addr: 5, op: opAssign},
	}, 0, 6, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `new GuiControlProfile("Game_Board_BigButtonProfile");`) || strings.Contains(got, `new Game_Board_ButtonProfile`) {
		t.Fatalf("profile clone constructor:\n%s", got)
	}
}

func TestRecoverSwappedBooleanAssignment(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "visible"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "vis"}},
		{addr: 2, op: opBoolAnd},
		{addr: 3, op: opPushVariable, operand: &operand{str: "showing"}},
		{addr: 4, op: opAssign},
	}, 0, 5, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "visible = visible && vis && showing;") || strings.Contains(got, "visible && vis = showing;") {
		t.Fatalf("swapped boolean assignment:\n%s", got)
	}
}

func TestRecoverEmbeddedOrAssignmentTarget(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "temp"}},
		{addr: 1, op: opPushString, operand: &operand{str: "v2"}},
		{addr: 2, op: opAccessMember},
		{addr: 3, op: opPushVariable, operand: &operand{str: "panel"}},
		{addr: 4, op: opPushString, operand: &operand{str: "Disguise2"}},
		{addr: 5, op: opEqual},
		{addr: 6, op: opBoolOr},
		{addr: 7, op: opPushVariable, operand: &operand{str: "panel"}},
		{addr: 8, op: opPushString, operand: &operand{str: "Animate2"}},
		{addr: 9, op: opEqual},
		{addr: 10, op: opAssign},
	}, 0, 11, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `temp.v2 = temp.v2 || panel == "Disguise2" || panel == "Animate2";`) {
		t.Fatalf("embedded or assignment:\n%s", got)
	}
}

func TestRecoverEmbeddedAndAssignmentTarget(t *testing.T) {
	lines := decompileRange([]instruction{
		{addr: 0, op: opPushVariable, operand: &operand{str: "visible"}},
		{addr: 1, op: opPushVariable, operand: &operand{str: "profilepicture"}},
		{addr: 2, op: opPushString, operand: &operand{str: ""}},
		{addr: 3, op: opNotEqual},
		{addr: 4, op: opBoolAnd},
		{addr: 5, op: opPushVariable, operand: &operand{str: "profilepicture"}},
		{addr: 6, op: opPushNumber, operand: &operand{number: 0, kind: "number"}},
		{addr: 7, op: opNotEqual},
		{addr: 8, op: opAssign},
	}, 0, 9, 0)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, `visible = visible && profilepicture != "" && profilepicture != 0;`) {
		t.Fatalf("embedded and assignment:\n%s", got)
	}
}
