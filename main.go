package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type opcode byte

const (
	opNone                opcode = 0x00
	opJmp                 opcode = 0x01
	opJeq                 opcode = 0x02
	opShortCircuitOr      opcode = 0x03
	opJne                 opcode = 0x04
	opShortCircuitAnd     opcode = 0x05
	opCall                opcode = 0x06
	opRet                 opcode = 0x07
	opSleep               opcode = 0x08
	opLoopCounter         opcode = 0x09
	opFunctionStart       opcode = 0x0a
	opWaitFor             opcode = 0x0b
	opPushNumber          opcode = 0x14
	opPushString          opcode = 0x15
	opPushVariable        opcode = 0x16
	opPushArray           opcode = 0x17
	opPushTrue            opcode = 0x18
	opPushFalse           opcode = 0x19
	opPushNull            opcode = 0x1a
	opPi                  opcode = 0x1b
	opCopy                opcode = 0x1e
	opSwap                opcode = 0x1f
	opPop                 opcode = 0x20
	opConvertToFloat      opcode = 0x21
	opConvertToString     opcode = 0x22
	opAccessMember        opcode = 0x23
	opConvertToObject     opcode = 0x24
	opEndArray            opcode = 0x25
	opNewUninitArray      opcode = 0x26
	opSetArray            opcode = 0x27
	opNew                 opcode = 0x28
	opMakeVar             opcode = 0x29
	opNewObject           opcode = 0x2a
	opConvertToVar        opcode = 0x2b
	opShortCircuitEnd     opcode = 0x2c
	opAssign              opcode = 0x32
	opEndParams           opcode = 0x33
	opInc                 opcode = 0x34
	opDec                 opcode = 0x35
	opAssignMember        opcode = 0x36
	opAdd                 opcode = 0x3c
	opSubtract            opcode = 0x3d
	opMultiply            opcode = 0x3e
	opDivide              opcode = 0x3f
	opModulo              opcode = 0x40
	opPower               opcode = 0x41
	opBoolAnd             opcode = 0x42
	opBoolOr              opcode = 0x43
	opLogicalNot          opcode = 0x44
	opUnarySubtract       opcode = 0x45
	opEqual               opcode = 0x46
	opNotEqual            opcode = 0x47
	opLessThan            opcode = 0x48
	opGreaterThan         opcode = 0x49
	opLE                  opcode = 0x4a
	opGE                  opcode = 0x4b
	opBitwiseOr           opcode = 0x4c
	opBitwiseAnd          opcode = 0x4d
	opBitwiseXor          opcode = 0x4e
	opBitwiseInvert       opcode = 0x4f
	opInRange             opcode = 0x50
	opIn                  opcode = 0x51
	opObjIndex            opcode = 0x52
	opObjType             opcode = 0x53
	opFormat              opcode = 0x54
	opInt                 opcode = 0x55
	opAbs                 opcode = 0x56
	opRandom              opcode = 0x57
	opSin                 opcode = 0x58
	opCos                 opcode = 0x59
	opArcTan              opcode = 0x5a
	opExp                 opcode = 0x5b
	opLog                 opcode = 0x5c
	opMin                 opcode = 0x5d
	opMax                 opcode = 0x5e
	opGetAngle            opcode = 0x5f
	opGetDir              opcode = 0x60
	opVecX                opcode = 0x61
	opVecY                opcode = 0x62
	opObjIndices          opcode = 0x63
	opObjLink             opcode = 0x64
	opShiftLeft           opcode = 0x65
	opShiftRight          opcode = 0x66
	opChar                opcode = 0x67
	opObjCompare          opcode = 0x68
	opObjTrim             opcode = 0x6e
	opObjLength           opcode = 0x6f
	opObjPos              opcode = 0x70
	opJoin                opcode = 0x71
	opObjCharAt           opcode = 0x72
	opObjSubstring        opcode = 0x73
	opObjStarts           opcode = 0x74
	opObjEnds             opcode = 0x75
	opObjTokenize         opcode = 0x76
	opGetTranslation      opcode = 0x77
	opObjPositions        opcode = 0x78
	opAppend              opcode = 0x79
	opObjSize             opcode = 0x82
	opArrayAccess         opcode = 0x83
	opAssignArray         opcode = 0x84
	opMultiDimArray       opcode = 0x85
	opAssignMultiDimArray opcode = 0x86
	opObjSubArray         opcode = 0x87
	opObjAddString        opcode = 0x88
	opObjDeleteString     opcode = 0x89
	opObjRemoveString     opcode = 0x8a
	opObjReplaceString    opcode = 0x8b
	opObjInsertString     opcode = 0x8c
	opObjClear            opcode = 0x8d
	opNewMultiDimArray    opcode = 0x8e
	opSetRegister         opcode = 0x2d
	opGetRegister         opcode = 0x2e
	opMarkRegisterVar     opcode = 0x2f
	opWith                opcode = 0x96
	opWithEnd             opcode = 0x97
	opForEach             opcode = 0xa3
	opThis                opcode = 0xb4
	opThisO               opcode = 0xb5
	opPlayer              opcode = 0xb6
	opPlayerO             opcode = 0xb7
	opLevel               opcode = 0xb8
	opTemp                opcode = 0xbd
	opParams              opcode = 0xbe
	opImmStringByte       opcode = 0xf0
	opImmStringShort      opcode = 0xf1
	opImmStringInt        opcode = 0xf2
	opImmByte             opcode = 0xf3
	opImmShort            opcode = 0xf4
	opImmInt              opcode = 0xf5
	opImmFloat            opcode = 0xf6
)

type operand struct {
	str    string
	number int
	float  string
	kind   string
}

type instruction struct {
	addr    int
	op      opcode
	operand *operand
}

type module struct {
	functions []functionDef
	strings   []string
	code      []instruction
}

type functionDef struct {
	name      string
	addr      int
	bodyStart int
	params    []string
}

type functionRange struct {
	functionDef
	start int
	end   int
}

type expr struct {
	text   string
	marker bool
	kind   string
}

func readInput(inputPath string) ([]byte, error) {
	var data []byte
	var err error
	if inputPath != "" {
		data, err = os.ReadFile(inputPath)
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return nil, err
	}
	if parsed, err := parseHexBytes(string(data)); err == nil {
		return parsed, nil
	}
	return data, nil
}

func decompileData(data []byte) (string, error) {
	mod, err := parseModule(data)
	if err != nil {
		return "", err
	}
	return decompileModule(mod), nil
}

func defaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	if strings.EqualFold(ext, ".gs2bc") {
		return strings.TrimSuffix(inputPath, ext) + ".gs2"
	}
	return inputPath + ".gs2"
}

func parseHexBytes(s string) ([]byte, error) {
	var digits strings.Builder
	for _, r := range s {
		if r == 0 || r == '\ufeff' || r == '\ufffd' || unicode.IsSpace(r) {
			continue
		}
		if !isHex(r) {
			return nil, fmt.Errorf("non-hex character %q", r)
		}
		digits.WriteRune(r)
	}
	hex := digits.String()
	if hex == "" || len(hex)%2 != 0 {
		return nil, errors.New("hex input must contain an even number of digits")
	}
	out := make([]byte, 0, len(hex)/2)
	for i := 0; i < len(hex); i += 2 {
		v, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		out = append(out, byte(v))
	}
	return out, nil
}

func isHex(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}

func parseModule(data []byte) (module, error) {
	data = bytecodePayload(data)
	r := byteReader{data: data}
	mod := module{}
	for section := 0; section < 4 && r.left() > 0; section++ {
		sectionType, err := r.u32()
		if err != nil {
			return mod, err
		}
		switch sectionType {
		case 1:
			length, err := r.u32()
			if err != nil {
				return mod, err
			}
			if err := r.skip(int(length)); err != nil {
				return mod, err
			}
		case 2:
			length, err := r.u32()
			if err != nil {
				return mod, err
			}
			end := r.pos + int(length)
			for r.pos < end {
				addr, err := r.u32()
				if err != nil {
					return mod, err
				}
				name, err := r.cstr()
				if err != nil {
					return mod, err
				}
				mod.functions = append(mod.functions, functionDef{name: name, addr: int(addr)})
			}
		case 3:
			length, err := r.u32()
			if err != nil {
				return mod, err
			}
			end := r.pos + int(length)
			for r.pos < end {
				s, err := r.cstr()
				if err != nil {
					return mod, err
				}
				mod.strings = append(mod.strings, s)
			}
		case 4:
			length, err := r.u32()
			if err != nil {
				return mod, err
			}
			end := r.pos + int(length)
			code, err := readInstructions(data[r.pos:end], mod.strings)
			if err != nil {
				return mod, err
			}
			mod.code = code
			r.pos = end
		default:
			return mod, fmt.Errorf("unknown section type %d", sectionType)
		}
	}
	mod.discoverFunctionPrologues()
	return mod, nil
}

func bytecodePayload(data []byte) []byte {
	if validSectionStream(data) {
		return data
	}
	for off := 1; off+8 <= len(data); off++ {
		if binary.BigEndian.Uint32(data[off:]) == 1 && validSectionStream(data[off:]) {
			return data[off:]
		}
	}
	return data
}

func validSectionStream(data []byte) bool {
	pos := 0
	seenCode := false
	for section := 0; section < 4 && pos < len(data); section++ {
		if pos+8 > len(data) {
			return false
		}
		sectionType := binary.BigEndian.Uint32(data[pos:])
		length := int(binary.BigEndian.Uint32(data[pos+4:]))
		pos += 8
		if sectionType < 1 || sectionType > 4 || length < 0 || pos+length > len(data) {
			return false
		}
		if sectionType == 4 {
			seenCode = true
		}
		pos += length
	}
	return seenCode
}

func (m *module) discoverFunctionPrologues() {
	for i := range m.functions {
		fn := &m.functions[i]
		fn.bodyStart = fn.addr
		if fn.addr < 0 || fn.addr >= len(m.code) || m.code[fn.addr].op != opPushArray {
			continue
		}

		var params []string
		pos := fn.addr + 1
		for pos < len(m.code) && m.code[pos].op == opPushVariable {
			if m.code[pos].operand != nil {
				params = append(params, m.code[pos].operand.str)
			}
			pos++
		}
		if pos >= len(m.code) || m.code[pos].op != opEndParams {
			continue
		}
		for left, right := 0, len(params)-1; left < right; left, right = left+1, right-1 {
			params[left], params[right] = params[right], params[left]
		}
		fn.params = params
		fn.bodyStart = pos + 1
		if fn.bodyStart < len(m.code) && m.code[fn.bodyStart].op == opFunctionStart {
			fn.bodyStart++
		}
	}
}

func readInstructions(data []byte, stringsTable []string) ([]instruction, error) {
	r := byteReader{data: data}
	var code []instruction
	for r.left() > 0 {
		b, err := r.u8()
		if err != nil {
			return nil, err
		}
		op := opcode(b)
		if isImmediate(op) {
			if len(code) == 0 {
				return nil, fmt.Errorf("immediate %x without instruction", b)
			}
			imm, err := readImmediate(&r, op, stringsTable)
			if err != nil {
				return nil, err
			}
			code[len(code)-1].operand = &imm
			continue
		}
		code = append(code, instruction{addr: len(code), op: op})
	}
	return code, nil
}

func isImmediate(op opcode) bool {
	return op == opImmStringByte || op == opImmStringShort || op == opImmStringInt ||
		op == opImmByte || op == opImmShort || op == opImmInt || op == opImmFloat
}

func readImmediate(r *byteReader, op opcode, stringsTable []string) (operand, error) {
	switch op {
	case opImmStringByte:
		idx, err := r.u8()
		return stringOperand(int(idx), stringsTable, err)
	case opImmStringShort:
		idx, err := r.u16()
		return stringOperand(int(idx), stringsTable, err)
	case opImmStringInt:
		idx, err := r.u32()
		return stringOperand(int(idx), stringsTable, err)
	case opImmByte:
		v, err := r.u8()
		return operand{number: int(int8(v)), kind: "number"}, err
	case opImmShort:
		v, err := r.u16()
		return operand{number: int(int16(v)), kind: "number"}, err
	case opImmInt:
		v, err := r.u32()
		return operand{number: int(int32(v)), kind: "number"}, err
	case opImmFloat:
		s, err := r.cstr()
		return operand{float: s, kind: "float"}, err
	default:
		return operand{}, fmt.Errorf("unknown immediate opcode %x", byte(op))
	}
}

