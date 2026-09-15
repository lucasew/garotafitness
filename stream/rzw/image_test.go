package rzw

import "testing"

func TestImgDotShift(t *testing.T) {
	// 0x404994: lea 0x200(%rax,%rcx); sar $10.
	if imgDot([16]int16{}, [8]int16{}, [8]int16{}) != 0 {
		t.Fatal("zero features")
	}
	var feat [16]int16
	feat[0] = 4
	var c0 [8]int16
	c0[0] = 256
	if imgDot(feat, c0, [8]int16{}) != (4*256+512)>>10 {
		t.Fatal("pmaddwd shift")
	}
}

func TestImgRiceAdapt(t *testing.T) {
	d := newDec(nil, 16)
	d.img.rice[0] = 2
	// v=9: 9 > 2<<2 (8) → ++
	if 9 <= 2<<2 {
		t.Fatal("fixture")
	}
	d.img.rice[0] = 2
	if 9 > 2<<2 {
		d.img.rice[0] = 3
	}
	if d.img.rice[0] != 3 {
		t.Fatal(d.img.rice[0])
	}
	d.img.rice[0] = 3
	if 4 < 1<<3 {
		d.img.rice[0] = 3 - (3+31)>>5
	}
	if d.img.rice[0] != 2 {
		t.Fatal(d.img.rice[0])
	}
}

func TestImgExtra20(t *testing.T) {
	// 1.00 0x42b720 / 1.03.7 file 0x29c60 VA 0x42c460.
	want := [20]byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if imgExtra20 != want {
		t.Fatalf("%v", imgExtra20)
	}
	if imgBase20[5] != 5 || imgBase20[6] != 7 {
		t.Fatalf("bases %v", imgBase20)
	}
}
