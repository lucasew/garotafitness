package magic2

import (
	"io"
)

// decodeStream reads the remainder of a v22c4b solid after ParseHeader.
//
// Static read of cls-lolz_x64 / lolly v20d3 (setup.exe zlb, data only):
// rANS renormalize at 1<<23, FCM binary + nibble probability tables,
// LZ dictionary, optional ROLZ list (disabled in v22), optional ldmf.
// First payload word is a plausible big-endian rANS state
// (fg-06 0x20000000, fg-02 0xc0037700). Models are context-adaptive;
// a flat binary rANS does not yield the steam_emu.ini prefix.
func decodeStream(r io.Reader) ([]byte, error) {
	if _, err := io.Copy(io.Discard, r); err != nil {
		return nil, err
	}
	return nil, errBitstream
}
