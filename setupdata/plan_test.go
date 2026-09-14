package setupdata

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/lucasew/garotafitness/internal/corpus"
	"github.com/stretchr/testify/require"
)

func TestSetupReconstructionPlan(t *testing.T) {
	f := corpus.File(t, "setup.exe")
	info, err := Scan(f)
	require.NoError(t, err)
	require.NotEmpty(t, info.Operations, "installer reconstruction metadata")
	require.NotEmpty(t, info.ManifestPath, "checksum destination metadata")
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
	require.Equal(t, 9, archives)
	require.Equal(t, 2, filtered)
	require.Equal(t, 1, optional)
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
		require.NoError(t, err)
		require.Equal(t, []Operation{{Kind: "extract", Source: "{src}\\" + name, Dest: "{app}\\Game Assets", Filter: "payload", Optional: true}}, plan)
		for n := 0; n < len(b); n++ {
			_, err := reconstructionPlan(b[:n])
			require.Error(t, err, "accepted truncation at %d", n)
		}
	}
	_, err := reconstructionPlan(fixtureIFPS("unused", true))
	require.ErrorContains(t, err, "unresolved")
}

func TestManifestDestinationFraming(t *testing.T) {
	var b []byte
	for _, c := range "{app}\\Checks\\files.md5" {
		b = binary.LittleEndian.AppendUint16(b, uint16(c))
	}
	data := binary.LittleEndian.AppendUint32(nil, uint32(len(b)))
	data = append(data, b...)
	require.Equal(t, "{app}\\Checks\\files.md5", installedManifestPath(data))
	data[0]++
	require.Empty(t, installedManifestPath(data), "invalid field framing")
}
