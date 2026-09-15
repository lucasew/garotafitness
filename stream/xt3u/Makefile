# Official LZ4 HC + ZSTD prefix-patch guest. -O0: conda binaryen 117 rejects
# emcc 4.0.9 wasm-opt flags (same as srep/pref).
EMCC ?= emcc
LZ4 = ../../third_party/lz4
ZSTD = ../../third_party/zstd
GUEST = xt3udec.wasm
ZSTD_SRC = \
	$(ZSTD)/common/debug.c \
	$(ZSTD)/common/entropy_common.c \
	$(ZSTD)/common/error_private.c \
	$(ZSTD)/common/fse_decompress.c \
	$(ZSTD)/common/xxhash.c \
	$(ZSTD)/common/zstd_common.c \
	$(ZSTD)/decompress/huf_decompress.c \
	$(ZSTD)/decompress/zstd_ddict.c \
	$(ZSTD)/decompress/zstd_decompress.c \
	$(ZSTD)/decompress/zstd_decompress_block.c
SRC = guest/main.c $(LZ4)/lz4.c $(LZ4)/lz4hc.c $(ZSTD_SRC)

.PHONY: guest clean
guest: $(GUEST)

$(GUEST): $(SRC) $(LZ4)/lz4.h $(LZ4)/lz4hc.h $(ZSTD)/zstd.h
	$(EMCC) $(SRC) \
		-O0 \
		-I$(LZ4) -I$(ZSTD) -I$(ZSTD)/common \
		-DZSTD_LEGACY_SUPPORT=0 -DZSTD_DISABLE_ASM=1 \
		--no-entry \
		-sSTANDALONE_WASM=1 \
		-sALLOW_MEMORY_GROWTH=1 \
		-sINITIAL_MEMORY=67108864 \
		-sMAXIMUM_MEMORY=2147483648 \
		-sSTACK_SIZE=2097152 \
		-sEXPORTED_FUNCTIONS=_xt3u_lz4hc,_xt3u_lz4,_xt3u_lz4_bound,_xt3u_zstd_prefix,_malloc,_free \
		-Wno-version-check \
		-o $(GUEST)

clean:
	rm -f $(GUEST)
