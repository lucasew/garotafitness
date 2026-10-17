package gifrec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPCFFixtures locks Recompress to the Precomp 0.4.8 PCF bytes in
// stream/pref/testdata. Those streams use an empty GIFDiff and block size 255.
func TestPCFFixtures(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("..", "testdata")
	for _, name := range []string{"box.gif", "anim.gif", "noise.gif"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pcf, err := os.ReadFile(filepath.Join(dir, name+".pcf"))
			require.NoError(t, err)
			want, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			got := restoreGIFPCF(t, pcf)
			require.Equal(t, want, got)
		})
	}
}

func restoreGIFPCF(t *testing.T, in []byte) []byte {
	t.Helper()
	require.GreaterOrEqual(t, len(in), 8)
	require.Equal(t, "PCF", string(in[:3]))
	require.Equal(t, []byte{0, 4, 8, 0}, in[3:7])
	i := 7
	for i < len(in) && in[i] != 0 {
		i++
	}
	require.Less(t, i, len(in))
	c := in[i+1:]
	var out []byte
	for len(c) > 0 {
		hdr := c[0]
		c = c[1:]
		if hdr == 0 {
			n, rest := takeVlint(t, c)
			c = rest
			if n == 0 {
				break
			}
			require.GreaterOrEqual(t, len(c), n)
			out = append(out, c[:n]...)
			c = c[n:]
			continue
		}
		require.NotEmpty(t, c)
		kind := c[0]
		c = c[1:]
		require.Equal(t, byte(5), kind)
		block := 255
		if hdr&4 != 0 {
			block = 254
		}
		diffN, rest := takeVlint(t, c)
		c = rest
		require.GreaterOrEqual(t, len(c), diffN)
		diff := c[:diffN]
		c = c[diffN:]
		var penalty []byte
		if hdr&2 != 0 {
			n, rest := takeVlint(t, c)
			c = rest
			require.GreaterOrEqual(t, len(c), n)
			penalty = append([]byte(nil), c[:n]...)
			c = c[n:]
		}
		recomp, rest := takeVlint(t, c)
		c = rest
		decomp, rest := takeVlint(t, c)
		c = rest
		require.GreaterOrEqual(t, len(c), decomp)
		src := c[:decomp]
		c = c[decomp:]
		gif, err := Recompress(src, block, diff)
		require.NoError(t, err)
		if recomp < len(gif) {
			gif = gif[:recomp]
		}
		if penalty != nil {
			gif, err = ApplyPenalty(gif, penalty)
			require.NoError(t, err)
		}
		out = append(out, gif...)
	}
	return out
}

func takeVlint(t *testing.T, b []byte) (int, []byte) {
	t.Helper()
	var v, o uint64
	var s uint
	for i, c := range b {
		if c < 128 {
			n := v + o + (uint64(c) << s)
			require.Less(t, n, uint64(1<<31))
			return int(n), b[i+1:]
		}
		v += uint64(c&127) << s
		s += 7
		o = (o + 1) << 7
	}
	t.Fatal("truncated vlint")
	return 0, nil
}
