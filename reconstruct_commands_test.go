package garotafitness

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lucasew/garotafitness/setupdata"
	"github.com/ulikunitz/xz/lzma"
)

func testPlan() *reconstructionPlan {
	return &reconstructionPlan{app: newStaging(), temp: newStaging(), volumes: map[string]Volume{}, remaining: map[string]int{}, decoded: map[string]*reconstruction{}, seen: map[string]bool{}}
}

func TestRecipeDrivenReconstruction(t *testing.T) {
	// These names, counts, locations, and compression options differ from the
	// corpus. The recipes supply every relationship between files.
	for _, count := range []int{1, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			p := testPlan()
			var recipe, moves strings.Builder
			for i := range count {
				name := fmt.Sprintf("chunk-%d", i)
				p.app.files["scratch/"+name+".raw"] = []byte("before")
				diff := binary.LittleEndian.AppendUint64(nil, 5)
				diff = binary.LittleEndian.AppendUint64(diff, 0)
				diff = append(diff, 5)
				diff = append(diff, "after"...)
				p.app.files["scratch/"+name+".patch"] = diff
				fmt.Fprintf(&recipe, "\"{tmp}\\engine\\x2.exe\" %s.raw %s.patch&&del %s.patch\n", name, name, name)
				fmt.Fprintf(&recipe, "cmd /c \"\"{tmp}\\engine\\fgpack.exe\" e -d16 -fb32 -lc2 -lp1 -pb1 %s.raw %s.packed&&del %s.raw\"\n", name, name, name)
				fmt.Fprintf(&moves, "move \"scratch\\%s.packed\" \"Assets\\chapter-%d.dat\"\n", name, i)
			}
			moves.WriteString("rd /q /s scratch\ndel \"installer note.txt\"\n")
			p.temp.files["engine/jobs.txt"] = []byte(recipe.String())
			p.temp.files["engine/build.bat"] = []byte("\"{tmp}\\engine\\run.exe\" \" \" https://example.invalid \"{tmp}\\engine\\jobs.txt\" 0\ndel \"{tmp}\\engine\\jobs.txt\"\n")
			p.temp.files["relocate.bat"] = []byte(moves.String())
			p.app.files["installer note.txt"] = []byte("temporary")
			ops := []setupdata.Operation{
				{Kind: "command", Program: "{tmp}\\engine\\build.bat", WorkDir: "{app}\\scratch"},
				{Kind: "command", Program: "{cmd}", Args: "/C call \"{tmp}\\relocate.bat\"", WorkDir: "{app}"},
			}
			if err := p.run(t.Context(), ops); err != nil {
				t.Fatal(err)
			}
			if len(p.app.files) != count {
				t.Fatalf("unexpected files: %v", sortedKeys(p.app.files))
			}
			for i := range count {
				b := p.app.files[fmt.Sprintf("Assets/chapter-%d.dat", i)]
				if len(b) < 13 || b[0] != (1*5+1)*9+2 || binary.LittleEndian.Uint32(b[1:]) != 1<<16 {
					t.Fatalf("recipe parameters ignored: %x", b)
				}
				r, err := lzma.NewReader(bytes.NewReader(b))
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(r)
				if err != nil || string(got) != "after" {
					t.Fatalf("%q %v", got, err)
				}
			}
		})
	}
}

func TestRecipeReplacementsAndManifestAppend(t *testing.T) {
	p := testPlan()
	p.temp.files["tasks/list.txt"] = []byte("ROOT\\payload")
	if err := p.command(t.Context(), "{tmp}\\fart.exe", "-w *.txt ROOT \"{app}\"", "tmp/tasks", 0); err != nil {
		t.Fatal(err)
	}
	if got := string(p.temp.files["tasks/list.txt"]); got != "{app}\\payload" {
		t.Fatal(got)
	}
	p.app.files["Verify/base.md5"] = []byte("first\n")
	p.app.files["Verify/extra.addon"] = []byte("second\n")
	if err := p.command(t.Context(), "{cmd}", "/C \"copy /b base.md5+*.addon&&del *.addon\"", "app/Verify", 0); err != nil {
		t.Fatal(err)
	}
	if string(p.app.files["Verify/base.md5"]) != "first\nsecond\n" {
		t.Fatal(p.app.files)
	}
	if _, ok := p.app.files["Verify/extra.addon"]; ok {
		t.Fatal("addon remains")
	}
}

func TestRecipeRejectsUnknownProgramsAndEscapes(t *testing.T) {
	for _, line := range []string{"unknown.exe input output", "del ../../escape", "del {src}\\fg-01.bin", "echo harmless | unknown.exe", "move missing destination"} {
		p := testPlan()
		if err := p.recipe(t.Context(), line, "app", 0); err == nil {
			t.Fatalf("accepted %q", line)
		}
	}
	for _, name := range []string{"{app}\\..\\src\\source.bin", "{tmp}\\..\\app\\file", "/etc/passwd", "C:\\escape"} {
		if _, err := virtualPath(name, ""); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}
