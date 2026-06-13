//go:build ignore

// Historical regression cases captured while reverse-engineering GS2 bytecode.
// This file is kept as reference material and is not part of normal `go test`.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleHex = `
0000 0001 0000 0004 0000 0000 0000 0002
0000 0000 0000 0003 0000 0033 6372 6561
7465 6400 7669 6e65 7332 2e70 6e67 0073
6574 696d 6700 6472 6177 6f76 6572 706c
6179 6572 0064 6f6e 7462 6c6f 636b 0000
0000 0400 0000 1c16 f000 2104 f310 1715
f001 16f0 0206 2017 16f0 0306 2017 16f0
0406 20`

func TestDecompilerEmitsDirectCallArguments(t *testing.T) {
	data, err := parseHexBytes(sampleHex)
	if err != nil {
		t.Fatalf("parse hex: %v", err)
	}

	module, err := parseModule(data)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got := strings.TrimSpace(decompileModule(module))
	want := strings.TrimSpace(`if (created)
{
    setimg("vines2.png");
    drawoverplayer();
    dontblock();
}`)

	if got != want {
		t.Fatalf("decompile mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestParseHexBytesAcceptsPowerShellUtf16PipeText(t *testing.T) {
	got, err := parseHexBytes("\xff\xfe0\x000\x000\x001\x00")
	if err != nil {
		t.Fatalf("parse utf16-ish hex: %v", err)
	}
	if len(got) != 2 || got[0] != 0x00 || got[1] != 0x01 {
		t.Fatalf("unexpected bytes: % x", got)
	}
}

func TestDecompilerSplitsNamedFunctions(t *testing.T) {
	data, err := os.ReadFile("weapon%045Adventure.gs2bc")
	if err != nil {
		t.Skipf("sample file not present: %v", err)
	}

	module, err := parseModule(data)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got := decompileModule(module)
	if !strings.Contains(got, "function onCreated()") {
		t.Fatalf("missing named function in output:\n%s", got[:min(len(got), 400)])
	}
	if strings.HasPrefix(strings.TrimSpace(got), "goto label_") {
		t.Fatalf("module output started with raw jump instead of function wrapper:\n%s", got[:min(len(got), 200)])
	}
	if strings.Contains(got, "goto label_966;") {
		t.Fatalf("module output contains synthetic module-tail jumps")
	}
	if strings.Contains(got, `"files" = "game";`) {
		t.Fatalf("array literal was misread as string assignment")
	}
	if !strings.Contains(got, `this.logtypes = {"sounds", "graphics", "net", "scripts", "files", "game"};`) {
		t.Fatalf("missing decoded logtypes array assignment")
	}
	for _, op := range []string{"0x1e", "0x1f", "0x2d", "0x2e", "0x2f", "0x34"} {
		if strings.Contains(got, "unhandled opcode "+op) {
			t.Fatalf("common VM plumbing opcode %s should not be emitted as unhandled", op)
		}
	}
	for _, op := range []string{"0x03", "0x28", "0x2a", "0x96", "0x97"} {
		if strings.Contains(got, "unhandled opcode "+op) {
			t.Fatalf("object/control opcode %s should not be emitted as unhandled", op)
		}
	}
	if strings.Contains(got, `"F2LogWindow_Tab" = "GuiTabCtrl";`) {
		t.Fatalf("object construction was misread as string assignment")
	}
	if strings.Contains(got, `new GuiTabCtrl("F2LogWindow_Tab");`) {
		t.Fatalf("with-object constructor should be emitted as a block, not a single statement")
	}
	if !strings.Contains(got, `new GuiTabCtrl("F2LogWindow_Tab") {`) {
		t.Fatalf("missing decoded GuiTabCtrl constructor block")
	}
	if strings.Contains(got, `new GuiBlueTabProfile(profile);`) {
		t.Fatalf("profile assignment was misread as constructor")
	}
	if !strings.Contains(got, `profile = "GuiBlueTabProfile";`) {
		t.Fatalf("missing profile assignment inside GuiTabCtrl block")
	}
	if strings.Contains(got, `"F2LogWindow_Scroll" @ temp.i = "GuiScrollCtrl";`) {
		t.Fatalf("dynamic object construction was misread as assignment")
	}
	if !strings.Contains(got, `new GuiScrollCtrl("F2LogWindow_Scroll" @ temp.i)`) {
		t.Fatalf("missing decoded dynamic GuiScrollCtrl constructor")
	}
	if strings.Contains(got, `"F2LogWindow_Scroll" @ selid.visible = true;`) {
		t.Fatalf("dynamic object member assignment needs parentheses")
	}
	if !strings.Contains(got, `("F2LogWindow_Scroll" @ selid).visible = true;`) {
		t.Fatalf("missing parenthesized dynamic object member assignment")
	}
	if strings.Contains(got, `"F2LogWindow_Text" @ selid.scrollToBottom();`) {
		t.Fatalf("dynamic object method call needs parentheses")
	}
	if !strings.Contains(got, `("F2LogWindow_Text" @ selid).scrollToBottom();`) {
		t.Fatalf("missing parenthesized dynamic object method call")
	}
	if strings.Contains(got, "if (temp.i < thiso.logtypes.size())\n        {") {
		t.Fatalf("simple counted loop should be recovered as for")
	}
	if !strings.Contains(got, `for (temp.i = 0; temp.i < thiso.logtypes.size(); temp.i += 1)`) {
		t.Fatalf("missing recovered temp.i for loop")
	}
	if !strings.Contains(got, `function F2LogWindow_Tab.onSelect(selid)`) {
		t.Fatalf("missing recovered onSelect parameter")
	}
	if !strings.Contains(got, `function F2LogWindow_Window.onKeyDown(keycode)`) {
		t.Fatalf("missing recovered keydown parameter")
	}
	if !strings.Contains(got, `function onLogMessage(msg, colred, colgreen, colblue, logtype)`) {
		t.Fatalf("missing recovered onLogMessage parameters")
	}
	if strings.Contains(got, `this.logtypes = logtype.index();`) {
		t.Fatalf("object index call was emitted with receiver/argument reversed")
	}
	if !strings.Contains(got, `this.logtypes.index(logtype)`) {
		t.Fatalf("missing decoded object index argument")
	}
	if strings.Contains(got, "\n    this.gotnewmessage;\n") || strings.Contains(got, "\n    temp.i;\n") {
		t.Fatalf("register setup leaked as standalone expression statements")
	}
	if strings.Contains(got, "reg0[reg1]") || strings.Contains(got, "reg1).") {
		t.Fatalf("register aliases leaked into onTimeout output")
	}
	if !strings.Contains(got, `for (temp.i = 0; temp.i < this.gotnewmessage.size(); temp.i += 1)`) {
		t.Fatalf("missing recovered onTimeout loop")
	}
	if !strings.Contains(got, `this.gotnewmessage[temp.i] = false;`) {
		t.Fatalf("missing recovered gotnewmessage array assignment")
	}
	if !strings.Contains(got, `if (this.gotnewmessage[temp.i] && ("F2LogWindow_Scroll" @ temp.i).isActuallyVisible())`) {
		t.Fatalf("missing folded onTimeout combined condition")
	}
	if strings.Contains(got, "    return 0;\n}\n\nfunction") || strings.HasSuffix(strings.TrimSpace(got), "return 0;\n}") {
		t.Fatalf("implicit terminal return 0 should be suppressed")
	}
	if !strings.Contains(got, "if (msg == \"\")\n    {\n        return 0;\n    }") {
		t.Fatalf("early return should remain visible")
	}
}

func TestDefaultOutputPathUsesInputFolderAndGs2Extension(t *testing.T) {
	got := defaultOutputPath(`G:\Development\Go\GByte\weapon%045Adventure.gs2bc`)
	want := `G:\Development\Go\GByte\weapon%045Adventure.gs2`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestSharedFixtureDoesNotEmitKnownUnhandledOpcodes(t *testing.T) {
	data, err := readInput("Shared.gs2bc")
	if err != nil {
		t.Skipf("sample file not present: %v", err)
	}

	module, err := parseModule(data)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got := decompileModule(module)
	for _, op := range []string{"0x40", "0x44", "0x77"} {
		if strings.Contains(got, "unhandled opcode "+op) {
			t.Fatalf("known opcode %s should not be emitted as unhandled", op)
		}
	}
}

func TestTriggerServerCallArgumentsUseVmPopOrder(t *testing.T) {
	data, err := readInput(filepath.Join("byte", "weapon%045New_IRC_Login3.gs2bc"))
	if err != nil {
		t.Skipf("stress fixture not present: %v", err)
	}
	module, err := parseModule(data)
	if err != nil {
		t.Fatalf("parse module: %v", err)
	}

	got := decompileModule(module)
	want := `triggerServer("gui", "-Serverlist", "sendPM", temp.msg);`
	if !strings.Contains(got, want) {
		t.Fatalf("missing correctly ordered triggerServer call %q", want)
	}
	if strings.Contains(got, `triggerServer(temp.msg, "sendPM", "-Serverlist", "gui");`) {
		t.Fatalf("triggerServer arguments are reversed")
	}
}

func TestClientExtentArrayPairIsWidthHeight(t *testing.T) {
	code := []instruction{
		pushVar("clientextent"),
		opOnly(opPushArray),
		pushNum(240),
		pushNum(600),
		opOnly(opEndArray),
		opOnly(opAssign),
	}

	got := strings.TrimSpace(strings.Join(decompileRange(code, 0, len(code), 0), "\n"))
	want := `clientextent = {600, 240};`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestDirectCallArgumentsUseVmPopOrder(t *testing.T) {
	code := []instruction{
		opOnly(opPushArray),
		pushVar("temp.msg"),
		pushString("sendPM"),
		pushString("-Serverlist"),
		pushString("gui"),
		pushVar("triggerServer"),
		opOnly(opCall),
		opOnly(opPop),
	}

	got := strings.TrimSpace(strings.Join(decompileRange(code, 0, len(code), 0), "\n"))
	want := `triggerServer("gui", "-Serverlist", "sendPM", temp.msg);`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestNotesOpcodesDecompileToReadableForms(t *testing.T) {
	tests := []struct {
		name string
		code []instruction
		want string
	}{
		{"pi", []instruction{opOnly(opPi), opOnly(opRet)}, "return pi;"},
		{"waitfor", []instruction{pushVar("someobject"), opOnly(opWaitFor), opOnly(opPop)}, "waitfor(someobject);"},
		{"power", []instruction{pushNum(2), pushNum(3), opOnly(opPower), opOnly(opRet)}, "return 2 ^ 3;"},
		{"unary subtract", []instruction{pushNum(7), opOnly(opUnarySubtract), opOnly(opRet)}, "return -7;"},
		{"bitwise xor", []instruction{pushNum(5), pushNum(1), opOnly(opBitwiseXor), opOnly(opRet)}, "return 5 ^ 1;"},
		{"bitwise invert", []instruction{pushNum(5), opOnly(opBitwiseInvert), opOnly(opRet)}, "return ~5;"},
		{"shift left", []instruction{pushNum(1), pushNum(3), opOnly(opShiftLeft), opOnly(opRet)}, "return 1 << 3;"},
		{"shift right", []instruction{pushNum(8), pushNum(1), opOnly(opShiftRight), opOnly(opRet)}, "return 8 >> 1;"},
		{"abs", []instruction{pushNum(-9), opOnly(opAbs), opOnly(opRet)}, "return abs(-9);"},
		{"arctan", []instruction{pushNum(45), opOnly(opArcTan), opOnly(opRet)}, "return arctan(45);"},
		{"exp", []instruction{pushNum(2), opOnly(opExp), opOnly(opRet)}, "return exp(2);"},
		{"log", []instruction{pushNum(10), opOnly(opLog), opOnly(opRet)}, "return log(10);"},
		{"getangle", []instruction{pushNum(10), pushNum(20), opOnly(opGetAngle), opOnly(opRet)}, "return getangle(10, 20);"},
		{"getdir", []instruction{pushNum(10), pushNum(20), opOnly(opGetDir), opOnly(opRet)}, "return getdir(10, 20);"},
		{"vecx", []instruction{pushNum(2), opOnly(opVecX), opOnly(opRet)}, "return vecx(2);"},
		{"vecy", []instruction{pushNum(2), opOnly(opVecY), opOnly(opRet)}, "return vecy(2);"},
		{"object indices", []instruction{pushVar("arr"), opOnly(opObjIndices), opOnly(opRet)}, "return arr.indices();"},
		{"object index", []instruction{pushVar("arr"), pushVar("value"), opOnly(opObjIndex), opOnly(opRet)}, "return arr.index(value);"},
		{"object subarray", []instruction{pushVar("arr"), pushNum(1), pushNum(3), opOnly(opObjSubArray), opOnly(opRet)}, "return arr.subarray(1, 3);"},
		{"replace string", []instruction{pushVar("arr"), pushVar("old"), pushVar("new"), opOnly(opObjReplaceString), opOnly(opPop)}, "arr.replace(old, new);"},
		{"multi dim access", []instruction{pushVar("arr"), pushNum(1), pushNum(2), opOnly(opMultiDimArray), opOnly(opRet)}, "return arr[1][2];"},
		{"multi dim assign", []instruction{pushVar("arr"), pushNum(1), pushNum(2), pushVar("value"), opOnly(opAssignMultiDimArray)}, "arr[1][2] = value;"},
		{"multi dim new", []instruction{pushNum(2), pushNum(3), opOnly(opNewMultiDimArray), opOnly(opRet)}, "return new [2][3];"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.TrimSpace(strings.Join(decompileRange(tt.code, 0, len(tt.code), 0), "\n"))
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
			if strings.Contains(got, "unhandled opcode") {
				t.Fatalf("unexpected unhandled opcode in %q", got)
			}
		})
	}
}

func TestDecompilerRecoversBackwardDispatchJumpTable(t *testing.T) {
	code := []instruction{
		jump(opJmp, 9),
		pushVar("x"),
		pushNum(1),
		opOnly(opAssign),
		jump(opJmp, 17),
		pushVar("x"),
		pushNum(2),
		opOnly(opAssign),
		jump(opJmp, 17),
		pushVar("textoption"),
		opOnly(opCopy),
		pushString("join"),
		opOnly(opEqual),
		jump(opJeq, 1),
		opOnly(opCopy),
		pushString("part"),
		opOnly(opEqual),
		jump(opJeq, 5),
		opOnly(opPop),
		pushNum(0),
		opOnly(opRet),
	}

	got := strings.TrimSpace(strings.Join(decompileRange(code, 0, len(code), 0), "\n"))
	want := strings.TrimSpace(`if (textoption == "join")
{
    x = 1;
}
else if (textoption == "part")
{
    x = 2;
}`)
	if got != want {
		t.Fatalf("decompile mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestBackwardJeqGotoKeepsPositiveCondition(t *testing.T) {
	code := []instruction{
		pushVar("text"),
		pushString("Open PM"),
		opOnly(opEqual),
		jump(opJeq, 0),
	}

	got := strings.TrimSpace(strings.Join(decompileRange(code, 0, len(code), 0), "\n"))
	want := `if (text == "Open PM") goto label_0;`
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestByteStressFolderHasNoUnhandledOpcodes(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("byte", "*.gs2bc"))
	if err != nil {
		t.Fatalf("glob stress fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Skip("stress fixture folder not present")
	}

	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat fixture: %v", err)
			}
			if info.Size() == 0 {
				return
			}

			data, err := readInput(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			module, err := parseModule(data)
			if err != nil {
				t.Fatalf("parse module: %v", err)
			}
			got := decompileModule(module)
			if strings.Contains(got, "unhandled opcode") {
				t.Fatalf("decompiled output contains unhandled opcode comments")
			}
		})
	}
}

func opOnly(op opcode) instruction {
	return instruction{op: op}
}

func pushNum(n int) instruction {
	return instruction{op: opPushNumber, operand: &operand{number: n, kind: "number"}}
}

func pushString(value string) instruction {
	return instruction{op: opPushString, operand: &operand{str: value, kind: "string"}}
}

func pushVar(name string) instruction {
	return instruction{op: opPushVariable, operand: &operand{str: name}}
}

func jump(op opcode, target int) instruction {
	return instruction{op: op, operand: &operand{number: target, kind: "number"}}
}
