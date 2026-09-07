package magic2

// Constants taken from a static read of the official images
// (INV-03: bytes only). See pe_notes.go.

const (
	// ransL is the rANS renormalize threshold. v22c4b / v20d3 .text
	// compares the 32-bit state to this and shifts in a byte while below.
	ransL = 1 << 23

	// fcmNibbleBytes is the VirtualAlloc size chosen when the model
	// width test is 4 (nibble FCM). String next to the v20d3 alloc:
	// "fcm nibble probs".
	fcmNibbleBytes = 0x3420000

	// fcmBinaryBytes is the VirtualAlloc size chosen when the model
	// width test is 2 (binary FCM). String: "fcm binary probs".
	fcmBinaryBytes = 0x992200
)

// versionBanner is the NUL-terminated compile stamp in cls-magic2_x64
// (zlb@0x22600 PE at +0x319d95). Same text on the x86 twin and on the
// second copy shipped as magic2l.
var versionBanner = []byte(" v22c4b [Dec 30 2018  16:44:57]")

// lollyBanner is the older lollypop image (zlb@0x22600 PE at +0x269195).
var lollyBanner = []byte(" v20d3 [Oct 30 2017  17:30:22]")

// allocNames are the four heap labels in lolly v20d3 .text (no .rdata).
// v22c4b dropped the FCM/dict/rolz labels from the image but kept the
// two FCM alloc sizes.
var allocNames = [][]byte{
	[]byte("fcm nibble probs"),
	[]byte("fcm binary probs"),
	[]byte("dictionary"),
	[]byte("rolz list buffer"),
}

// optionHelpPrefix sits in v22c4b immediately after optionDefaults16.
var optionHelpPrefix = []byte("available options: \n")

// optionTable32 is the first 32 bytes of the 64-byte block that ends
// at optionHelpPrefix in cls-magic2_x64 .text. Layout as 8 little-endian
// uint32:
//
//	1, 0x00010001, 0x00000101, 0x00010101,
//	0, 0x00010000, 0x00000100, 0x00010100
//
// The next 16 bytes (not in this slice) repeat the first 16, then
// optionDefaults16. Packed as 16 flag bytes + 16 flag bytes, or as
// those uint32s; the image does not name the fields.
var optionTable32 = []byte{
	0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
	0x01, 0x01, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00,
}

// optionDefaults16 is the 16 bytes after the repeated flag row:
// uint32{2, 4, 4, 4}.
var optionDefaults16 = []byte{
	0x02, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x00, 0x00,
}
