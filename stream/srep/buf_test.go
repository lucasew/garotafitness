package srep

import "testing"

func TestClassIndex(t *testing.T) {
	t.Parallel()
	if classIndex(1) != 0 || classIndex(64) != 0 {
		t.Fatalf("64-class %d %d", classIndex(1), classIndex(64))
	}
	if classIndex(65) != 1 || classIndex(128) != 1 {
		t.Fatalf("128-class %d %d", classIndex(65), classIndex(128))
	}
	if classIndex(maxBlock) != numClass-1 {
		t.Fatalf("max %d want %d", classIndex(maxBlock), numClass-1)
	}
}

func TestGetPutSameClass(t *testing.T) {
	t.Parallel()
	a := getBuf(100)
	if cap(a) != 128 {
		t.Fatalf("cap %d", cap(a))
	}
	putBuf(a)
	b := getBuf(100)
	if cap(b) != 128 {
		t.Fatalf("cap %d", cap(b))
	}
	putBuf(b)
}