func stringOperand(idx int, stringsTable []string, err error) (operand, error) {
	if err != nil {
		return operand{}, err
	}
	if idx < 0 || idx >= len(stringsTable) {
		return operand{}, fmt.Errorf("string index %d out of range", idx)
	}
	return operand{str: stringsTable[idx], kind: "string"}, nil
}

func decompileModule(mod module) string {
	if len(mod.functions) > 0 {
		funcs := append([]functionDef(nil), mod.functions...)
		sort.Slice(funcs, func(i, j int) bool {
			return funcs[i].addr < funcs[j].addr
		})
		ranges := buildFunctionRanges(funcs, mod.code)
		var chunks []string
		for _, fn := range ranges {
			if fn.addr < 0 || fn.addr >= len(mod.code) || fn.start >= len(mod.code) || fn.start >= fn.end {
				continue
			}
			state := newDecompileState()
			if !isSyntheticFunction(fn.name) {
				state.skip = nestedFunctionRanges(fn, ranges)
			}
			body := decompileRangeWithState(mod.code, fn.start, fn.end, 1, state)
			body = removeDuplicateGotos(body)
			body = recoverProfileCloneBlocks(body)
			body = recoverBareConstructorBlocks(body)
			body = removeRepeatedAssignmentRuns(body)
			body = recoverForwardGotoGuardsFixedPoint(body)
			body = recoverForwardIfGotoLoops(body)
			body = recoverInvertedIfGotoLoops(body)
			body = recoverLoopGotoContinues(body)
			body = recoverSleepLoopBlocks(body)
			chunks = append(chunks, functionSignature(fn.name, fn.params)+" {\n"+strings.Join(body, "\n")+"\n}")
		}
		return strings.Join(chunks, "\n\n") + "\n"
	}
	lines := decompileRange(mod.code, 0, len(mod.code), 0)
	lines = removeDuplicateGotos(lines)
	lines = recoverProfileCloneBlocks(lines)
	lines = recoverBareConstructorBlocks(lines)
	lines = removeRepeatedAssignmentRuns(lines)
	lines = recoverForwardGotoGuardsFixedPoint(lines)
	lines = recoverForwardIfGotoLoops(lines)
	lines = recoverInvertedIfGotoLoops(lines)
	lines = recoverLoopGotoContinues(lines)
	lines = recoverSleepLoopBlocks(lines)
	return strings.Join(lines, "\n") + "\n"
}

func buildFunctionRanges(funcs []functionDef, code []instruction) []functionRange {
	out := make([]functionRange, len(funcs))
	nextConcrete := len(code)
	for i := len(funcs) - 1; i >= 0; i-- {
		start := funcs[i].bodyStart
		if start == 0 {
			start = funcs[i].addr
		}
		end := nextConcrete
		if isSyntheticFunction(funcs[i].name) {
			end = firstReturnEnd(code, start, nextConcrete)
		} else {
			nextConcrete = funcs[i].addr
		}
		out[i] = functionRange{functionDef: funcs[i], start: start, end: end}
	}
	return out
}

func firstReturnEnd(code []instruction, start, fallback int) int {
	for pc := start; pc < fallback && pc < len(code); pc++ {
		if code[pc].op == opRet {
			return pc + 1
		}
	}
	return fallback
}

func nestedFunctionRanges(parent functionRange, funcs []functionRange) []functionRange {
	var out []functionRange
	for _, fn := range funcs {
		if fn.addr == parent.addr || !isSyntheticFunction(fn.name) {
			continue
		}
		if fn.addr > parent.start && fn.end <= parent.end {
			out = append(out, fn)
		}
	}
	return out
}

func isSyntheticFunction(name string) bool {
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	if !strings.HasPrefix(name, "function_") {
		return false
	}
	parts := strings.Split(name, "_")
	if len(parts) != 3 {
		return false
	}
	_, errA := strconv.Atoi(parts[1])
	_, errB := strconv.Atoi(parts[2])
	return errA == nil && errB == nil
}

func functionSignature(name string, params []string) string {
	for _, visibility := range []string{"public", "private", "protected"} {
		prefix := visibility + "."
		if strings.HasPrefix(name, prefix) {
			return visibility + " function " + strings.TrimPrefix(name, prefix) + "(" + strings.Join(params, ", ") + ")"
		}
	}
	return "function " + name + "(" + strings.Join(params, ", ") + ")"
}

type decompileState struct {
	registers map[int]expr
	skip      []functionRange
}

func newDecompileState() *decompileState {
	return &decompileState{registers: map[int]expr{}}
}

func decompileRange(code []instruction, start, end, indent int) []string {
	return decompileRangeWithState(code, start, end, indent, newDecompileState())
}

func decompileRangeWithState(code []instruction, start, end, indent int, state *decompileState) []string {
	return decompileRangeWithStateAndStack(code, start, end, indent, state, nil)
}

