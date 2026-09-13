package setupdata

import (
	"fmt"
	"slices"
	"strings"
)

// Operation describes file work requested by the installer's metadata. Paths
// retain Inno constants so the extractor can keep {app}, {tmp}, and {src}
// separate without creating host directories or running installer commands.
type Operation struct {
	Kind                   string
	Source, Dest, Filter   string
	Program, Args, WorkDir string
	Optional               bool
}

type psSlot struct {
	value    psValue
	ref      uint32
	indirect bool
}
type psState struct {
	stack   []psSlot
	globals map[uint32]psValue
	params  []psValue
	flag    psValue
}

func (s *psState) clone() *psState {
	g := make(map[uint32]psValue, len(s.globals))
	for k, v := range s.globals {
		g[k] = v
	}
	return &psState{stack: slices.Clone(s.stack), globals: g, params: s.params, flag: s.flag}
}
func (s *psState) get(a uint32) psValue {
	for range 64 {
		if a < 0x40000000 {
			return s.globals[a]
		}
		if a < 0x60000000 {
			i := int(0x60000000-a) - 1
			if i < len(s.params) {
				return s.params[i]
			}
			return psValue{}
		}
		i := int(a - 0x60000000)
		if i >= len(s.stack) {
			return psValue{}
		}
		v := s.stack[i]
		if !v.indirect {
			return v.value
		}
		a = v.ref
	}
	return psValue{}
}
func (s *psState) set(a uint32, v psValue) {
	for range 64 {
		if a < 0x40000000 {
			s.globals[a] = v
			return
		}
		if a < 0x60000000 {
			return
		}
		i := int(a - 0x60000000)
		if i >= len(s.stack) {
			return
		}
		if !s.stack[i].indirect {
			s.stack[i].value = v
			return
		}
		a = s.stack[i].ref
	}
}
func (s *psState) value(o psOperand) psValue {
	if o.tag == 1 {
		return o.value
	}
	if o.tag == 0 {
		return s.get(o.address)
	}
	return psValue{}
}
func (s *psState) merge(other *psState) (bool, error) {
	if len(s.stack) != len(other.stack) {
		return false, fmt.Errorf("IFPS: inconsistent stack at branch join")
	}
	changed := false
	for i, a := range s.stack {
		b := other.stack[i]
		if a != b && a != (psSlot{}) {
			s.stack[i] = psSlot{}
			changed = true
		}
	}
	for k, a := range s.globals {
		if a != other.globals[k] && a.kind != 0 {
			s.globals[k] = psValue{}
			changed = true
		}
	}
	if s.flag != other.flag && s.flag.kind != 0 {
		s.flag = psValue{}
		changed = true
	}
	return changed, nil
}

func psSignature(f psProc) (name string, result bool, n int) {
	name = f.name
	d := f.decl
	if !f.external {
		return name, false, 0
	}
	if strings.HasPrefix(d, "dll:") {
		parts := strings.SplitN(d, "\x00", 3)
		if len(parts) != 3 || len(parts[2]) < 4 {
			return name, false, 0
		}
		name = parts[1]
		d = parts[2][3:]
	} else if strings.HasPrefix(d, "class:") {
		return name, false, 0
	}
	if d == "" {
		return name, false, 0
	}
	return name, d[0] != 0, len(d) - 1
}

type psCall struct {
	name   string
	values []psValue
}

