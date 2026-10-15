package xt3u

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/lucasew/garotafitness/internal/xtl"
	"github.com/stretchr/testify/require"
)

func TestNativeLZ4SHA256(t *testing.T) {
	t.Parallel()
	songs := bytes.Repeat([]byte("Songs of Conquest "), 64)
	requireLZ4(t, "hc12-songs", songs, true, 12, 33, "7125bb10d7d0b6fc3dd32af1fb5d7ecce8376b7e97d07a5e4a21a8dde33600b5")
	requireLZ4(t, "fast127-songs", songs, false, 127, 1158, "ccf6411921ca8e2dc0b334b2c52bf9e38bbb528d906393fa61bb6e59cd00672c")
	rep2 := bytes.Repeat([]byte("abc"), 1000)
	requireLZ4(t, "fast1-rep2", rep2, false, 1, 24, "d192c8a72f9b81ab706fb91bf3d91d1c9877670db05156f65cb0ff3da60b6ccd")
	requireLZ4(t, "fast4-rep2", rep2, false, 4, 27, "e6fd3c1ed9eef1692fb9956bed3275ec86c490b90fd17320280c8e406fb4d837")
	requireLZ4(t, "hc12-rep2", rep2, true, 12, 24, "d192c8a72f9b81ab706fb91bf3d91d1c9877670db05156f65cb0ff3da60b6ccd")
	k64 := make([]byte, 65536)
	for i := range k64 {
		k64[i] = byte(3 + i*0x11)
	}
	requireLZ4(t, "hc1-k64", k64, true, 1, 522, "849503747001a65674f951224cb75b177044f341c11332d21317b6208987d16f")
	requireLZ4(t, "fast1-k64", k64, false, 1, 522, "849503747001a65674f951224cb75b177044f341c11332d21317b6208987d16f")
	over := make([]byte, 65647)
	for i := range over {
		over[i] = byte(i * 13)
	}
	requireLZ4(t, "hc12-over", over, true, 12, 523, "1183f33f838b51d9bc4c216174c1582837e4930b327fafc9a65424d556aac1fc")
	requireLZ4(t, "fast1-over", over, false, 1, 523, "1183f33f838b51d9bc4c216174c1582837e4930b327fafc9a65424d556aac1fc")
	hard := hardInput(t)
	requireLZ4(t, "hc1-hard", hard, true, 1, 1157, "85a5a3d773b4566b86d2731a8dbe40a3fde5fe85e6e77fab5368e00afd82b35b")
	requireLZ4(t, "hc12-hard", hard, true, 12, 1154, "de243d9873252c8101d0922c9918fa96431a0b5df89e90acd53e3ccb4754d33f")
	requireLZ4(t, "fast1-hard", hard, false, 1, 1167, "9b0213057c0480e2f021a14614bf9fbd9aa535dbf9210bce1b805af19c5cf925")
	requireLZ4(t, "fast4-hard", hard, false, 4, 1218, "8ada065f4a5b4fa2a0e981daf2ed5808ed2c24a778a53d7d5de7941dc9b94fcd")
	requireLZ4(t, "empty", nil, true, 9, 1, "6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d")
	requireLZ4(t, "a", []byte("a"), false, 1, 2, "50ce734df436010eedd9c23b1b99535e7435808214559b6cfc1fa12c60f1eeb6")
}

func requireLZ4(t *testing.T, name string, src []byte, hc bool, param, n int, sum string) {
	t.Helper()
	var (
		got []byte
		err error
	)
	if hc {
		got, err = compressHC(src, param, 0)
	} else {
		got, err = compressFast(src, param, 0)
	}
	require.NoError(t, err)
	require.Len(t, got, n, name)
	require.Equal(t, sum, shaHex(got), name)
}