func decompileRangeWithStateAndStack(code []instruction, start, end, indent int, state *decompileState, initialStack []expr) []string {
	var lines []string
	stack := append([]expr(nil), initialStack...)
	for pc := start; pc < end; pc++ {
		if skipEnd, ok := skipRangeEnd(state.skip, pc); ok {
			pc = skipEnd - 1
			continue
		}
		if dispatchLines, newPC, ok := recoverTailDispatch(code, pc, end, indent, state); ok {
			lines = append(lines, dispatchLines...)
			pc = newPC
			continue
		}
		ins := code[pc]
		switch ins.op {
		case opNone:
		case opPushArray:
			stack = append(stack, expr{marker: true})
		case opPushString:
			stack = append(stack, expr{text: quote(ins.operand.str), kind: "string"})
		case opPushVariable:
			stack = append(stack, expr{text: variableName(ins.operand.str)})
		case opPushNumber:
			stack = append(stack, expr{text: numberText(ins.operand)})
		case opPushTrue:
			stack = append(stack, expr{text: "true"})
		case opPushFalse:
			stack = append(stack, expr{text: "false"})
		case opPushNull:
			stack = append(stack, expr{text: "null"})
		case opPi:
			stack = append(stack, expr{text: "pi"})
		case opThis:
			stack = append(stack, expr{text: "this"})
		case opThisO:
			stack = append(stack, expr{text: "thiso"})
		case opPlayer:
			stack = append(stack, expr{text: "player"})
		case opPlayerO:
			stack = append(stack, expr{text: "playero"})
		case opLevel:
			stack = append(stack, expr{text: "level"})
		case opTemp:
			stack = append(stack, expr{text: "temp"})
		case opParams:
			stack = append(stack, expr{text: "params"})
		case opConvertToFloat, opConvertToString, opConvertToObject, opConvertToVar, opEndParams, opFunctionStart, opLoopCounter, opShortCircuitEnd:
		case opNew, opWithEnd:
		case opNewObject:
			className := popExpr(&stack)
			target := popExpr(&stack)
			if isUnknownObjectPlaceholder(target.text) {
				obj := expr{text: "new " + unquoteText(className.text) + "()", kind: "object"}
				if len(stack) == 0 {
					stack = append(stack, expr{text: "temp.object"}, obj)
				} else {
					stack = append(stack, obj)
				}
			} else if len(stack) > 0 {
				stack = append(stack, expr{text: "new " + unquoteText(className.text) + "(" + constructorArg(target) + ")", kind: "object"})
			} else {
				className.kind = "class"
				stack = append(stack, target, className)
			}
		case opWith:
			target := jumpTarget(ins)
			targetExpr := popExpr(&stack)
			if target > pc && target <= end && len(lines) > 0 && isConstructorLine(lines[len(lines)-1]) && constructorLineMatchesTarget(lines[len(lines)-1], targetExpr.text) {
				assignmentConstructor := isAssignmentConstructorLine(lines[len(lines)-1])
				lines[len(lines)-1] = strings.TrimSuffix(lines[len(lines)-1], ";") + " {"
				lines = append(lines, decompileRangeWithState(code, pc+1, target, indent+1, state)...)
				close := "}"
				if assignmentConstructor {
					close = "};"
				}
				lines = append(lines, pad(indent)+close)
				pc = target - 1
			} else if target > pc && target <= end {
				lines = append(lines, pad(indent)+"with ("+targetExpr.text+") {")
				lines = append(lines, decompileRangeWithState(code, pc+1, target, indent+1, state)...)
				lines = append(lines, pad(indent)+"}")
				pc = target - 1
			}
		case opShortCircuitOr, opShortCircuitAnd:
			if len(stack) < 2 {
				continue
			}
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			if lhs.marker {
				stack = append(stack, lhs, rhs)
				continue
			}
			stack = append(stack, expr{text: lhs.text + " " + infix(ins.op) + " " + rhs.text})
		case opEndArray:
			args := collectArgs(&stack)
			stack = append(stack, expr{text: "{" + strings.Join(args, ", ") + "}"})
		case opNewUninitArray:
			size := popExpr(&stack)
			stack = append(stack, expr{text: "new [" + size.text + "]"})
		case opCopy:
			item := popExpr(&stack)
			stack = append(stack, item, item)
		case opSwap:
			a, b := popExpr(&stack), popExpr(&stack)
			stack = append(stack, a, b)
		case opSetRegister:
			item := popExpr(&stack)
			state.registers[operandNumber(ins)] = item
			stack = append(stack, item)
		case opGetRegister:
			id := operandNumber(ins)
			if item, ok := state.registers[id]; ok {
				stack = append(stack, item)
			} else {
				stack = append(stack, expr{text: fmt.Sprintf("reg%d", id)})
			}
		case opMarkRegisterVar:
		case opInc:
			item := popExpr(&stack)
			lines = append(lines, pad(indent)+item.text+" += 1;")
			stack = append(stack, item)
		case opDec:
			item := popExpr(&stack)
			lines = append(lines, pad(indent)+item.text+" -= 1;")
			stack = append(stack, item)
		case opAccessMember:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: memberBase(lhs.text) + "." + memberName(rhs.text)})
		case opAssignMember:
			rhs, prop, obj := popExpr(&stack), popExpr(&stack), popExpr(&stack)
			lines = append(lines, pad(indent)+memberBase(obj.text)+"."+memberName(prop.text)+" = "+rhs.text+";")
		case opAdd, opSubtract, opMultiply, opDivide, opModulo, opPower, opBoolAnd, opBoolOr, opEqual, opNotEqual, opLessThan, opGreaterThan, opLE, opGE, opBitwiseOr, opBitwiseAnd, opBitwiseXor, opShiftLeft, opShiftRight, opIn, opJoin, opAppend:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: lhs.text + " " + infix(ins.op) + " " + rhs.text})
		case opInRange:
			upper, lower, item := popExpr(&stack), popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: item.text + " in <" + lower.text + ", " + upper.text + ">"})
		case opLogicalNot:
			item := popExpr(&stack)
			stack = append(stack, expr{text: "!" + memberBase(item.text)})
		case opUnarySubtract:
			item := popExpr(&stack)
			stack = append(stack, expr{text: "-" + memberBase(item.text)})
		case opBitwiseInvert:
			item := popExpr(&stack)
			stack = append(stack, expr{text: "~" + memberBase(item.text)})
		case opArrayAccess:
			index, arr := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: arr.text + "[" + index.text + "]"})
		case opAssignArray, opSetArray:
			rhs, index, arr := popExpr(&stack), popExpr(&stack), popExpr(&stack)
			lines = append(lines, pad(indent)+arr.text+"["+index.text+"] = "+rhs.text+";")
		case opMultiDimArray:
			stack = append(stack, multiDimArrayExpr(&stack))
		case opAssignMultiDimArray:
			rhs := popExpr(&stack)
			target := multiDimTarget(&stack)
			lines = append(lines, pad(indent)+target+" = "+rhs.text+";")
		case opObjStarts:
			stack = append(stack, objectCall(&stack, "starts", 1, false))
		case opGetTranslation:
			arg := popExpr(&stack)
			stack = append(stack, expr{text: "_(" + arg.text + ")"})
		case opObjSubstring:
			stack = append(stack, objectCall(&stack, "substring", 2, false))
		case opObjSize:
			stack = append(stack, objectCall(&stack, "size", 0, false))
		case opObjIndex:
			stack = append(stack, objectCall(&stack, "index", 1, false))
		case opInt:
			stack = append(stack, functionCall(&stack, "int", 1))
		case opChar:
			stack = append(stack, functionCall(&stack, "char", 1))
		case opSleep:
			lines = append(lines, pad(indent)+functionCall(&stack, "sleep", 1).text+";")
		case opWaitFor:
			lines = append(lines, pad(indent)+functionCall(&stack, "waitfor", 1).text+";")
		case opMakeVar:
			stack = append(stack, functionCall(&stack, "makevar", 1))
		case opAbs:
			stack = append(stack, functionCall(&stack, "abs", 1))
		case opRandom:
			stack = append(stack, functionCall(&stack, "random", 2))
		case opSin:
			stack = append(stack, functionCall(&stack, "sin", 1))
		case opCos:
			stack = append(stack, functionCall(&stack, "cos", 1))
		case opArcTan:
			stack = append(stack, functionCall(&stack, "arctan", 1))
		case opExp:
			stack = append(stack, functionCall(&stack, "exp", 1))
		case opLog:
			stack = append(stack, functionCall(&stack, "log", 1))
		case opMin:
			stack = append(stack, functionCall(&stack, "min", 2))
		case opMax:
			stack = append(stack, functionCall(&stack, "max", 2))
		case opGetAngle:
			stack = append(stack, functionCall(&stack, "getangle", 2))
		case opGetDir:
			stack = append(stack, functionCall(&stack, "getdir", 2))
		case opVecX:
			stack = append(stack, functionCall(&stack, "vecx", 1))
		case opVecY:
			stack = append(stack, functionCall(&stack, "vecy", 1))
		case opObjCompare:
			stack = append(stack, functionCall(&stack, "objcompare", 2))
		case opFormat:
			args := collectArgs(&stack)
			stack = append(stack, expr{text: "format(" + strings.Join(args, ", ") + ")"})
		case opObjType:
			stack = append(stack, objectCall(&stack, "type", 0, false))
		case opObjIndices:
			stack = append(stack, objectCall(&stack, "indices", 0, false))
		case opObjLink:
			stack = append(stack, objectCall(&stack, "link", 0, false))
		case opObjTrim:
			stack = append(stack, objectCall(&stack, "trim", 0, false))
		case opObjLength:
			stack = append(stack, objectCall(&stack, "length", 0, false))
		case opObjPos:
			stack = append(stack, objectCall(&stack, "pos", 1, false))
		case opObjCharAt:
			stack = append(stack, objectCall(&stack, "charat", 1, false))
		case opObjEnds:
			stack = append(stack, objectCall(&stack, "ends", 1, false))
		case opObjTokenize:
			stack = append(stack, objectCall(&stack, "tokenize", 1, false))
		case opObjPositions:
			stack = append(stack, objectCall(&stack, "positions", 1, false))
		case opObjSubArray:
			stack = append(stack, objectCall(&stack, "subarray", 2, false))
		case opObjAddString:
			lines = append(lines, pad(indent)+objectCall(&stack, "add", 1, true).text+";")
		case opObjDeleteString:
			lines = append(lines, pad(indent)+objectCall(&stack, "delete", 1, true).text+";")
		case opObjRemoveString:
			lines = append(lines, pad(indent)+objectCall(&stack, "remove", 1, true).text+";")
		case opObjReplaceString:
			lines = append(lines, pad(indent)+objectCall(&stack, "replace", 2, true).text+";")
		case opObjInsertString:
			lines = append(lines, pad(indent)+objectCall(&stack, "insert", 2, true).text+";")
		case opObjClear:
			lines = append(lines, pad(indent)+objectCall(&stack, "clear", 0, true).text+";")
		case opNewMultiDimArray:
			stack = append(stack, newMultiDimArrayExpr(&stack))
		case opAssign:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			if recoveredLHS, recoveredRHS, ok := recoverFormatAssignment(lhs, rhs); ok {
				lhs, rhs = recoveredLHS, recoveredRHS
			}
			if recoveredLHS, recoveredRHS, ok := recoverNewMultiDimAssignment(lhs, rhs); ok {
				lhs, rhs = recoveredLHS, recoveredRHS
			}
			if recoveredLHS, recoveredRHS, ok := recoverEmbeddedBooleanAssignment(lhs, rhs); ok {
				lhs, rhs = recoveredLHS, recoveredRHS
			}
			if recoveredLHS, recoveredRHS, ok := recoverSwappedBooleanAssignment(lhs, rhs); ok {
				lhs, rhs = recoveredLHS, recoveredRHS
			}
			if isHiddenFunctionBinding(lhs, rhs) {
				continue
			}
			rhs = normalizeAssignmentValue(lhs, rhs)
			if isNamedProfileCloneConstruction(lhs, rhs) {
				arg, _ := constructorExprArg(rhs.text)
				lines = append(lines, pad(indent)+"new GuiControlProfile("+arg+");")
			} else if isNamedGuiConstruction(lhs, rhs) {
				lines = append(lines, pad(indent)+rhs.text+";")
			} else if isConstructorTarget(lhs, rhs) {
				lines = append(lines, pad(indent)+"new "+unquoteText(rhs.text)+"("+constructorArg(lhs)+");")
			} else if rhs.kind == "class" {
				lines = append(lines, pad(indent)+classAssignmentTarget(lhs)+" = new "+unquoteText(rhs.text)+"();")
			} else {
				lines = append(lines, pad(indent)+lhs.text+" = "+rhs.text+";")
			}
		case opCall:
			call := buildCall(&stack)
			stack = append(stack, expr{text: call, kind: "call"})
		case opPop:
			item := popExpr(&stack)
			if item.kind == "call" && item.text != "" {
				lines = append(lines, pad(indent)+item.text+";")
			}
		case opJne, opJeq:
			target := jumpTarget(ins)
			condition := popExpr(&stack).text
			if assignLines, newPC, ok := recoverConditionalAssignmentChain(code, pc, target, end, indent, state, condition, ins.op, stack); ok {
				lines = append(lines, assignLines...)
				stack = stack[:len(stack)-1]
				pc = newPC
				continue
			}
			if assignLines, newPC, ok := recoverTernaryAssignment(code, pc, target, end, indent, state, condition, ins.op, stack); ok {
				lines = append(lines, assignLines...)
				stack = stack[:len(stack)-1]
				pc = newPC
				continue
			}
			if assignLines, newPC, ok := recoverSelfTernaryAssignment(code, pc, target, end, indent, state, condition, ins.op, stack); ok {
				lines = append(lines, assignLines...)
				stack = stack[:len(stack)-1]
				pc = newPC
				continue
			}
			if value, newPC, ok := recoverTernaryExpression(code, pc, target, end, state, condition, ins.op); ok {
				stack = append(stack, expr{text: value})
				pc = newPC
				continue
			}
			if target > pc && target <= end {
				if ins.op == opJeq {
					condition = "!(" + condition + ")"
				}
				body := decompileRangeWithStateAndStack(code, pc+1, target, indent+1, state, stack)
				body = trimAfterReturn(body)
				if forLoop, ok := recoverForLoop(lines, body, condition, pc, indent); ok {
					lines = forLoop
				} else if whileLoop, ok := recoverWhileLoop(body, condition, pc, indent); ok {
					lines = append(lines, whileLoop...)
				} else {
					lines = append(lines, pad(indent)+"if ("+condition+") {")
					lines = append(lines, body...)
					lines = append(lines, pad(indent)+"}")
				}
				pc = target - 1
			} else {
				loopCondition := condition
				if ins.op == opJeq {
					loopCondition = "!(" + condition + ")"
				}
				loopEnd := loopRecoveryEnd(code, pc+1, end, pc)
				recovered := false
				if loopEnd > pc+1 {
					body := decompileRangeWithStateAndStack(code, pc+1, loopEnd, indent+1, state, stack)
					if forLoop, ok := recoverForLoop(lines, body, loopCondition, pc, indent); ok {
						lines = forLoop
						pc = loopEnd - 1
						recovered = true
					}
				}
				if !recovered {
					lines = append(lines, pad(indent)+fmt.Sprintf("if (%s) goto label_%d;", condition, target))
				}
			}
		case opJmp:
			target := jumpTarget(ins)
			if dispatchLines, newPC, ok := recoverForwardDispatch(code, pc, target, end, indent, state); ok {
				lines = append(lines, dispatchLines...)
				pc = newPC
			} else if dispatchLines, newPC, ok := recoverBackwardDispatch(code, pc, target, end, indent, state); ok {
				lines = append(lines, dispatchLines...)
				pc = newPC
			} else if skipsEmbeddedFunction(state.skip, pc+1, target) {
			} else if target == end {
				pc = end - 1
			} else if target > pc && target <= end && isJumpPadding(code, pc+1, target) {
				pc = target - 1
			} else if target < end {
				lines = append(lines, pad(indent)+fmt.Sprintf("goto label_%d;", target))
			}
		case opForEach:
			target := jumpTarget(ins)
			_, collection, iter := popExpr(&stack), popExpr(&stack), popExpr(&stack)
			condition := iter.text + " in " + collection.text
			if target > pc && target <= end {
				body := decompileRangeWithState(code, pc+1, target, indent+1, state)
				body = trimForEachBookkeeping(body)
				lines = append(lines, pad(indent)+"for ("+condition+") {")
				lines = append(lines, body...)
				lines = append(lines, pad(indent)+"}")
				pc = target - 1
			} else {
				stack = append(stack, expr{text: condition})
			}
		case opRet:
			if len(stack) > 0 {
				ret := popExpr(&stack).text
				if !(indent == 1 && isTerminalRet(code, pc, end) && ret == "0") {
					lines = append(lines, pad(indent)+"return "+ret+";")
				}
			} else {
				if !(indent == 1 && isTerminalRet(code, pc, end)) {
					lines = append(lines, pad(indent)+"return;")
				}
			}
		default:
			lines = append(lines, pad(indent)+fmt.Sprintf("// unhandled opcode 0x%02x at %d", byte(ins.op), ins.addr))
		}
	}
	return collapseNestedIfs(lines)
}

func skipRangeEnd(ranges []functionRange, pc int) (int, bool) {
	for _, r := range ranges {
		if pc == r.addr {
			return r.end, true
		}
	}
	return 0, false
}

func skipsEmbeddedFunction(ranges []functionRange, start, target int) bool {
	for _, r := range ranges {
		if r.addr == start && r.end == target {
			return true
		}
	}
	return false
}

func isHiddenFunctionBinding(lhs, rhs expr) bool {
	lhsText := strings.TrimSpace(lhs.text)
	if lhsText != "" && lhsText != "/* missing */" {
		return false
	}
	return isSyntheticFunction(rhs.text)
}

func recoverForLoop(lines []string, body []string, condition string, pc int, indent int) ([]string, bool) {
	if len(lines) == 0 || len(body) < 2 {
		return nil, false
	}

	gotoLine := strings.TrimSpace(body[len(body)-1])
	if !strings.HasPrefix(gotoLine, "goto label_") || !strings.HasSuffix(gotoLine, ";") {
		return nil, false
	}
	labelText := strings.TrimSuffix(strings.TrimPrefix(gotoLine, "goto label_"), ";")
	label, err := strconv.Atoi(labelText)
	if err != nil || label > pc || label < pc-16 {
		return nil, false
	}

	workBody := append([]string(nil), body[:len(body)-1]...)
	if len(workBody) == 0 {
		return nil, false
	}

	incLine := strings.TrimSpace(workBody[len(workBody)-1])
	if len(workBody) >= 2 {
		maybeBare := strings.TrimSpace(workBody[len(workBody)-1])
		prev := strings.TrimSpace(workBody[len(workBody)-2])
		if strings.HasSuffix(prev, " += 1;") && maybeBare == strings.TrimSuffix(strings.TrimSuffix(prev, " += 1;"), " ")+";" {
			workBody = workBody[:len(workBody)-1]
			incLine = strings.TrimSpace(workBody[len(workBody)-1])
		}
	}
	incVar, inc, ok := parseLoopIncrement(incLine)
	if !ok {
		return nil, false
	}

	initLine := strings.TrimSpace(lines[len(lines)-1])
	initPrefix := incVar + " = "
	if !strings.HasPrefix(initLine, initPrefix) || !strings.HasSuffix(initLine, ";") {
		return nil, false
	}
	if !strings.Contains(condition, incVar) {
		return nil, false
	}

	init := strings.TrimSuffix(initLine, ";")
	result := append([]string(nil), lines[:len(lines)-1]...)
	result = append(result, pad(indent)+"for ("+init+"; "+condition+"; "+inc+") {")
	result = append(result, replaceGotoTarget(workBody[:len(workBody)-1], label, "continue;")...)
	result = append(result, pad(indent)+"}")
	return result, true
}