// Constant propagation visits both successors of unknown conditions and joins
// their values. It records calls only after convergence. No branch is chosen
// from an unknown value, and unresolved reconstruction arguments are rejected.
func (p *psProgram) calls(index int, params []psValue, depth int) ([]psCall, error) {
	if depth > 32 {
		return nil, fmt.Errorf("IFPS: recursive reconstruction metadata")
	}
	ins, err := p.instructions(p.procs[index].code)
	if err != nil {
		return nil, err
	}
	if len(ins) == 0 {
		return nil, nil
	}
	byOffset := make(map[int]int, len(ins))
	for i, x := range ins {
		byOffset[x.offset] = i
	}
	states := make([]*psState, len(ins))
	states[0] = &psState{stack: []psSlot{{}}, globals: map[uint32]psValue{}, params: params}
	queue := []int{0}
	steps := 0
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		steps++
		if steps > len(ins)*64 {
			return nil, fmt.Errorf("IFPS: reconstruction analysis did not converge")
		}
		s := states[i].clone()
		x := ins[i]
		next := []int{i + 1}
		get := func(j int) psValue { return s.value(x.operands[j]) }
		set := func(j int, v psValue) {
			if x.operands[j].tag == 0 {
				s.set(x.operands[j].address, v)
			}
		}
		switch x.op {
		case 0:
			set(0, get(1))
		case 1:
			a, b := get(0), get(1)
			v := psValue{}
			if a.kind == 's' && b.kind == 's' && x.sub == 0 && len(a.text)+len(b.text) <= 1<<20 {
				v = psValue{text: a.text + b.text, kind: 's'}
			}
			if a.kind == 'n' && b.kind == 'n' {
				if x.sub == 0 {
					v = psValue{number: a.number + b.number, kind: 'n'}
				}
				if x.sub == 1 {
					v = psValue{number: a.number - b.number, kind: 'n'}
				}
			}
			set(0, v)
		case 2:
			s.stack = append(s.stack, psSlot{value: get(0)})
		case 3:
			o := x.operands[0]
			s.stack = append(s.stack, psSlot{ref: o.address, indirect: o.tag == 0})
		case 11:
			s.stack = append(s.stack, psSlot{})
		case 4, 25, 26:
			n := 1
			if x.op == 26 {
				n = 2
			}
			if len(s.stack) <= n {
				return nil, fmt.Errorf("IFPS: stack underflow")
			}
			s.stack = s.stack[:len(s.stack)-n]
		case 5:
			if int(x.arg) >= len(p.procs) {
				return nil, fmt.Errorf("IFPS: invalid call")
			}
			f := p.procs[x.arg]
			name, result, n := psSignature(f)
			v := psValue{}
			args := psArguments(s, result, n)
			if len(args) > 0 && args[0].kind == 's' {
				a := args[0].text
				switch strings.ToUpper(name) {
				case "EXPANDCONSTANT":
					v = args[0]
				case "ADDQUOTES":
					v = psValue{kind: 's', text: "\"" + a + "\""}
				case "REMOVEQUOTES":
					v = psValue{kind: 's', text: strings.Trim(a, "\"")}
				case "EXTRACTFILEPATH":
					j := strings.LastIndexAny(a, "/\\")
					v = psValue{kind: 's', text: a[:j+1]}
				}
			}
			// Subsequent records describe the successful installation path. The
			// extractor checks each corresponding operation before advancing.
			if name == "ISExec" || name == "ISArcExtract" {
				v = psValue{kind: 'n', number: 1}
			}
			if result && len(s.stack) > 1 {
				s.set(0x60000000+uint32(len(s.stack)-1), v)
			}
		case 12:
			a, b := get(1), get(2)
			v := psValue{}
			if a.kind != 0 && a.kind == b.kind && (x.sub == 4 || x.sub == 5) {
				equal := a == b
				if x.sub == 4 {
					equal = !equal
				}
				v.kind = 'n'
				if equal {
					v.number = 1
				}
			}
			set(0, v)
		case 15:
			v := get(0)
			if v.kind == 'n' {
				if v.number == 0 {
					v.number = 1
				} else {
					v.number = 0
				}
			}
			set(0, v)
		case 16, 21, 23, 24:
			set(0, psValue{})
		case 14, 22:
			set(0, psValue{})
		case 17:
			s.flag = get(0)
			if s.flag.kind == 'n' && x.sub != 0 {
				if s.flag.number == 0 {
					s.flag.number = 1
				} else {
					s.flag.number = 0
				}
			}
		case 9:
			next = nil
		}
		if len(s.stack) > 65536 {
			return nil, fmt.Errorf("IFPS: excessive stack")
		}
		if x.op == 6 || x.op == 7 || x.op == 8 || x.op == 18 || x.op == 25 || x.op == 26 {
			target, ok := byOffset[x.end+int(int32(x.arg))]
			if !ok {
				return nil, fmt.Errorf("IFPS: procedure %d instruction %x: invalid branch target %x", index, x.offset, x.end+int(int32(x.arg)))
			}
			if x.op == 6 || x.op == 25 || x.op == 26 {
				next = []int{target}
			} else {
				v := s.flag
				if x.op == 7 || x.op == 8 {
					v = get(0)
					if x.op == 8 && v.kind == 'n' {
						if v.number == 0 {
							v.number = 1
						} else {
							v.number = 0
						}
					}
				}
				if v.kind == 'n' {
					if v.number != 0 {
						next = []int{target}
					}
				} else {
					next = append(next, target)
				}
			}
		}
		for _, j := range next {
			if j >= len(ins) {
				continue
			}
			if states[j] == nil {
				states[j] = s.clone()
				queue = append(queue, j)
			} else {
				changed, err := states[j].merge(s)
				if err != nil {
					return nil, err
				}
				if changed {
					queue = append(queue, j)
				}
			}
		}
	}
	var out []psCall
	for i, x := range ins {
		if x.op != 5 || states[i] == nil {
			continue
		}
		f := p.procs[x.arg]
		s := states[i]
		if !f.external {
			nested, err := p.calls(int(x.arg), psArguments(s, false, len(s.stack)-1), depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
			continue
		}
		name, result, n := psSignature(f)
		switch name {
		case "ISExec", "ISArcExtract", "EXEC", "SHELLEXEC", "DELETEFILE", "DELTREE":
			out = append(out, psCall{name, psArguments(s, result, n)})
		}
	}
	return out, nil
}

