package jpgrec

import "fmt"

// RestorePackJPG expands a packJPG (PJG) payload into the merged JPEG.
// The in-memory path is the one used when the JPEG is at most JPG_MAX_MEMORY_SIZE.
func RestorePackJPG(payload []byte) ([]byte, error) {
	if len(payload) > jpgMaxMemory {
		return nil, fmt.Errorf("jpgrec: input length %d exceeds JPG_MAX_MEMORY_SIZE", len(payload))
	}
	return restorePJG(payload)
}

// RestoreBrunsli expands a Brunsli payload into the JPEG written by WriteJpeg.
// brotli selects the metadata section decoder used by BrunsliDecodeJpeg.
func RestoreBrunsli(payload []byte, brotli bool) ([]byte, error) {
	if len(payload) > jpgMaxMemory {
		return nil, fmt.Errorf("jpgrec: input length %d exceeds JPG_MAX_MEMORY_SIZE", len(payload))
	}
	return restoreBrunsli(payload, brotli)
}

// StripMotionDHT removes the motion-JPEG DHT that precomp inserts before SOS.
// recompressedLen is the original JPEG size; the writer buffer is that size plus MJPGDHT_LEN.
func StripMotionDHT(jpeg []byte, recompressedLen int) ([]byte, error) {
	if recompressedLen > jpgMaxMemory {
		return nil, fmt.Errorf("jpgrec: recompressed length %d exceeds JPG_MAX_MEMORY_SIZE", recompressedLen)
	}
	ffdaPos := -1
	foundFF := false
	found := false
	for !found {
		ffdaPos++
		if ffdaPos >= len(jpeg) {
			break
		}
		if foundFF {
			found = jpeg[ffdaPos] == 0xDA
			if found {
				break
			}
			foundFF = false
		} else {
			foundFF = jpeg[ffdaPos] == 0xFF
		}
	}
	cut := ffdaPos - 1 - mjpgDHTLen
	if cut < 0 {
		return nil, fmt.Errorf("jpgrec: motion JPEG DHT does not fit before SOS")
	}
	end := recompressedLen + mjpgDHTLen
	if end > len(jpeg) || ffdaPos-1 > len(jpeg) {
		return nil, fmt.Errorf("jpgrec: JPEG shorter than recompressed length plus MJPGDHT_LEN")
	}
	out := make([]byte, 0, recompressedLen)
	out = append(out, jpeg[:cut]...)
	out = append(out, jpeg[ffdaPos-1:end]...)
	return out, nil
}