func replaceGotoTarget(lines []string, target int, replacement string) []string {
	out := append([]string(nil), lines...)
	want := fmt.Sprintf("goto label_%d;", target)
	for i, line := range out {
		if strings.TrimSpace(line) == want {
			out[i] = strings.Repeat(" ", parseLineIndent(line)) + replacement
		}
	}
	return out
}

func parseLoopIncrement(line string) (string, string, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")
	if strings.HasSuffix(line, " += 1") {
		return strings.TrimSuffix(line, " += 1"), line, true
	}
	if idx := strings.Index(line, " = "); idx >= 0 {
		lhs := line[:idx]
		rhs := line[idx+3:]
		prefix := lhs + " + "
		if strings.HasPrefix(rhs, prefix) {
			step := strings.TrimSpace(strings.TrimPrefix(rhs, prefix))
			if step != "" {
				return lhs, lhs + " += " + step, true
			}
		}
	}
	return "", "", false
}

func recoverForwardIfGotoLoops(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if i+1 >= len(lines) {
			out = append(out, lines[i])
			continue
		}
		initLine := strings.TrimSpace(lines[i])
		if !strings.HasSuffix(initLine, ";") || !strings.Contains(initLine, " = ") || parseLineIndent(lines[i]) != parseLineIndent(lines[i+1]) {
			out = append(out, lines[i])
			continue
		}
		condition, ok := parseBlockIfLine(lines[i+1])
		if !ok {
			out = append(out, lines[i])
			continue
		}
		blockEnd := matchingBlockEnd(lines, i+1)
		if blockEnd < 0 {
			out = append(out, lines[i])
			continue
		}
		body := lines[i+2 : blockEnd]
		if len(body) < 2 || !isGotoLine(strings.TrimSpace(body[len(body)-1])) {
			out = append(out, lines[i])
			continue
		}
		incVar, inc, ok := parseLoopIncrement(body[len(body)-2])
		if !ok || !strings.HasPrefix(initLine, incVar+" = ") || !strings.Contains(condition, incVar) {
			out = append(out, lines[i])
			continue
		}
		labelText := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(body[len(body)-1]), "goto label_"), ";")
		label, err := strconv.Atoi(labelText)
		if err != nil {
			out = append(out, lines[i])
			continue
		}
		indent := parseLineIndent(lines[i])
		out = append(out, strings.Repeat(" ", indent)+"for ("+strings.TrimSuffix(initLine, ";")+"; "+condition+"; "+inc+") {")
		out = append(out, replaceGotoTarget(body[:len(body)-2], label, "continue;")...)
		out = append(out, strings.Repeat(" ", indent)+"}")
		i = blockEnd
	}
	return out
}

func parseBlockIfLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "if (") || !strings.HasSuffix(trimmed, ") {") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(trimmed, "if ("), ") {"), true
}

func recoverInvertedIfGotoLoops(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		initLine := strings.TrimSpace(lines[i])
		if !strings.HasSuffix(initLine, ";") || !strings.Contains(initLine, " = ") {
			out = append(out, lines[i])
			continue
		}
		indent := parseLineIndent(lines[i])
		parts := strings.SplitN(strings.TrimSuffix(initLine, ";"), " = ", 2)
		loopVar := parts[0]
		if loopVar == "" {
			out = append(out, lines[i])
			continue
		}
		condIdx := -1
		condition := ""
		for j := i + 1; j < len(lines) && j <= i+8; j++ {
			if parseLineIndent(lines[j]) != indent {
				continue
			}
			cond, ok := parseBlockIfLine(lines[j])
			if ok && strings.HasPrefix(cond, "!(") && strings.HasSuffix(cond, ")") && strings.Contains(cond, loopVar) {
				condIdx = j
				condition = strings.TrimSuffix(strings.TrimPrefix(cond, "!("), ")")
				break
			}
		}
		if condIdx < 0 {
			out = append(out, lines[i])
			continue
		}
		blockEnd := matchingBlockEnd(lines, condIdx)
		if blockEnd < 0 {
			out = append(out, lines[i])
			continue
		}
		incIdx := -1
		incText := ""
		for j := blockEnd + 1; j+1 < len(lines) && j <= blockEnd+12; j++ {
			incVar, inc, ok := parseLoopIncrement(lines[j])
			if ok && incVar == loopVar && isGotoLine(strings.TrimSpace(lines[j+1])) {
				incIdx = j
				incText = inc
				break
			}
		}
		if incIdx < 0 {
			out = append(out, lines[i])
			continue
		}
		out = append(out, strings.Repeat(" ", indent)+"for ("+initLine[:len(initLine)-1]+"; "+condition+"; "+incText+") {")
		for _, line := range lines[i+1 : condIdx] {
			out = append(out, reindentBlockLine(line, indent, indent+2))
		}
		for _, line := range lines[condIdx+1 : blockEnd] {
			out = append(out, line)
		}
		for _, line := range lines[blockEnd+1 : incIdx] {
			out = append(out, line)
		}
		out = append(out, strings.Repeat(" ", indent)+"}")
		i = incIdx + 1
	}
	return out
}

func recoverLoopGotoContinues(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !(strings.HasPrefix(trimmed, "for ") || strings.HasPrefix(trimmed, "while ")) || !strings.HasSuffix(trimmed, "{") {
			out = append(out, lines[i])
			continue
		}
		end := matchingBlockEnd(lines, i)
		if end < 0 {
			out = append(out, lines[i])
			continue
		}
		out = append(out, lines[i])
		body := recoverLoopGotoContinues(lines[i+1 : end])
		for j := 0; j < len(body); j++ {
			if converted, next, ok := convertEmptyIfToContinue(body, j); ok {
				out = append(out, converted...)
				j = next
				continue
			}
			converted := convertGotoToContinue(body[j])
			out = append(out, converted...)
		}
		out = append(out, lines[end])
		i = end
	}
	return out
}

func convertGotoToContinue(line string) []string {
	indent := parseLineIndent(line)
	trimmed := strings.TrimSpace(line)
	if isGotoLine(trimmed) {
		return []string{strings.Repeat(" ", indent) + "continue;"}
	}
	cond, _, _, ok := parseGotoIfLine(line)
	if !ok {
		return []string{line}
	}
	prefix := strings.Repeat(" ", indent)
	return []string{prefix + "if (" + cond + ") {", prefix + "  continue;", prefix + "}"}
}

func convertEmptyIfToContinue(lines []string, index int) ([]string, int, bool) {
	if index+1 >= len(lines) {
		return nil, 0, false
	}
	cond, ok := parseBlockIfLine(lines[index])
	if !ok || strings.TrimSpace(lines[index+1]) != "}" || parseLineIndent(lines[index+1]) != parseLineIndent(lines[index]) {
		return nil, 0, false
	}
	indent := parseLineIndent(lines[index])
	prefix := strings.Repeat(" ", indent)
	return []string{prefix + "if (" + cond + ") {", prefix + "  continue;", prefix + "}"}, index + 1, true
}

func recoverWhileLoop(body []string, condition string, pc int, indent int) ([]string, bool) {
	if len(body) == 0 {
		return nil, false
	}
	gotoLine := strings.TrimSpace(body[len(body)-1])
	if !isGotoLine(gotoLine) {
		return nil, false
	}
	labelText := strings.TrimSuffix(strings.TrimPrefix(gotoLine, "goto label_"), ";")
	label, err := strconv.Atoi(labelText)
	if err != nil || label > pc || label < pc-16 {
		return nil, false
	}
	workBody := fillEmptyLoopExitIfs(body[:len(body)-1])
	result := []string{pad(indent) + "while (" + condition + ") {"}
	result = append(result, workBody...)
	result = append(result, pad(indent)+"}")
	return result, true
}

func fillEmptyLoopExitIfs(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		if strings.HasSuffix(strings.TrimSpace(lines[i]), "{") && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "}" {
			indent := parseLineIndent(lines[i])
			out = append(out, strings.Repeat(" ", indent)+pad(1)+"break;")
		}
	}
	return out
}

func loopRecoveryEnd(code []instruction, start, end, branchPC int) int {
	for i := start; i < end; i++ {
		if code[i].op == opJmp && jumpTarget(code[i]) <= branchPC {
			return i + 1
		}
		if code[i].op == opRet {
			return 0
		}
	}
	return 0
}

type dispatchCase struct {
	condition string
	target    int
}

func recoverForwardDispatch(code []instruction, pc, target, end, indent int, state *decompileState) ([]string, int, bool) {
	if target <= pc+1 || target >= end {
		return nil, 0, false
	}
	cases, tail, selector, ok := parseForwardDispatchCases(code, pc, target, end, state)
	if !ok || len(cases) == 0 {
		return nil, 0, false
	}
	targets := make([]int, 0, len(cases))
	seen := map[int]bool{}
	for _, c := range cases {
		if c.target <= pc || c.target >= end || seen[c.target] {
			continue
		}
		seen[c.target] = true
		targets = append(targets, c.target)
	}
	if len(targets) == 0 {
		return nil, 0, false
	}
	sort.Ints(targets)
	targetToNext := map[int]int{}
	for i, t := range targets {
		next := target
		if i+1 < len(targets) {
			next = targets[i+1]
		} else if t >= tail {
			next = end
		}
		if t < target && next > target {
			next = target
		}
		if t >= tail {
			next = caseBodyEnd(code, t, next)
		}
		targetToNext[t] = next
	}
	commonEnd, hasCommonEnd := forwardDispatchCommonEnd(code, targets, target)
	var lines []string
	if selectorNeedsBinding(selector) {
		lines = append(lines, pad(indent)+"temp.switchvalue = "+selector+";")
	}
	maxEnd := tail
	for i, c := range cases {
		bodyEnd := targetToNext[c.target]
		if bodyEnd <= c.target {
			return nil, 0, false
		}
		if bodyEnd > maxEnd {
			maxEnd = bodyEnd
		}
		body := removeDuplicateGotos(decompileRangeWithState(code, c.target, bodyEnd, indent+1, state))
		if hasCommonEnd {
			body = trimTrailingGoto(body, commonEnd)
		}
		if c.condition == "" {
			lines = append(lines, pad(indent)+"else {")
		} else if i == 0 {
			lines = append(lines, pad(indent)+"if ("+c.condition+") {")
		} else {
			lines = append(lines, pad(indent)+"else if ("+c.condition+") {")
		}
		lines = append(lines, body...)
		lines = append(lines, pad(indent)+"}")
	}
	return lines, maxEnd - 1, true
}