func psArguments(s *psState, result bool, n int) []psValue {
	end := len(s.stack) - 1
	if result {
		end--
	}
	if n < 0 || n > end {
		return nil
	}
	args := make([]psValue, n)
	for i := range args {
		args[i] = s.get(0x60000000 + uint32(end-i))
	}
	return args
}

func reconstructionPlan(data []byte) ([]Operation, error) {
	p, err := parseIFPS(data)
	if err != nil {
		return nil, err
	}
	var calls []psCall
	for i, f := range p.procs {
		if strings.EqualFold(f.name, "CURSTEPCHANGED") {
			// Inno's ssInstall callback parameter. Other lifecycle callbacks
			// describe UI, uninstall, and machine side effects.
			calls, err = p.calls(i, []psValue{{kind: 'n', number: 2}}, 0)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	var out []Operation
	for _, call := range calls {
		a := call.values
		text := func(i int) (string, error) {
			if i >= len(a) || a[i].kind != 's' {
				return "", fmt.Errorf("IFPS: unresolved %s argument %d", call.name, i)
			}
			return a[i].text, nil
		}
		var op Operation
		switch call.name {
		case "ISArcExtract":
			op.Kind = "extract"
			op.Source, err = text(2)
			if err != nil {
				return nil, err
			}
			op.Dest, err = text(3)
			if err != nil {
				return nil, err
			}
			op.Filter, err = text(4)
			if err != nil {
				return nil, err
			}
			if len(a) == 0 || a[0].kind != 'n' {
				return nil, fmt.Errorf("IFPS: unresolved optional-volume flag")
			}
			op.Optional = a[0].number != 0
		case "ISExec":
			op.Kind = "command"
			op.Program, err = text(3)
			if err != nil {
				return nil, err
			}
			op.Args, err = text(4)
			if err != nil {
				return nil, err
			}
			op.WorkDir, err = text(5)
			if err != nil {
				return nil, err
			}
		case "DELETEFILE", "DELTREE":
			// Ignore UI resource cleanup whose path is not a known constant.
			if len(a) == 0 || a[0].kind != 's' {
				continue
			}
			op.Kind = "remove"
			op.Source = a[0].text
		case "EXEC":
			op.Kind = "command"
			op.Program, err = text(0)
			if err != nil {
				return nil, err
			}
			op.Args, err = text(1)
			if err != nil {
				return nil, err
			}
			op.WorkDir, err = text(2)
			if err != nil {
				return nil, err
			}
		default:
			continue
		}
		out = append(out, op)
	}
	return out, nil
}
