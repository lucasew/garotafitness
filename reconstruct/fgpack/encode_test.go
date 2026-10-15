package fgpack

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOracle(t *testing.T) {
	t.Parallel()
	low := bytesRepeatABC(100 << 10)
	high := pyRand256(1, 100<<10)
	cases := []struct {
		name string
		data []byte
		n    int
		sum  string
	}{
		{"empty", nil, 18, "6196a4d495c2801d2a776b4edf5f533ec31ad99d03ecb033d53dfdeecdd1d08a"},
		{"short", []byte("hello sevenz"), 30, "a820e3ca06fcd1ab2664cc63dc3edb19d9a241ce180168566a1e5027653ab36e"},
		{"low", low, 101, "250008959183807e7ff04f17dd44ca0735e06c1e83b1f86afa3af386acd61993"},
		{"high", high, 103839, "9b038f9a3681ff57f1774aecf557cc6f7279d4187136446201cf40d92ee53a06"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := Encode(t.Context(), tc.data)
			require.NoError(t, err)
			require.Len(t, out, tc.n)
			sum := sha256.Sum256(out)
			require.Equal(t, tc.sum, hex.EncodeToString(sum[:]))
		})
	}
}

func bytesRepeatABC(n int) []byte {
	out := make([]byte, n)
	pat := []byte("abc")
	for i := range out {
		out[i] = pat[i%3]
	}
	return out
}

// pyRand256 is CPython random.Random(seed).randrange(256) repeated n times.
func pyRand256(seed uint32, n int) []byte {
	r := newPyMT(seed)
	out := make([]byte, n)
	for i := range out {
		for {
			v := r.u32() >> 23
			if v < 256 {
				out[i] = byte(v)
				break
			}
		}
	}
	return out
}

const pyN, pyM = 624, 397

type pyMT struct {
	st [pyN]uint32
	i  int
}

func newPyMT(seed uint32) *pyMT {
	r := &pyMT{i: pyN}
	r.st[0] = 19650218
	for i := 1; i < pyN; i++ {
		r.st[i] = 1812433253*(r.st[i-1]^(r.st[i-1]>>30)) + uint32(i)
	}
	key := []uint32{seed}
	i, j := 1, 0
	for k := pyN; k > 0; k-- {
		r.st[i] = (r.st[i] ^ ((r.st[i-1] ^ (r.st[i-1] >> 30)) * 1664525)) + key[j] + uint32(j)
		i++
		j++
		if i >= pyN {
			r.st[0] = r.st[pyN-1]
			i = 1
		}
		if j >= len(key) {
			j = 0
		}
	}
	for k := pyN - 1; k > 0; k-- {
		r.st[i] = (r.st[i] ^ ((r.st[i-1] ^ (r.st[i-1] >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= pyN {
			r.st[0] = r.st[pyN-1]
			i = 1
		}
	}
	r.st[0] = 0x80000000
	return r
}

func (r *pyMT) u32() uint32 {
	if r.i >= pyN {
		for kk := 0; kk < pyN-pyM; kk++ {
			y := (r.st[kk] & 0x80000000) | (r.st[kk+1] & 0x7fffffff)
			r.st[kk] = r.st[kk+pyM] ^ (y >> 1)
			if y&1 != 0 {
				r.st[kk] ^= 0x9908b0df
			}
		}
		for kk := pyN - pyM; kk < pyN-1; kk++ {
			y := (r.st[kk] & 0x80000000) | (r.st[kk+1] & 0x7fffffff)
			r.st[kk] = r.st[kk+(pyM-pyN)] ^ (y >> 1)
			if y&1 != 0 {
				r.st[kk] ^= 0x9908b0df
			}
		}
		y := (r.st[pyN-1] & 0x80000000) | (r.st[0] & 0x7fffffff)
		r.st[pyN-1] = r.st[pyM-1] ^ (y >> 1)
		if y&1 != 0 {
			r.st[pyN-1] ^= 0x9908b0df
		}
		r.i = 0
	}
	y := r.st[r.i]
	r.i++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}