func parseForwardDispatchCases(code []instruction, pc, target, end int, state *decompileState) ([]dispatchCase, int, string, bool) {
	selector, pos, ok := dispatchSelector(code, target, state)
	if !ok {
		return nil, 0, "", false
	}
	conditionSelector := selector
	if selectorNeedsBinding(selector) {
		conditionSelector = "temp.switchvalue"
	}
	var cases []dispatchCase
	for pos+4 < end {
		if code[pos].op != opCopy || code[pos+2].op != opEqual {
			break
		}
		lit, ok := dispatchLiteral(code[pos+1])
		if !ok {
			break
		}
		jump := code[pos+3]
		if jump.op != opJeq && jump.op != opJne {
			break
		}
		caseTarget := jumpTarget(jump)
		if caseTarget <= pc || caseTarget >= end {
			break
		}
		condition := conditionSelector + " == " + lit
		if jump.op == opJne {
			condition = conditionSelector + " != " + lit
		}
		cases = append(cases, dispatchCase{condition: condition, target: caseTarget})
		pos += 4
	}
	if pos < end && code[pos].op == opJmp {
		defaultTarget := jumpTarget(code[pos])
		if defaultTarget > pc && defaultTarget < target {
			cases = append(cases, dispatchCase{target: defaultTarget})
			pos++
		}
	}
	if pos < end && code[pos].op == opPop {
		pos++
	}
	return cases, pos, selector, len(cases) > 0
}

func selectorNeedsBinding(selector string) bool {
	return strings.Contains(selector, "(")
}

func dispatchLiteral(ins instruction) (string, bool) {
	if ins.operand == nil {
		return "", false
	}
	switch ins.op {
	case opPushString:
		return quote(ins.operand.str), true
	case opPushNumber:
		return numberText(ins.operand), true
	default:
		return "", false
	}
}

func forwardDispatchCommonEnd(code []instruction, targets []int, dispatchStart int) (int, bool) {
	commonEnd := -1
	for i, target := range targets {
		limit := dispatchStart
		if i+1 < len(targets) {
			limit = targets[i+1]
		}
		endJump := -1
		for pos := target; pos < limit; pos++ {
			if code[pos].op == opJmp && jumpTarget(code[pos]) >= dispatchStart {
				endJump = jumpTarget(code[pos])
				break
			}
		}
		if endJump < 0 {
			continue
		}
		if commonEnd < 0 {
			commonEnd = endJump
		} else if commonEnd != endJump {
			return 0, false
		}
	}
	return commonEnd, commonEnd >= 0
}

func caseBodyEnd(code []instruction, start, limit int) int {
	if limit > len(code) {
		limit = len(code)
	}
	for i := start; i < limit; i++ {
		switch code[i].op {
		case opRet, opJmp:
			return i + 1
		}
	}
	return limit
}

func recoverBackwardDispatch(code []instruction, pc, target, end, indent int, state *decompileState) ([]string, int, bool) {
	if target <= pc || target >= end {
		return nil, 0, false
	}
	cases, tail, ok := parseBackwardDispatchCases(code, pc, target, end, state)
	if !ok || len(cases) == 0 {
		return nil, 0, false
	}
	commonEnd, ok := dispatchCommonEnd(code, cases, target)
	if !ok || commonEnd <= target || commonEnd > end {
		return nil, 0, false
	}

	targets := make([]int, len(cases))
	for i, c := range cases {
		targets[i] = c.target
	}
	sort.Ints(targets)
	targetToNext := map[int]int{}
	for i, t := range targets {
		next := target
		if i+1 < len(targets) {
			next = targets[i+1]
		}
		targetToNext[t] = next
	}

	var lines []string
	for i, c := range cases {
		bodyEnd := targetToNext[c.target]
		if bodyEnd <= c.target {
			return nil, 0, false
		}
		body := removeDuplicateGotos(decompileRangeWithState(code, c.target, bodyEnd, indent+1, state))
		body = trimTrailingGoto(body, commonEnd)
		if i == 0 {
			lines = append(lines, pad(indent)+"if ("+c.condition+") {")
		} else {
			lines = append(lines, pad(indent)+"else if ("+c.condition+") {")
		}
		lines = append(lines, body...)
		lines = append(lines, pad(indent)+"}")
	}
	return lines, skipDispatchTail(code, tail, commonEnd, end), true
}

type conditionalAssignmentCase struct {
	condition string
	value     string
}

func recoverTernaryAssignment(code []instruction, pc, target, end, indent int, state *decompileState, condition string, branchOp opcode, stack []expr) ([]string, int, bool) {
	if len(stack) == 0 || target <= pc+1 || target >= end || code[target-1].op != opJmp {
		return nil, 0, false
	}
	common := jumpTarget(code[target-1])
	if common <= target || common >= end || code[common].op != opAssign {
		return nil, 0, false
	}
	trueValue, ok := evalExprRange(code, pc+1, target-1, state)
	if !ok {
		return nil, 0, false
	}
	falseValue, ok := evalExprRange(code, target, common, state)
	if !ok {
		return nil, 0, false
	}
	if branchOp == opJeq {
		trueValue, falseValue = falseValue, trueValue
	}
	lhs := stack[len(stack)-1].text
	lines := []string{
		pad(indent) + "if (" + condition + ") {",
		pad(indent+1) + lhs + " = " + trueValue + ";",
		pad(indent) + "}",
		pad(indent) + "else {",
		pad(indent+1) + lhs + " = " + falseValue + ";",
		pad(indent) + "}",
	}
	return lines, common, true
}

func recoverSelfTernaryAssignment(code []instruction, pc, target, end, indent int, state *decompileState, condition string, branchOp opcode, stack []expr) ([]string, int, bool) {
	if len(stack) == 0 || target <= pc+1 || target >= end || code[target].op != opAssign {
		return nil, 0, false
	}
	falseValue, ok := evalExprRange(code, pc+1, target, state)
	if !ok {
		return nil, 0, false
	}
	trueValue := condition
	if branchOp == opJeq {
		trueValue, falseValue = falseValue, trueValue
	}
	lhs := stack[len(stack)-1].text
	lines := []string{
		pad(indent) + "if (" + condition + ") {",
		pad(indent+1) + lhs + " = " + trueValue + ";",
		pad(indent) + "}",
		pad(indent) + "else {",
		pad(indent+1) + lhs + " = " + falseValue + ";",
		pad(indent) + "}",
	}
	return lines, target, true
}

func recoverTernaryExpression(code []instruction, pc, target, end int, state *decompileState, condition string, branchOp opcode) (string, int, bool) {
	if target <= pc+1 || target >= end || code[target-1].op != opJmp {
		return "", 0, false
	}
	common := jumpTarget(code[target-1])
	if common <= target || common > end {
		return "", 0, false
	}
	trueValue, ok := evalExprRange(code, pc+1, target-1, state)
	if !ok {
		return "", 0, false
	}
	falseValue, ok := evalExprRange(code, target, common, state)
	if !ok {
		return "", 0, false
	}
	if branchOp == opJeq {
		trueValue, falseValue = falseValue, trueValue
	}
	return "(" + condition + " ? " + trueValue + " : " + falseValue + ")", common - 1, true
}

func recoverConditionalAssignmentChain(code []instruction, pc, target, end, indent int, state *decompileState, firstCondition string, firstOp opcode, stack []expr) ([]string, int, bool) {
	if firstOp != opJne || len(stack) == 0 || target != pc+3 || pc+2 >= end || code[pc+2].op != opJmp {
		return nil, 0, false
	}
	lhs := stack[len(stack)-1].text
	common := jumpTarget(code[pc+2])
	if common <= target || common >= end || code[common].op != opAssign {
		return nil, 0, false
	}
	value, ok := evalExprRange(code, pc+1, pc+2, state)
	if !ok {
		return nil, 0, false
	}
	cases := []conditionalAssignmentCase{{condition: firstCondition, value: value}}
	pos := target
	defaultValue := ""
	for pos < common {
		branch := -1
		for i := pos; i+2 < common && i < pos+12; i++ {
			if code[i].op == opJne && jumpTarget(code[i]) == i+3 && code[i+2].op == opJmp && jumpTarget(code[i+2]) == common {
				branch = i
				break
			}
		}
		if branch < 0 {
			defaultValue, ok = evalExprRange(code, pos, common, state)
			if !ok {
				return nil, 0, false
			}
			break
		}
		condition, ok := evalExprRange(code, pos, branch, state)
		if !ok {
			return nil, 0, false
		}
		value, ok := evalExprRange(code, branch+1, branch+2, state)
		if !ok {
			return nil, 0, false
		}
		cases = append(cases, conditionalAssignmentCase{condition: condition, value: value})
		pos = jumpTarget(code[branch])
	}
	if defaultValue == "" || len(cases) == 0 {
		return nil, 0, false
	}
	lines := make([]string, 0, len(cases)*3+3)
	for i, c := range cases {
		if i == 0 {
			lines = append(lines, pad(indent)+"if ("+c.condition+") {")
		} else {
			lines = append(lines, pad(indent)+"else if ("+c.condition+") {")
		}
		lines = append(lines, pad(indent+1)+lhs+" = "+c.value+";")
		lines = append(lines, pad(indent)+"}")
	}
	lines = append(lines, pad(indent)+"else {")
	lines = append(lines, pad(indent+1)+lhs+" = "+defaultValue+";")
	lines = append(lines, pad(indent)+"}")
	return lines, common, true
}

func evalExprRange(code []instruction, start, end int, state *decompileState) (string, bool) {
	var stack []expr
	for pc := start; pc < end; pc++ {
		ins := code[pc]
		switch ins.op {
		case opPushArray:
			stack = append(stack, expr{marker: true})
		case opPushString:
			stack = append(stack, expr{text: quote(ins.operand.str), kind: "string"})
		case opPushVariable:
			stack = append(stack, expr{text: variableName(ins.operand.str)})
		case opPushNumber:
			stack = append(stack, expr{text: numberText(ins.operand)})
		case opPushTrue:
			stack = append(stack, expr{text: "true"})
		case opPushFalse:
			stack = append(stack, expr{text: "false"})
		case opPushNull:
			stack = append(stack, expr{text: "null"})
		case opPi:
			stack = append(stack, expr{text: "pi"})
		case opThis:
			stack = append(stack, expr{text: "this"})
		case opThisO:
			stack = append(stack, expr{text: "thiso"})
		case opTemp:
			stack = append(stack, expr{text: "temp"})
		case opPlayer:
			stack = append(stack, expr{text: "player"})
		case opPlayerO:
			stack = append(stack, expr{text: "playero"})
		case opLevel:
			stack = append(stack, expr{text: "level"})
		case opParams:
			stack = append(stack, expr{text: "params"})
		case opGetRegister:
			id := operandNumber(ins)
			if state != nil {
				if item, ok := state.registers[id]; ok {
					stack = append(stack, item)
					break
				}
			}
			stack = append(stack, expr{text: fmt.Sprintf("reg%d", id)})
		case opJne, opJeq:
			target := jumpTarget(ins)
			condition := popExpr(&stack).text
			value, newPC, ok := recoverTernaryExpression(code, pc, target, end, state, condition, ins.op)
			if !ok {
				return "", false
			}
			stack = append(stack, expr{text: value})
			pc = newPC
		case opConvertToFloat, opConvertToString, opConvertToObject, opConvertToVar, opEndParams, opShortCircuitEnd:
		case opEndArray:
			args := collectArgs(&stack)
			stack = append(stack, expr{text: "{" + strings.Join(args, ", ") + "}"})
		case opAccessMember:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: memberBase(lhs.text) + "." + memberName(rhs.text)})
		case opArrayAccess:
			index, arr := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: arr.text + "[" + index.text + "]"})
		case opAdd, opSubtract, opMultiply, opDivide, opModulo, opPower, opBoolAnd, opBoolOr, opEqual, opNotEqual, opLessThan, opGreaterThan, opLE, opGE, opBitwiseOr, opBitwiseAnd, opBitwiseXor, opShiftLeft, opShiftRight, opIn, opJoin, opAppend:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: lhs.text + " " + infix(ins.op) + " " + rhs.text})
		default:
			return "", false
		}
	}
	if len(stack) != 1 {
		return "", false
	}
	return stack[0].text, true
}

