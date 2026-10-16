package mp3rec

// Restore expands a packMP3 (magic 'M','S') payload to an MP3 byte stream.
// This is the in-memory D_MP3 path: pmplib_init_streams + pmplib_convert_stream2mem.
func Restore(payload []byte) ([]byte, error) {
	if len(payload) < 3 || payload[0] != 'M' || payload[1] != 'S' {
		return nil, errf("not a PMP payload")
	}
	if payload[2] != appVersion {
		return nil, errf("incompatible file")
	}
	s := &session{r: &memReader{b: payload, i: 3}}
	if err := s.readHeader(); err != nil {
		return nil, err
	}
	if s.nBadFirst > 0 {
		if err := s.unstoreUnmute(); err != nil {
			return nil, err
		}
	}
	dec := newArith(s.r)
	if s.dataBeforeN > 0 || s.dataAfterN > 0 {
		if err := s.decodeID3(dec); err != nil {
			return nil, err
		}
	}
	if s.iPadding != 0 {
		if err := s.decodePadding(dec); err != nil {
			return nil, err
		}
	}
	if s.iSblocks != 0 {
		if err := s.decodeBlockTypes(dec); err != nil {
			return nil, err
		}
	}
	if err := s.decodeGlobalGain(dec); err != nil {
		return nil, err
	}
	if err := s.buildContext(); err != nil {
		return nil, err
	}
	if err := s.decodeSlength(dec); err != nil {
		return nil, err
	}
	if err := s.decodeRegion(dec); err != nil {
		return nil, err
	}
	if s.iShare != 0 {
		if err := s.decodeSharing(dec); err != nil {
			return nil, err
		}
	}
	if s.iPreemphasis != 0 {
		if err := s.decodePreemphasis(dec); err != nil {
			return nil, err
		}
	}
	if s.iCoarse != 0 {
		if err := s.decodeCoarse(dec); err != nil {
			return nil, err
		}
	}
	if s.iSbGain != 0 {
		if err := s.decodeSubblock(dec); err != nil {
			return nil, err
		}
	}
	if s.iStereoMS != 0 {
		if err := s.decodeStereoMS(dec); err != nil {
			return nil, err
		}
	}
	main, err := s.decodeMain(dec)
	if err != nil {
		return nil, err
	}
	if err := s.unmuteFrames(); err != nil {
		return nil, err
	}
	return s.writeMP3(main)
}
