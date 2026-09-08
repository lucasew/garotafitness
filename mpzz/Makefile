# Hybrid mpzz guest: RetDec CLS-OGGRE.c transcription (INV-03).
# -O0: conda binaryen 117 rejects emcc 4.0.9 wasm-opt flags (same as srep).
EMCC ?= emcc
GUEST = mpzzdec.wasm
SRC = guest/main.cpp guest/retdec.c guest/host.c

.PHONY: guest
guest: $(GUEST)

$(GUEST): $(SRC) guest/host.h guest/image.inc
	$(EMCC) guest/retdec.c guest/host.c \
		-O0 -Iguest -c \
		-Wno-version-check -Wno-int-conversion -Wno-incompatible-pointer-types -w
	$(EMCC) guest/main.cpp \
		-O0 -std=c++17 -fno-exceptions -fno-rtti -Iguest -c \
		-Wno-version-check -w
	$(EMCC) retdec.o host.o main.o \
		-O0 \
		--no-entry \
		-sSTANDALONE_WASM=1 \
		-sALLOW_MEMORY_GROWTH=1 \
		-sINITIAL_MEMORY=67108864 \
		-sMAXIMUM_MEMORY=2147483648 \
		-sSTACK_SIZE=2097152 \
		-sEXPORTED_FUNCTIONS=_mpzz_decode,_malloc,_free \
		-Wno-version-check \
		-o $(GUEST)
	rm -f retdec.o host.o main.o
