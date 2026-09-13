package setupdata

import (
	"bytes"
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

func TestSetupReconstructionPlan(t *testing.T) {
	f, err := os.Open(rimworldSetup)
	if err != nil {
		t.Skip("corpus not mounted")
	}
	defer f.Close()
	info, err := Scan(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Operations) == 0 {
		t.Fatal("missing installer reconstruction metadata")
	}
	if info.ManifestPath == "" {
		t.Fatal("missing checksum destination metadata")
	}
	archives := 0
	filtered := 0
	optional := 0
	for _, op := range info.Operations {
		if op.Kind == "extract" {
			archives++
			if op.Filter != "" {
				filtered++
			}
			if op.Optional {
				optional++
			}
		}
	}
	if archives != 9 || filtered != 2 || optional != 1 {
		t.Fatalf("archives=%d filtered=%d optional=%d", archives, filtered, optional)
	}
}

func fixtureIFPS(source string, unknown bool) []byte {
	var code []byte
	u32 := func(v uint32) { code = binary.LittleEndian.AppendUint32(code, v) }
	pushType := func(t uint32) { code = append(code, 11); u32(t) }
	assign := func(slot uint32, v any) {
		code = append(code, 0, 0)
		u32(0x60000000 + slot)
		code = append(code, 1)
		switch v := v.(type) {
		case string:
			u32(1)
			u32(uint32(len(v)))
			code = append(code, v...)
		case uint32:
			u32(0)
			u32(v)
		}
	}
	args := []any{uint32(1), uint32(0), source, "{app}\\Game Assets", "payload", uint32(0), "{tmp}\\unarc.dll", "{tmp}\\arc.ini", "{tmp}", uint32(0)}
	pushType(0)
	for i := len(args) - 1; i >= 0; i-- {
		typ := uint32(0)
		if _, ok := args[i].(string); ok {
			typ = 1
		}
		pushType(typ)
		if !(unknown && i == 2) {
			assign(uint32(len(args)-i+1), args[i])
		}
	}
	code = append(code, 3, 0)
	u32(0x60000001)
	code = append(code, 5)
	u32(0)
	code = append(code, bytes.Repeat([]byte{4}, 12)...)
	code = append(code, 9)
	var b []byte
	put := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	text := func(s string) { put(uint32(len(s))); b = append(b, s...) }
	b = append(b, "IFPS"...)
	for _, n := range []uint32{23, 2, 2, 0, ^uint32(0), 0} {
		put(n)
	}
	b = append(b, 5)
	put(0)
	b = append(b, 10)
	put(0)
	b = append(b, 3, 0)
	text("dll:files:ISDone.dll\x00ISArcExtract\x00" + string([]byte{3, 1, 0, 1}) + string(make([]byte, 10)))
	b = append(b, 2)
	offset := len(b)
	put(0)
	put(uint32(len(code)))
	text("CURSTEPCHANGED")
	text("-1 @0")
	binary.LittleEndian.PutUint32(b[offset:], uint32(len(b)))
	return append(b, code...)
}

func TestPlanUsesEncodedNames(t *testing.T) {
	for _, name := range []string{"fg-content.bin", "fg-other-language.bin"} {
		b := fixtureIFPS("{src}\\"+name, false)
		plan, err := reconstructionPlan(b)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan) != 1 || plan[0].Source != "{src}\\"+name || plan[0].Dest != "{app}\\Game Assets" || plan[0].Filter != "payload" || !plan[0].Optional {
			t.Fatalf("%+v", plan)
		}
		for n := 0; n < len(b); n++ {
			if _, err := reconstructionPlan(b[:n]); err == nil {
				t.Fatalf("accepted truncation at %d", n)
			}
		}
	}
	if _, err := reconstructionPlan(fixtureIFPS("unused", true)); err == nil || !strings.Contains(err.Error(), "unresolved") {
		t.Fatalf("accepted unknown source: %v", err)
	}
}

func TestManifestDestinationFraming(t *testing.T) {
	var b []byte
	for _, c := range "{app}\\Checks\\files.md5" {
		b = binary.LittleEndian.AppendUint16(b, uint16(c))
	}
	data := binary.LittleEndian.AppendUint32(nil, uint32(len(b)))
	data = append(data, b...)
	if got := installedManifestPath(data); got != "{app}\\Checks\\files.md5" {
		t.Fatal(got)
	}
	data[0]++
	if got := installedManifestPath(data); got != "" {
		t.Fatalf("accepted invalid field framing: %s", got)
	}
}
