#include "lz4.h"
#include "lz4hc.h"
#include "zstd.h"
#include "zstd_errors.h"

#include <stdint.h>

// xt3u_lz4hc is LZ4_compress_HC from lz4 1.9.4 (PrecompLZ4.pas LZ4Restore).
int32_t xt3u_lz4hc(const char *src, int32_t src_size, char *dst, int32_t dst_cap, int32_t level) {
	if (src == 0 || dst == 0 || src_size < 0 || dst_cap < 0) {
		return 0;
	}
	return LZ4_compress_HC(src, dst, src_size, dst_cap, level);
}

int32_t xt3u_lz4(const char *src, int32_t src_size, char *dst, int32_t dst_cap, int32_t accel) {
	if (src == 0 || dst == 0 || src_size < 0 || dst_cap < 0) {
		return 0;
	}
	if (accel <= 0) {
		accel = 1;
	}
	return LZ4_compress_fast(src, dst, src_size, dst_cap, accel);
}

int32_t xt3u_lz4_bound(int32_t src_size) {
	if (src_size < 0) {
		return 0;
	}
	return LZ4_compressBound(src_size);
}

// xt3u_zstd_prefix is ZSTD_DCtx_refPrefix + decompressStream (PrecompDecodePatchEx).
// Negative return is -(ZSTD_ErrorCode+1) so 0 stays "empty/fail".
int32_t xt3u_zstd_prefix(const void *patch, int32_t patch_size, const void *prefix, int32_t prefix_size,
	void *dst, int32_t dst_cap, int32_t window_log) {
	ZSTD_DCtx *ctx;
	ZSTD_inBuffer in;
	ZSTD_outBuffer out;
	size_t res;

	if (patch == 0 || dst == 0 || patch_size < 0 || prefix_size < 0 || dst_cap < 0) {
		return 0;
	}
	ctx = ZSTD_createDCtx();
	if (ctx == 0) {
		return 0;
	}
	if (window_log > 0) {
		res = ZSTD_DCtx_setParameter(ctx, ZSTD_d_windowLogMax, window_log);
		if (ZSTD_isError(res)) {
			ZSTD_freeDCtx(ctx);
			return -(int32_t)ZSTD_getErrorCode(res) - 1;
		}
	}
	if (prefix != 0 && prefix_size > 0) {
		res = ZSTD_DCtx_refPrefix(ctx, prefix, (size_t)prefix_size);
		if (ZSTD_isError(res)) {
			ZSTD_freeDCtx(ctx);
			return -(int32_t)ZSTD_getErrorCode(res) - 1;
		}
	}
	in.src = patch;
	in.size = (size_t)patch_size;
	in.pos = 0;
	out.dst = dst;
	out.size = (size_t)dst_cap;
	out.pos = 0;
	while (in.pos < in.size) {
		res = ZSTD_decompressStream(ctx, &out, &in);
		if (ZSTD_isError(res)) {
			ZSTD_freeDCtx(ctx);
			return -(int32_t)ZSTD_getErrorCode(res) - 1;
		}
		if (res == 0) {
			break;
		}
		if (out.pos == out.size && in.pos < in.size) {
			ZSTD_freeDCtx(ctx);
			return 0;
		}
	}
	ZSTD_freeDCtx(ctx);
	return (int32_t)out.pos;
}