func TestNativeZstdPrefix(t *testing.T) {
	t.Parallel()
	hello, err := hex.DecodeString("28b52ffd241189000068656c6c6f207a737464207072656669789f94d2a1")
	require.NoError(t, err)
	out, err := decodePrefix(hello, nil, 32, 0)
	require.NoError(t, err)
	require.Equal(t, []byte("hello zstd prefix"), out)
	require.Equal(t, "60efbf558c4cedb268d275842d046264a080f86e54ec139a4d41a3f331be850e", shaHex(out))

	_, err = decodePrefix(nil, nil, 16, 0)
	require.EqualError(t, err, "xt3u: guest: zstd 0")
	_, err = decodePrefix(hello, nil, 32, 7)
	require.EqualError(t, err, "xt3u: guest: zstd -43")

	hard := hardInput(t)
	pre, err := compressHC(hard, 12, 0)
	require.NoError(t, err)
	require.Len(t, pre, 1154)
	require.Equal(t, 11, windowLog(len(pre)))
	patch, err := hex.DecodeString("28b52ffd648f03d50000701f50415443485441494c5441494c0200ef14c9953216a26636aefff166")
	require.NoError(t, err)
	want := append(append(append([]byte{}, pre[:400]...), []byte("PATCH")...), pre[400:]...)
	want = append(want, []byte("TAILTAIL")...)
	require.Len(t, want, 1167)
	got, err := decodePrefix(patch, pre, len(want), windowLog(len(pre)))
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, "b9657dcf1815d4d862b257d2bf0817c6979353ffca73198ce251d324c68c262e", shaHex(got))
	_, err = decodePrefix(patch, nil, len(want), 11)
	require.EqualError(t, err, "xt3u: guest: zstd -21")

	// bit 31 absent; extended ext is the patch. Restored bytes are the installed file.
	opt := int32(1 | (12 << 3))
	frame := xt3uFrame("lz4hc:l12", kindExtended, hard, patch, len(want), opt)
	r, err := NewReader(t.Context(), bytes.NewReader(frame))
	require.NoError(t, err)
	test.CloseOnCleanup(t, r)
	all, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, want, all)
}

func TestNativeFastReader(t *testing.T) {
	t.Parallel()
	raw := bytes.Repeat([]byte("abc"), 1000)
	comp, err := compressFast(raw, 4, 0)
	require.NoError(t, err)
	require.Len(t, comp, 27)
	opt := int32(4 << 7)
	frame := xt3uFrame("lz4", kindDefault, raw, nil, len(comp), opt)
	r, err := NewReader(t.Context(), bytes.NewReader(frame))
	require.NoError(t, err)
	test.CloseOnCleanup(t, r)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, comp, got)
	require.Equal(t, "e6fd3c1ed9eef1692fb9956bed3275ec86c490b90fd17320280c8e406fb4d837", shaHex(got))
}

func TestWindowLog(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, windowLog(0))
	require.Equal(t, 0, windowLog(-1))
	require.Equal(t, 1, windowLog(1))
	require.Equal(t, 2, windowLog(2))
	require.Equal(t, 10, windowLog(512))
	require.Equal(t, 11, windowLog(1024))
	require.Equal(t, 11, windowLog(1154))
}

func TestLZ4BlockBits(t *testing.T) {
	t.Parallel()
	raw := []byte("Songs of Conquest ")
	opt := int32(1 | (1 << 15))
	frame := xt3uFrame("lz4hc:l12", kindDefault, raw, nil, 1, opt)
	r, err := NewReader(t.Context(), bytes.NewReader(frame))
	require.NoError(t, err)
	test.CloseOnCleanup(t, r)
	_, err = io.ReadAll(r)
	require.EqualError(t, err, "xt3u: lz4hc block 1")

	opt = int32(1 << 15)
	frame = xt3uFrame("lz4", kindDefault, raw, nil, 1, opt)
	r, err = NewReader(t.Context(), bytes.NewReader(frame))
	require.NoError(t, err)
	test.CloseOnCleanup(t, r)
	_, err = io.ReadAll(r)
	require.EqualError(t, err, "xt3u: lz4 block 1")
}

func shaHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hardInput(t *testing.T) []byte {
	t.Helper()
	b, err := hex.DecodeString(hardHex)
	require.NoError(t, err)
	require.Len(t, b, 1994)
	return b
}

func xt3uFrame(method string, kind byte, raw, ext []byte, oldSize int, opt int32) []byte {
	payload := append([]byte{}, raw...)
	if kind&kindExtended == kindExtended {
		payload = append(payload, ext...)
		var sz [4]byte
		binary.LittleEndian.PutUint32(sz[:], uint32(len(ext)))
		payload = append(payload, sz[:]...)
	}
	var b bytes.Buffer
	b.WriteString(XTL0)
	putI32(&b, 1)
	b.WriteByte(byte(len(method)))
	b.WriteString(method)
	putI32(&b, 0)
	putI32(&b, -2)
	b.WriteByte(0)
	putI32(&b, 0)
	putI32(&b, 1)
	putI64(&b, int64(len(payload)))
	var h [xtl.StreamHeaderSize]byte
	h[0] = kind
	binary.LittleEndian.PutUint32(h[1:5], uint32(oldSize))
	binary.LittleEndian.PutUint32(h[5:9], uint32(len(payload)))
	h[13] = codecLZ4
	binary.LittleEndian.PutUint32(h[14:18], uint32(opt))
	b.Write(h[:])
	b.Write(payload)
	putU32(&b, 0)
	putU32(&b, 0)
	putI32(&b, 0)
	putI32(&b, int32(-1<<31))
	return b.Bytes()
}

