package sevenz

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLzmaOracle(t *testing.T) {
	t.Parallel()
	enc, err := newLzmaEnc(t.Context())
	require.NoError(t, err)
	low := bytesRepeatABC(100 << 10)
	high := pyRand256(1, 100<<10)
	short := []byte("hello sevenz")
	cases := []struct {
		name string
		data []byte
		opts lzmaOpts
		n    int
		sum  string
	}{
		{"header-empty", nil, headerOpts(), 18, "d10a4d932930d4378422e8917683171a0b8b02416f865ab956ead19609cd1d0d"},
		{"header-short", short, headerOpts(), 30, "32f95f687acbd488daa200fd66affc35b993c5715dd306ac9f7585b94acb950b"},
		{"header-low", low, headerOpts(), 101, "ea69beedbd9cba2c33a07f8cdfa866832e759831b2db4d89f78fb128b5382005"},
		{"header-high", high, headerOpts(), 103837, "b20ab1c185143d6faa496117a75f029f8919f1ec5403d4d0af013f75346c7f1c"},
		{"file-short", short, fileOpts(roundDict(uint32(len(short)))), 30, "843282d420daff79125799663c94585bdee16f3d7dea7612ea87575e8dc27c90"},
		{"file-low", low, fileOpts(roundDict(uint32(len(low)))), 101, "ea69beedbd9cba2c33a07f8cdfa866832e759831b2db4d89f78fb128b5382005"},
		{"file-high", high, fileOpts(roundDict(uint32(len(high)))), 103891, "59bf4b510d12370b3a106a9c3849872d2e7ed0527e172201af75d06555d0f316"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, err := enc.encode(t.Context(), tc.data, tc.opts)
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
