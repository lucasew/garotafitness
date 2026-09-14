LZMA SDK 19.00 by Igor Pavlov (public domain, see `NOTICE`). These files come
from the [Android mirror](https://android.googlesource.com/platform/external/lzma/+/0d3c9641352f8b2603beeac80181d360aff831d0),
commit `0d3c9641352f8b2603beeac80181d360aff831d0`, which imports SDK 19.00.

The C files are unmodified. `reconstruct/fgpack/guest/main.c` supplies the parameters used
by the installer's LZMA console invocation and writes its 13-byte header.
It builds with `_7ZIP_ST` because Guests run without threads.
