package fsb

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/stereo441.ogg
var stereo441 []byte

//go:embed testdata/mono22050.ogg
var mono22050 []byte

//go:embed testdata/mono12k.ogg
var mono12k []byte

//go:embed testdata/ch6.ogg
var ch6 []byte

func TestRemuxMatchesGuest(t *testing.T) {
	chained := append(append([]byte(nil), stereo441...), mono22050...)
	cases := []struct {
		name string
		in   []byte
		n    int
		sum  string
	}{
		{"stereo441", stereo441, 1056, "6dff76db7bba526531977dac2c27c4bad02515f5f82c7781e805acda8db08e31"},
		{"mono22050", mono22050, 384, "dac32455d8fdb16e64082e2960643de8c372d19715cd3baa99ee26b19fd64460"},
		{"mono12k", mono12k, 400, "46aaa5ff9c8ba8ef2abebbe891efed06c1f8c9301c7e84b2985769ce2b791b64"},
		{"ch6", ch6, 672, "96d0e5689d5bf09202102e0e13b72055dd91d4c323e8d4105dc536b4bf73b438"},
		{"chained", chained, 1360, "3ca61992e343b328f5ac7094569fc260c3a6128b5347aceb972c0aa428b24458"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := Remux(t.Context(), &out, bytes.NewReader(tc.in))
			require.NoError(t, err)
			require.Equal(t, tc.n, out.Len())
			sum := sha256.Sum256(out.Bytes())
			require.Equal(t, tc.sum, hex.EncodeToString(sum[:]))
		})
	}
}

func TestRemuxRejects(t *testing.T) {
	var dst bytes.Buffer
	require.Error(t, Remux(t.Context(), io.Discard, nil))
	require.Error(t, Remux(t.Context(), nil, bytes.NewReader(stereo441)))
	require.Error(t, Remux(t.Context(), &dst, bytes.NewReader(nil)))
	require.Error(t, Remux(t.Context(), &dst, bytes.NewReader([]byte("not ogg"))))
	require.Error(t, Remux(t.Context(), &dst, bytes.NewReader(stereo441[:20])))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.Error(t, Remux(ctx, &dst, bytes.NewReader(stereo441)))
	require.Equal(t, 0, dst.Len())
}

func TestNewReader(t *testing.T) {
	_, err := NewReader(t.Context(), nil)
	require.Error(t, err)
	rc, err := NewReader(t.Context(), bytes.NewReader(stereo441))
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	sum := sha256.Sum256(got)
	require.Equal(t, "6dff76db7bba526531977dac2c27c4bad02515f5f82c7781e805acda8db08e31", hex.EncodeToString(sum[:]))
}