func recoverTailDispatch(code []instruction, pc, end, indent int, state *decompileState) ([]string, int, bool) {
	cases, tail, ok := parseTailDispatchCases(code, pc, end, state)
	if !ok || len(cases) == 0 {
		return nil, 0, false
	}
	commonEnd, ok := dispatchCommonEnd(code, cases, pc)
	if !ok || commonEnd < pc || commonEnd > end {
		return nil, 0, false
	}
	targets := make([]int, 0, len(cases))
	seenTargets := map[int]bool{}
	for _, c := range cases {
		if seenTargets[c.target] {
			continue
		}
		seenTargets[c.target] = true
		targets = append(targets, c.target)
	}
	sort.Ints(targets)
	targetToNext := map[int]int{}
	for i, t := range targets {
		next := pc
		if i+1 < len(targets) {
			next = targets[i+1]
		}
		targetToNext[t] = next
	}
	var lines []string
	for i, c := range cases {
		bodyEnd := targetToNext[c.target]
		if bodyEnd <= c.target {
			return nil, 0, false
		}
		body := removeDuplicateGotos(decompileRangeWithState(code, c.target, bodyEnd, indent+1, state))
		body = trimTrailingGoto(body, commonEnd)
		if i == 0 {
			lines = append(lines, pad(indent)+"if ("+c.condition+") {")
		} else {
			lines = append(lines, pad(indent)+"else if ("+c.condition+") {")
		}
		lines = append(lines, body...)
		lines = append(lines, pad(indent)+"}")
	}
	return lines, skipDispatchTail(code, tail, commonEnd, end), true
}

func parseTailDispatchCases(code []instruction, pc, end int, state *decompileState) ([]dispatchCase, int, bool) {
	selector, pos, ok := dispatchSelector(code, pc, state)
	if !ok {
		return nil, 0, false
	}
	var cases []dispatchCase
	for pos+4 < end {
		if code[pos].op != opCopy || code[pos+2].op != opEqual {
			break
		}
		lit, ok := dispatchLiteral(code[pos+1])
		if !ok {
			break
		}
		jump := code[pos+3]
		if jump.op != opJeq && jump.op != opJne {
			break
		}
		caseTarget := jumpTarget(jump)
		if caseTarget >= pc {
			pos += 4
			continue
		}
		if caseTarget < 0 {
			break
		}
		condition := selector + " == " + lit
		if jump.op == opJne {
			condition = selector + " != " + lit
		}
		cases = append(cases, dispatchCase{condition: condition, target: caseTarget})
		pos += 4
	}
	if len(cases) == 0 {
		return nil, 0, false
	}
	if pos < end && code[pos].op == opPop {
		pos++
	}
	return cases, pos, true
}

func parseBackwardDispatchCases(code []instruction, pc, target, end int, state *decompileState) ([]dispatchCase, int, bool) {
	selector, pos, ok := dispatchSelector(code, target, state)
	if !ok {
		return nil, 0, false
	}
	var cases []dispatchCase
	for pos+4 < end {
		if code[pos].op != opCopy || code[pos+2].op != opEqual {
			break
		}
		lit, ok := dispatchLiteral(code[pos+1])
		if !ok {
			break
		}
		jump := code[pos+3]
		if jump.op != opJeq && jump.op != opJne {
			break
		}
		caseTarget := jumpTarget(jump)
		if len(cases) == 0 && caseTarget > target {
			pos += 4
			continue
		}
		if caseTarget <= pc || caseTarget >= target {
			break
		}
		condition := selector + " == " + lit
		if jump.op == opJne {
			condition = selector + " != " + lit
		}
		cases = append(cases, dispatchCase{condition: condition, target: caseTarget})
		pos += 4
	}
	if len(cases) == 0 {
		return nil, 0, false
	}
	if pos < end && code[pos].op == opPop {
		pos++
	}
	return cases, pos, true
}

func dispatchSelector(code []instruction, target int, state *decompileState) (string, int, bool) {
	var stack []expr
	for pos := target; pos < len(code); pos++ {
		ins := code[pos]
		if ins.op == opCopy {
			if len(stack) != 1 {
				return "", 0, false
			}
			return stack[0].text, pos, true
		}
		switch ins.op {
		case opPushArray:
			stack = append(stack, expr{marker: true})
		case opPushVariable:
			if ins.operand == nil {
				return "", 0, false
			}
			stack = append(stack, expr{text: variableName(ins.operand.str)})
		case opPushString:
			if ins.operand == nil {
				return "", 0, false
			}
			stack = append(stack, expr{text: quote(ins.operand.str), kind: "string"})
		case opPushNumber:
			stack = append(stack, expr{text: numberText(ins.operand)})
		case opThis:
			stack = append(stack, expr{text: "this"})
		case opThisO:
			stack = append(stack, expr{text: "thiso"})
		case opTemp:
			stack = append(stack, expr{text: "temp"})
		case opPlayer:
			stack = append(stack, expr{text: "player"})
		case opPlayerO:
			stack = append(stack, expr{text: "playero"})
		case opLevel:
			stack = append(stack, expr{text: "level"})
		case opParams:
			stack = append(stack, expr{text: "params"})
		case opGetRegister:
			id := operandNumber(ins)
			if state != nil {
				if item, ok := state.registers[id]; ok {
					stack = append(stack, item)
					break
				}
			}
			stack = append(stack, expr{text: fmt.Sprintf("reg%d", id)})
		case opConvertToFloat, opConvertToString, opConvertToObject, opConvertToVar, opEndParams:
		case opCall:
			stack = append(stack, expr{text: buildCall(&stack), kind: "call"})
		case opObjSubstring:
			stack = append(stack, objectCall(&stack, "substring", 2, false))
		case opInt:
			stack = append(stack, functionCall(&stack, "int", 1))
		case opRandom:
			stack = append(stack, functionCall(&stack, "random", 2))
		case opAdd, opSubtract, opMultiply, opDivide, opModulo, opPower, opBoolAnd, opBoolOr, opEqual, opNotEqual, opLessThan, opGreaterThan, opLE, opGE, opBitwiseOr, opBitwiseAnd, opBitwiseXor, opShiftLeft, opShiftRight, opIn, opJoin, opAppend:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: lhs.text + " " + infix(ins.op) + " " + rhs.text})
		case opAccessMember:
			rhs, lhs := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: memberBase(lhs.text) + "." + memberName(rhs.text)})
		case opArrayAccess:
			index, arr := popExpr(&stack), popExpr(&stack)
			stack = append(stack, expr{text: arr.text + "[" + index.text + "]"})
		default:
			return "", 0, false
		}
	}
	return "", 0, false
}

func dispatchCommonEnd(code []instruction, cases []dispatchCase, dispatchStart int) (int, bool) {
	commonEnd := -1
	for _, c := range cases {
		limit := dispatchStart
		for _, other := range cases {
			if other.target > c.target && other.target < limit {
				limit = other.target
			}
		}
		endJump := -1
		for i := c.target; i < limit; i++ {
			if code[i].op == opJmp && jumpTarget(code[i]) >= dispatchStart {
				endJump = jumpTarget(code[i])
				break
			}
		}
		if endJump < 0 {
			return 0, false
		}
		if commonEnd < 0 {
			commonEnd = endJump
		} else if commonEnd != endJump {
			return 0, false
		}
	}
	return commonEnd, commonEnd >= 0
}

func trimTrailingGoto(body []string, target int) []string {
	if len(body) == 0 {
		return body
	}
	want := fmt.Sprintf("goto label_%d;", target)
	if strings.TrimSpace(body[len(body)-1]) == want {
		return body[:len(body)-1]
	}
	return body
}

func normalizeAssignmentValue(lhs, rhs expr) expr {
	if !isExtentField(lhs.text) {
		return rhs
	}
	a, b, ok := parseNumericPairLiteral(rhs.text)
	if !ok {
		return rhs
	}
	rhs.text = "{" + b + ", " + a + "}"
	return rhs
}

func isExtentField(name string) bool {
	last := name
	if idx := strings.LastIndex(last, "."); idx >= 0 {
		last = last[idx+1:]
	}
	switch strings.ToLower(last) {
	case "clientextent", "extent", "minextent":
		return true
	default:
		return false
	}
}

func parseNumericPairLiteral(value string) (string, string, bool) {
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return "", "", false
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}"))
	parts := strings.Split(body, ",")
	if len(parts) != 2 {
		return "", "", false
	}
	a := strings.TrimSpace(parts[0])
	b := strings.TrimSpace(parts[1])
	if !isNumberLiteral(a) || !isNumberLiteral(b) {
		return "", "", false
	}
	return a, b, true
}

func isNumberLiteral(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func skipDispatchTail(code []instruction, tail, commonEnd, end int) int {
	pos := tail
	if pos+1 < end && code[pos].op == opPushNumber && operandNumber(code[pos]) == 0 && code[pos+1].op == opRet {
		pos += 2
	}
	if commonEnd > pos {
		pos = commonEnd
		if pos < end && code[pos].op == opPop {
			pos++
		}
		if pos+1 < end && code[pos].op == opPushNumber && operandNumber(code[pos]) == 0 && code[pos+1].op == opRet {
			pos += 2
		}
	}
	return pos - 1
}

func trimForEachBookkeeping(body []string) []string {
	out := append([]string(nil), body...)
	if len(out) > 0 && isGotoLine(strings.TrimSpace(out[len(out)-1])) {
		out = out[:len(out)-1]
	}
	if len(out) > 0 {
		last := strings.TrimSpace(out[len(out)-1])
		if strings.HasSuffix(last, " += 1;") || strings.HasSuffix(last, " -= 1;") {
			out = out[:len(out)-1]
		}
	}
	return out
}

func trimAfterReturn(body []string) []string {
	for i, line := range body {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "return") && strings.HasSuffix(trimmed, ";") {
			return body[:i+1]
		}
	}
	return body
}

func isGotoLine(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "goto label_") || !strings.HasSuffix(line, ";") {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(line, "goto label_"), ";"))
	return err == nil
}

func isJumpPadding(code []instruction, start, end int) bool {
	if start >= end {
		return false
	}
	for i := start; i < end; i++ {
		if code[i].op != opJmp && code[i].op != opNone {
			return false
		}
	}
	return true
}

func recoverProfileCloneBlocks(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		name, _, indent, ok := parseProfileCloneAssignment(lines[i])
		if !ok {
			out = append(out, lines[i])
			continue
		}
		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "with ("+quote(name)+") {" {
			blockEnd := matchingBlockEnd(lines, i+1)
			if blockEnd > i+1 && blockEnd+1 < len(lines) && strings.TrimSpace(lines[blockEnd+1]) == "addcontrol("+quote(name)+");" {
				out = append(out, strings.Repeat(" ", indent)+"new GuiControlProfile("+quote(name)+") {")
				sourceFieldIndent := parseLineIndent(lines[i+1]) + 2
				targetFieldIndent := indent + 2
				for _, field := range lines[i+2 : blockEnd] {
					out = append(out, reindentBlockLine(field, sourceFieldIndent, targetFieldIndent))
				}
				out = append(out, strings.Repeat(" ", indent)+"}")
				out = append(out, lines[blockEnd+1])
				i = blockEnd + 1
				continue
			}
		}
		addIdx := -1
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "addcontrol("+quote(name)+");" {
				addIdx = j
				break
			}
			if parseLineIndent(lines[j]) != indent || strings.HasSuffix(trimmed, "{") || strings.HasPrefix(trimmed, "}") {
				break
			}
		}
		if addIdx < 0 {
			out = append(out, lines[i])
			continue
		}
		out = append(out, strings.Repeat(" ", indent)+"new GuiControlProfile("+quote(name)+") {")
		for _, field := range lines[i+1 : addIdx] {
			out = append(out, strings.Repeat(" ", indent)+pad(1)+strings.TrimSpace(field))
		}
		out = append(out, strings.Repeat(" ", indent)+"}")
		out = append(out, lines[addIdx])
		i = addIdx
	}
	return out
}