const hardHex = "3535353535353535353535353535353535353535353535353535353535353535353535353535" +
	"353535353535353535353535353535353535353535353535353535353535353535353535a5ce" +
	"49152a10a5ce49152a10a5ce49152a10a5ce49152a10a5ce49152a10a5ce49152a10a5ce4915" +
	"2a10a5ce49152a10a5ce49152a10a5ce49152a10a5ce49152a10a5ce49152a10d23bd26aad2e" +
	"34468b78576ec4de6568058fd66e34b9aa8077ff6c1a3787dd331333a7a93a90327b9b2131c8" +
	"f54ca673427946141beff478d97e6f31aa599734e5e667f386f561e751b3dc86c75543fa3878" +
	"b3dc86c75543fa3878b3dc86c75543fa3878b3dc86c75543fa3878b3dc86c75543fa3878b3dc" +
	"86c75543fa3878b3dc86c75543fa3878b3dc86c75543fa3878b3dc86c75543fa3878b3dc86c7" +
	"5543fa3878b3dc86c75543fa38788d039ce46cca7394a10cb811da0c0b181bd099e7a90dc336" +
	"d78c16ad161fb23c0732116cd28f33375c3e4f7a76f785cc681af9267442c9ea217e743c4fde" +
	"fbd7836212c74ffc47189dc5f5e9d7aa76209979ae9b7ade8b95c2a5a36a309b9963347cd153" +
	"a16cd6ed7d8cbac821f3e131553d888704c7c9650c531ed4d9aadac21488f2072c124d7954aa" +
	"d94795f97e26894b637e44060ee28d9a0ac3ee55135504ccb52ea00dec7684c11edde6fa546e" +
	"38306fcca48d761ea38e2c5e37eb0bf4b58078797a58595a31323378797a58595a3132335a5a" +
	"5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a78797a58595a31323378797a58595a31" +
	"323378797a58595a31323378797a58595a31323378797a58595a31323378797a58595a313233" +
	"1a1b750cbd83e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83e81a1b75" +
	"0cbd83e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83" +
	"e81a1b750cbd83e81a1b750cbd83e81a1b750cbd83e8a9bf4222e13a314e54add46f80b4b582" +
	"0520d738d7eb2533e94bc35ed111b0d98e90482ae3a0601670e3d14511649169871cbb91c443" +
	"12629969e5960115dbdf055534bbd67689cdb54f2ea76e15c9c08ea8637912fb7e698f525e78" +
	"797a58595a31323378797a58595a313233282828282828282828282828282828282828282828" +
	"28282828282828282828282878797a58595a31323378797a58595a31323378797a58595a3132" +
	"3378797a58595a31323378797a58595a31323378797a58595a31323378797a58595a31323378" +
	"797a58595a31323378797a58595a31323378797a58595a3132334a4a4a4a4a4a4a4a4a4a4a4a" +
	"4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a4a166898f8c339b22deaba" +
	"721633b7ec619e94320122de66fd68facff2524f02e825d3a35b29aee962fad61a50809596df" +
	"00fc23f195efbbf5239d6bd6edb4e24af6b820836b4592295a02e9f78e5c02deb49adf181017" +
	"7fd82e17c0f06b3b880958f2182209804ae0516f7a70091fe5faa94d241f181c74cd8704fd85" +
	"334c28bda3666c997e505eb5223392d4d8824e38becb3d5f19d10f8ba178081c100b77a7392e" +
	"9183ee1d36d877878a38453cbdb988bb1b20d195b98f0346fa6826d9b125525a772d92c21db6" +
	"6eec67ef34e264b3a0ae0cd936a1c7d8bf7a43bfc0c690606a23c06a5d6218acc1203517ba4e" +
	"54b7ecd4ad9994a4da57e746ed47d1b4a23e0f4ab6a6683e941830e07508e8f3217926686a65" +
	"b6be03988f04ad1eb054d228dd4ae9f3a006bf0b2aeef8037e1d37c132d141f49bc502106f55" +
	"5aec5be7610517f0f8c689e8ad325714e5f8f588d9731710a7c3f8780b66ab634f965ddf3683" +
	"c46f20bdcb4cd2fa3587d8b6bbccb6d285036aeabb6d2fa206c0d668d97fd6a23b086a98266d" +
	"9a2b685178dea696507bfc0343f821019de2896547ae9c455ea5ce97b3e6f6d45ae86b87d6df" +
	"fb1faffbaf19a5fdbae0222f97dfaee939b1e4d310cbb303b50bf0d9701e9c636f3acf3c1b86" +
	"a3ad1ae74c09d080f68b7296537e66fb7c7c8ab060a64c20c463696ac353f89a28309d6f0e1b" +
	"b22ce6949ffcc08d71be37a6c9bd3c4af3c4b3884c45264e2f831670b6c7b236f00c67d20ad3" +
	"d97c3529acd49c6dfcec5892f0ba3200aeb1eb4d8c1a20e75cfc9a4d51247b52eb133db4abda" +
	"b37439d2f82def9b0faef53c9934be155c9f5daef472c2ac06beade468ead5a1dcdbf46151f5" +
	"1a6215fd005135533edb600a3edb600a3edb600a3edb600a78797a58595a31323378797a5859" +
	"5a313233d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7d7" +
	"d7d7d7d7d7d7d7d7d7d7d7d71e38386535352b28045c1e38386535352b28045c1e3838653535" +
	"2b28045c1e38386535352b28045c1e38386535352b28045c1e38386535352b28045c78797a58" +
	"595a31323378797a58595a31323378797a58595a31323378797a58595a31323378797a58595a" +
	"3132339672a48e42fd559672a48e42fd559672a48e42fd559672a48e42fd559672a48e42fd55" +
	"9672a48e42fd559672a48e42fd5540c9f2c21e5d40c9f2c21e5d40c9f2c21e5d40c9f2c21e5d" +
	"40c9f2c21e5d40c9f2c21e5d40c9f2c21e5d40c9f2c21e5d40c9f2c21e5d40c9f2c21e5d40c9" +
	"f2c21e5d90909090909090909090909090909090dbdbdbdbdbdbdbdb9c27620ae6c19c27620a" +
	"e6c19c27620ae6c19c27620ae6c19c27620ae6c19c27620ae6c19c27620ae6c19c27620ae6c1" +
	"78797a58595a31323378797a58595a313233"
