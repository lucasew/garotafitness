package mpzz

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// FS25 fg-02 block 4 is one OGGRE buffer. The oracle is the HOST_DEBUG
// build of guest/kernel.cpp at d00b775^ on this buffer: output length
// 134217728 and IEEE CRC 0x6deceb68. Input is the carved buffer at
// offset 0, 104157861 bytes, SHA-256 below.
func TestFS25Block4Oracle(t *testing.T) {
	path := os.Getenv("GAROTAFITNESS_MPZZ_BLOCK4")
	if path == "" {
		t.Skip("set GAROTAFITNESS_MPZZ_BLOCK4 to the carved fg-02 block 4 OGGRE buffer")
	}
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, 104157861, len(src))
	sum := sha256.Sum256(src)
	require.Equal(t, "d6706fd4f8f325157842dcf3b075d4b9203c8a93289cf46ecb7a5206f209b7d1", hex.EncodeToString(sum[:]))
	out, err := DecodeGuest(t.Context(), src, 0)
	require.NoError(t, err)
	require.Equal(t, 134217728, len(out))
	require.Equal(t, uint32(0x6deceb68), crc32.ChecksumIEEE(out))
}