func matchingBlockEnd(lines []string, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasSuffix(trimmed, "{") {
			depth++
		}
		if strings.HasPrefix(trimmed, "}") {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func reindentBlockLine(line string, sourceFieldIndent, targetFieldIndent int) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	return strings.Repeat(" ", targetFieldIndent+max(0, parseLineIndent(line)-sourceFieldIndent)) + trimmed
}

func parseProfileCloneAssignment(line string) (string, string, int, bool) {
	indent := parseLineIndent(line)
	trimmed := strings.TrimSpace(line)
	if !strings.HasSuffix(trimmed, ";") || !strings.Contains(trimmed, " = ") {
		return "", "", 0, false
	}
	parts := strings.SplitN(strings.TrimSuffix(trimmed, ";"), " = ", 2)
	if len(parts) != 2 || !isQuotedProfileName(parts[0]) || !isQuotedProfileName(parts[1]) {
		return "", "", 0, false
	}
	name := unquoteText(parts[0])
	base := unquoteText(parts[1])
	if !strings.HasSuffix(name, "Profile") || !strings.HasSuffix(base, "Profile") {
		return "", "", 0, false
	}
	return name, base, indent, true
}

func isQuotedProfileName(value string) bool {
	return strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && !strings.Contains(value, " @ ")
}

func parseLineIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func recoverBareConstructorBlocks(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		indent := parseLineIndent(lines[i])
		trimmed := strings.TrimSpace(lines[i])
		if !isBareGuiConstructorLine(trimmed) {
			out = append(out, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && isConstructorFieldLine(lines[end], indent) {
			end++
		}
		if end == i+1 {
			out = append(out, lines[i])
			continue
		}
		out = append(out, strings.Repeat(" ", indent)+strings.TrimSuffix(trimmed, ";")+" {")
		for _, field := range lines[i+1 : end] {
			out = append(out, strings.Repeat(" ", indent)+pad(1)+strings.TrimSpace(field))
		}
		out = append(out, strings.Repeat(" ", indent)+"}")
		i = end - 1
	}
	return out
}

func isBareGuiConstructorLine(line string) bool {
	return strings.HasPrefix(line, "new Gui") && strings.HasSuffix(line, ");") && !strings.Contains(line, "{")
}

func isConstructorFieldLine(line string, indent int) bool {
	if parseLineIndent(line) != indent {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasSuffix(trimmed, ";") || strings.HasPrefix(trimmed, "new ") || strings.HasPrefix(trimmed, "addcontrol(") || strings.Contains(trimmed, "goto label_") {
		return false
	}
	if strings.Contains(trimmed, " = ") {
		lhs := strings.TrimSpace(strings.SplitN(trimmed, " = ", 2)[0])
		return !strings.ContainsAny(lhs, " (){}")
	}
	return false
}

func isTerminalRet(code []instruction, pc int, end int) bool {
	for i := pc + 1; i < end; i++ {
		if code[i].op == opJmp && jumpTarget(code[i]) >= end {
			continue
		}
		return false
	}
	return true
}

func removeDuplicateGotos(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(out) > 0 && strings.TrimSpace(line) == strings.TrimSpace(out[len(out)-1]) && isGotoLine(line) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func removeRepeatedAssignmentRuns(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		bestLen := 0
		for n := 2; n <= 8 && i+2*n <= len(lines); n++ {
			if sameAssignmentRun(lines[i:i+n], lines[i+n:i+2*n]) {
				bestLen = n
			}
		}
		if bestLen > 0 {
			out = append(out, lines[i:i+bestLen]...)
			i += bestLen * 2
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return out
}

func sameAssignmentRun(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) || !isSimpleAssignmentLine(a[i]) {
			return false
		}
	}
	return true
}

func isSimpleAssignmentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasSuffix(trimmed, ";") && strings.Contains(trimmed, " = ") && !strings.HasPrefix(trimmed, "if ") && !strings.HasPrefix(trimmed, "for ")
}

func recoverForwardGotoGuards(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		cond, _, indent, ok := parseGotoIfLine(lines[i])
		if !ok || i+1 >= len(lines) {
			out = append(out, lines[i])
			continue
		}
		if blockEnd, ok := forwardGuardBlockEnd(lines, i+1, indent); ok {
			out = append(out, strings.Repeat(" ", indent)+"if (!("+cond+")) {")
			for _, line := range recoverForwardGotoGuards(lines[i+1 : blockEnd]) {
				out = append(out, reindentBlockLine(line, indent, indent+2))
			}
			out = append(out, strings.Repeat(" ", indent)+"}")
			i = blockEnd - 1
			continue
		}
		if !isSimpleStatementLine(lines[i+1], indent) {
			out = append(out, lines[i])
			continue
		}
		out = append(out, strings.Repeat(" ", indent)+"if (!("+cond+")) {")
		out = append(out, strings.Repeat(" ", indent)+pad(1)+strings.TrimSpace(lines[i+1]))
		out = append(out, strings.Repeat(" ", indent)+"}")
		i++
	}
	return out
}

func recoverForwardGotoGuardsFixedPoint(lines []string) []string {
	for i := 0; i < 4; i++ {
		next := recoverForwardGotoGuards(lines)
		if strings.Join(next, "\n") == strings.Join(lines, "\n") {
			return next
		}
		lines = next
	}
	return lines
}

func forwardGuardBlockEnd(lines []string, start, indent int) (int, bool) {
	if start >= len(lines) || parseLineIndent(lines[start]) != indent || !strings.HasSuffix(strings.TrimSpace(lines[start]), "{") {
		return 0, false
	}
	end := matchingBlockEnd(lines, start)
	if end < 0 {
		return 0, false
	}
	return end + 1, end > start
}

func parseGotoIfLine(line string) (string, int, int, bool) {
	indent := parseLineIndent(line)
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "if (") || !strings.Contains(trimmed, ") goto label_") || !strings.HasSuffix(trimmed, ";") {
		return "", 0, 0, false
	}
	pivot := strings.LastIndex(trimmed, ") goto label_")
	if pivot < len("if (") {
		return "", 0, 0, false
	}
	labelText := strings.TrimSuffix(strings.TrimPrefix(trimmed[pivot+len(") goto label_"):], ""), ";")
	label, err := strconv.Atoi(labelText)
	if err != nil {
		return "", 0, 0, false
	}
	return trimmed[len("if ("):pivot], label, indent, true
}

func isSimpleStatementLine(line string, indent int) bool {
	if parseLineIndent(line) != indent {
		return false
	}
	trimmed := strings.TrimSpace(line)
	return strings.HasSuffix(trimmed, ";") && !strings.HasPrefix(trimmed, "goto label_") && !strings.HasPrefix(trimmed, "if ") && !strings.HasPrefix(trimmed, "for ")
}

func recoverSleepLoopBlocks(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		indent := parseLineIndent(lines[i])
		if strings.TrimSpace(lines[i]) != "if (true) {" {
			out = append(out, lines[i])
			continue
		}
		end := matchingBlockLine(lines, i)
		if end < 0 {
			out = append(out, lines[i])
			continue
		}
		body := lines[i+1 : end]
		recovered, ok := recoverSleepLoopBody(body, indent)
		if !ok {
			out = append(out, lines[i])
			continue
		}
		out = append(out, recovered...)
		i = end
	}
	return out
}

func recoverSleepLoopBody(body []string, indent int) ([]string, bool) {
	if len(body) < 4 {
		return nil, false
	}
	ifLine := strings.TrimSpace(body[len(body)-4])
	sleepLine := strings.TrimSpace(body[len(body)-3])
	gotoLine := strings.TrimSpace(body[len(body)-2])
	closeLine := strings.TrimSpace(body[len(body)-1])
	cond, ok := parseIfOpenCondition(ifLine)
	if !ok || !strings.HasPrefix(sleepLine, "sleep(") || !isGotoLine(gotoLine) || closeLine != "}" {
		return nil, false
	}
	out := []string{strings.Repeat(" ", indent) + "while (true) {"}
	out = append(out, body[:len(body)-4]...)
	out = append(out, strings.Repeat(" ", indent)+pad(1)+"if ("+cond+") {")
	out = append(out, strings.Repeat(" ", indent)+pad(2)+"break;")
	out = append(out, strings.Repeat(" ", indent)+pad(1)+"}")
	out = append(out, strings.Repeat(" ", indent)+pad(1)+sleepLine)
	out = append(out, strings.Repeat(" ", indent)+"}")
	return out, true
}

func parseIfOpenCondition(line string) (string, bool) {
	if !strings.HasPrefix(line, "if (") || !strings.HasSuffix(line, ") {") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(line, "if ("), ") {"), true
}

func matchingBlockLine(lines []string, start int) int {
	depth := 0
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasSuffix(trimmed, "{") {
			depth++
		}
		if trimmed == "}" {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func collapseNestedIfs(lines []string) []string {
	for {
		changed := false
		out := make([]string, 0, len(lines))
		for i := 0; i < len(lines); i++ {
			cond1, ok := parseIfLine(lines[i])
			if !ok || i+4 >= len(lines) || strings.TrimSpace(lines[i+1]) != "{" {
				out = append(out, lines[i])
				continue
			}
			closeOuter := matchingCloseBrace(lines, i+1)
			if closeOuter < 0 {
				out = append(out, lines[i])
				continue
			}
			cond2, ok := parseIfLine(lines[i+2])
			if !ok || strings.TrimSpace(lines[i+3]) != "{" {
				out = append(out, lines[i])
				continue
			}
			closeInner := matchingCloseBrace(lines, i+3)
			if closeInner != closeOuter-1 {
				out = append(out, lines[i])
				continue
			}
			indent := leadingWhitespace(lines[i])
			out = append(out, indent+"if ("+cond1+" && "+cond2+")")
			out = append(out, lines[i+1])
			out = append(out, unindentOnce(lines[i+4:closeInner])...)
			out = append(out, lines[closeOuter])
			i = closeOuter
			changed = true
		}
		lines = out
		if !changed {
			return lines
		}
	}
}

func parseIfLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "if (") || !strings.HasSuffix(trimmed, ")") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(trimmed, "if ("), ")"), true
}

func matchingCloseBrace(lines []string, openIndex int) int {
	depth := 0
	for i := openIndex; i < len(lines); i++ {
		switch strings.TrimSpace(lines[i]) {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

func unindentOnce(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimPrefix(line, "    ")
	}
	return out
}

func buildCall(stack *[]expr) string {
	callee := popExpr(stack).text
	args := collectCallArgs(stack)
	return callTarget(callee) + "(" + strings.Join(args, ", ") + ")"
}

func callTarget(s string) string {
	if strings.Contains(s, " @ ") && !(strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")) {
		return "(" + s + ")"
	}
	return s
}

func collectCallArgs(stack *[]expr) []string {
	args := []string{}
	for len(*stack) > 0 {
		item := popExpr(stack)
		if item.marker {
			break
		}
		args = append(args, item.text)
	}
	return args
}

func functionCall(stack *[]expr, name string, argc int) expr {
	args := fixedArgs(stack, argc)
	return expr{text: name + "(" + strings.Join(args, ", ") + ")", kind: "call"}
}

func objectCall(stack *[]expr, name string, argc int, statement bool) expr {
	args := fixedArgs(stack, argc)
	obj := popExpr(stack)
	kind := ""
	if statement {
		kind = "call"
	}
	return expr{text: memberBase(obj.text) + "." + name + "(" + strings.Join(args, ", ") + ")", kind: kind}
}

func multiDimArrayExpr(stack *[]expr) expr {
	return expr{text: multiDimTarget(stack)}
}

func multiDimTarget(stack *[]expr) string {
	parts := drainStack(stack)
	if len(parts) == 0 {
		return "/* missing */"
	}
	target := parts[0]
	for _, index := range parts[1:] {
		target += "[" + index + "]"
	}
	return target
}

func newMultiDimArrayExpr(stack *[]expr) expr {
	dims := drainStack(stack)
	if len(dims) == 0 {
		return expr{text: "new []"}
	}
	var out strings.Builder
	out.WriteString("new ")
	for _, dim := range dims {
		out.WriteString("[")
		out.WriteString(dim)
		out.WriteString("]")
	}
	return expr{text: out.String()}
}

func drainStack(stack *[]expr) []string {
	items := make([]string, 0, len(*stack))
	for len(*stack) > 0 {
		items = append(items, popExpr(stack).text)
	}
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items
}

func fixedArgs(stack *[]expr, argc int) []string {
	args := make([]string, 0, argc)
	for i := 0; i < argc; i++ {
		args = append(args, popExpr(stack).text)
	}
	for i, j := 0, len(args)-1; i < j; i, j = i+1, j-1 {
		args[i], args[j] = args[j], args[i]
	}
	return args
}

func collectArgs(stack *[]expr) []string {
	args := []string{}
	for len(*stack) > 0 {
		item := popExpr(stack)
		if item.marker {
			break
		}
		args = append(args, item.text)
	}
	for i, j := 0, len(args)-1; i < j; i, j = i+1, j-1 {
		args[i], args[j] = args[j], args[i]
	}
	return args
}

func infix(op opcode) string {
	switch op {
	case opAdd:
		return "+"
	case opSubtract:
		return "-"
	case opMultiply:
		return "*"
	case opDivide:
		return "/"
	case opModulo:
		return "%"
	case opPower:
		return "^"
	case opBoolAnd:
		return "&&"
	case opBoolOr:
		return "||"
	case opShortCircuitAnd:
		return "&&"
	case opShortCircuitOr:
		return "||"
	case opEqual:
		return "=="
	case opNotEqual:
		return "!="
	case opLessThan:
		return "<"
	case opGreaterThan:
		return ">"
	case opLE:
		return "<="
	case opGE:
		return ">="
	case opBitwiseOr:
		return "|"
	case opBitwiseAnd:
		return "&"
	case opBitwiseXor:
		return "^"
	case opShiftLeft:
		return "<<"
	case opShiftRight:
		return ">>"
	case opIn:
		return "in"
	case opJoin:
		return "@"
	case opAppend:
		return "@"
	default:
		return "?"
	}
}

func memberBase(s string) string {
	if strings.Contains(s, " @ ") && !(strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")) {
		return "(" + s + ")"
	}
	return s
}

func memberName(s string) string {
	if strings.Contains(s, " @ ") && !(strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")) {
		return "(" + s + ")"
	}
	if len(s) >= 2 && strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		name := strings.Trim(s, "\"")
		if isIdentifier(name) {
			return name
		}
	}
	return s
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return false
			}
			continue
		}
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func popExpr(stack *[]expr) expr {
	if len(*stack) == 0 {
		return expr{text: "/* missing */"}
	}
	last := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]
	return last
}

func jumpTarget(ins instruction) int {
	if ins.operand == nil {
		return ins.addr + 1
	}
	return ins.operand.number
}

func operandNumber(ins instruction) int {
	if ins.operand == nil {
		return 0
	}
	return ins.operand.number
}

func numberText(op *operand) string {
	if op == nil {
		return "0"
	}
	if op.kind == "float" {
		return op.float
	}
	return strconv.Itoa(op.number)
}

func quote(s string) string {
	return strconv.Quote(s)
}

func variableName(s string) string {
	if s == "unknown_object" {
		return "temp.object"
	}
	return s
}

func isUnknownObjectPlaceholder(s string) bool {
	return s == "unknown_object" || s == "temp.object"
}

func unquoteText(s string) string {
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	return s
}

func isConstructorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "new ") && strings.HasSuffix(trimmed, ");") {
		return true
	}
	return strings.Contains(trimmed, " = new Gui") && strings.HasSuffix(trimmed, ");")
}

func isAssignmentConstructorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, " = new ") && strings.HasSuffix(trimmed, ");")
}

func isNamedGuiConstruction(lhs, rhs expr) bool {
	if rhs.kind != "object" || !strings.HasPrefix(rhs.text, "new Gui") {
		return false
	}
	arg, ok := constructorExprArg(rhs.text)
	if !ok {
		return false
	}
	return strings.TrimSpace(lhs.text) == strings.TrimSpace(arg)
}

func isNamedProfileCloneConstruction(lhs, rhs expr) bool {
	if rhs.kind != "object" {
		return false
	}
	className, ok := constructorExprClass(rhs.text)
	if !ok || strings.HasPrefix(className, "Gui") || !strings.HasSuffix(className, "Profile") {
		return false
	}
	arg, ok := constructorExprArg(rhs.text)
	if !ok {
		return false
	}
	return strings.TrimSpace(lhs.text) == strings.TrimSpace(arg)
}

func constructorExprClass(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "new ") {
		return "", false
	}
	start := strings.Index(trimmed, "(")
	if start < 0 {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed[:start], "new ")), true
}

func constructorExprArg(value string) (string, bool) {
	start := strings.Index(value, "(")
	end := strings.LastIndex(value, ")")
	if start < 0 || end <= start {
		return "", false
	}
	return value[start+1 : end], true
}

func constructorLineMatchesTarget(line, target string) bool {
	if target == "/* missing */" {
		return true
	}
	trimmed := strings.TrimSpace(strings.TrimSuffix(line, ";"))
	if strings.HasPrefix(trimmed, "new ") {
		start := strings.Index(trimmed, "(")
		end := strings.LastIndex(trimmed, ")")
		return start >= 0 && end > start && unquoteText(trimmed[start+1:end]) == unquoteText(target)
	}
	if idx := strings.Index(trimmed, " = new "); idx > 0 {
		return strings.TrimSpace(trimmed[:idx]) == target
	}
	return false
}

func recoverFormatAssignment(lhs, rhs expr) (expr, expr, bool) {
	if lhs.text != "/* missing */" || !strings.HasPrefix(rhs.text, "format(") || !strings.HasSuffix(rhs.text, ")") {
		return lhs, rhs, false
	}
	args := splitTopLevelArgs(strings.TrimSuffix(strings.TrimPrefix(rhs.text, "format("), ")"))
	if len(args) < 2 {
		return lhs, rhs, false
	}
	if recoveredLHS, recoveredArg, ok := splitFormatAssignmentArg(args[0]); ok {
		args[0] = recoveredArg
		return expr{text: recoveredLHS}, expr{text: "format(" + strings.Join(args, ", ") + ")"}, true
	}
	args[0] = trimLeadingShortCircuit(args[0])
	if !isAssignableText(args[0]) {
		return lhs, rhs, false
	}
	return expr{text: args[0]}, expr{text: "format(" + strings.Join(args[1:], ", ") + ")"}, true
}

func splitFormatAssignmentArg(s string) (string, string, bool) {
	for _, op := range []string{" || ", " && "} {
		if idx := strings.Index(s, op); idx > 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])
			if isAssignableText(left) && right != "" {
				return left, right, true
			}
		}
	}
	return "", "", false
}

func trimLeadingShortCircuit(s string) string {
	s = strings.TrimSpace(s)
	for _, op := range []string{"|| ", "&& "} {
		if strings.HasPrefix(s, op) {
			return strings.TrimSpace(strings.TrimPrefix(s, op))
		}
	}
	return s
}

func recoverNewMultiDimAssignment(lhs, rhs expr) (expr, expr, bool) {
	if lhs.text != "/* missing */" || !strings.HasPrefix(rhs.text, "new [") {
		return lhs, rhs, false
	}
	target, dims, ok := splitNewMultiDim(rhs.text)
	if !ok || !isAssignableText(target) || len(dims) == 0 {
		return lhs, rhs, false
	}
	var out strings.Builder
	out.WriteString("new ")
	for _, dim := range dims {
		out.WriteString("[")
		out.WriteString(dim)
		out.WriteString("]")
	}
	return expr{text: target}, expr{text: out.String()}, true
}

func recoverSwappedBooleanAssignment(lhs, rhs expr) (expr, expr, bool) {
	if isAssignableText(lhs.text) || !isAssignableText(rhs.text) {
		return lhs, rhs, false
	}
	if strings.Contains(lhs.text, " || ") || strings.Contains(lhs.text, " && ") {
		return rhs, lhs, true
	}
	return lhs, rhs, false
}

func recoverEmbeddedBooleanAssignment(lhs, rhs expr) (expr, expr, bool) {
	target, rest, op, ok := splitBooleanAssignmentHead(lhs.text)
	if !ok {
		return lhs, rhs, false
	}
	return expr{text: target}, expr{text: target + op + rest + op + rhs.text}, true
}

func splitBooleanAssignmentHead(value string) (string, string, string, bool) {
	for _, op := range []string{" || ", " && "} {
		if idx := strings.Index(value, op); idx > 0 {
			left := strings.TrimSpace(value[:idx])
			right := strings.TrimSpace(value[idx+len(op):])
			if isAssignableText(left) && right != "" {
				return left, right, op, true
			}
		}
	}
	return "", "", "", false
}

func splitNewMultiDim(value string) (string, []string, bool) {
	parts, ok := splitBracketParts(value)
	if !ok || len(parts) < 2 {
		return "", nil, false
	}
	target := parts[0]
	dims := parts[1:]
	if strings.HasPrefix(target, "new [") {
		nestedTarget, nestedDims, nestedOK := splitNewMultiDim(target)
		if !nestedOK {
			return "", nil, false
		}
		target = nestedTarget
		dims = append(nestedDims, dims...)
	}
	for i, dim := range dims {
		dims[i] = unwrapSingleNewDim(dim)
	}
	return target, dims, true
}

func splitBracketParts(value string) ([]string, bool) {
	if !strings.HasPrefix(value, "new ") {
		return nil, false
	}
	var parts []string
	for i := len("new "); i < len(value); {
		if value[i] != '[' {
			return nil, false
		}
		end := matchingBracket(value, i)
		if end < 0 {
			return nil, false
		}
		parts = append(parts, strings.TrimSpace(value[i+1:end]))
		i = end + 1
	}
	return parts, true
}

func matchingBracket(value string, start int) int {
	depth := 0
	for i := start; i < len(value); i++ {
		switch value[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func unwrapSingleNewDim(value string) string {
	parts, ok := splitBracketParts(value)
	if ok && len(parts) == 1 {
		return parts[0]
	}
	return value
}

func splitTopLevelArgs(s string) []string {
	var args []string
	var current strings.Builder
	depth := 0
	quoteChar := rune(0)
	escaped := false
	for _, r := range s {
		if quoteChar != 0 {
			current.WriteRune(r)
			if escaped {
				escaped = false
			} else if r == '\\' {
				escaped = true
			} else if r == quoteChar {
				quoteChar = 0
			}
			continue
		}
		switch r {
		case '"', '\'':
			quoteChar = r
			current.WriteRune(r)
		case '(', '[', '{':
			depth++
			current.WriteRune(r)
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	args = append(args, strings.TrimSpace(current.String()))
	return args
}

func isAssignableText(s string) bool {
	if s == "" || strings.Contains(s, "/* missing */") || strings.HasPrefix(s, "\"") {
		return false
	}
	return !strings.ContainsAny(s, "+-*/<>=!&|")
}

func isObjectNameExpr(value expr) bool {
	return value.kind == "string" || strings.Contains(value.text, " @ ")
}

func isConstructorTarget(lhs, rhs expr) bool {
	return rhs.kind == "class" && strings.HasPrefix(unquoteText(rhs.text), "Gui") && lhs.text != "" && lhs.text != "/* missing */"
}

func classAssignmentTarget(value expr) string {
	if value.kind == "string" {
		return unquoteText(value.text)
	}
	return value.text
}

func constructorArg(value expr) string {
	if isObjectNameExpr(value) {
		return value.text
	}
	if looksLikeGuiObjectName(value.text) {
		return quote(value.text)
	}
	return value.text
}

func looksLikeGuiObjectName(value string) bool {
	if value == "" || strings.ContainsAny(value, ".[]() @") {
		return false
	}
	return strings.Contains(value, "_") || value[0] >= 'A' && value[0] <= 'Z'
}

func pad(level int) string {
	return strings.Repeat("  ", level)
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) left() int {
	return len(r.data) - r.pos
}

func (r *byteReader) skip(n int) error {
	if n < 0 || r.left() < n {
		return io.ErrUnexpectedEOF
	}
	r.pos += n
	return nil
}

func (r *byteReader) u8() (byte, error) {
	if r.left() < 1 {
		return 0, io.ErrUnexpectedEOF
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *byteReader) u16() (uint16, error) {
	if r.left() < 2 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v, nil
}

func (r *byteReader) u32() (uint32, error) {
	if r.left() < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v, nil
}

func (r *byteReader) cstr() (string, error) {
	start := r.pos
	for r.pos < len(r.data) && r.data[r.pos] != 0 {
		r.pos++
	}
	if r.pos >= len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	s := string(r.data[start:r.pos])
	r.pos++
	return s, nil
}
