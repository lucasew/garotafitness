Sources for the FSB5 remux Guest:

- [oggvorbis2fsb5](https://github.com/uyjulian/oggvorbis2fsb5), commit `03bd46a46981f20a345df559dfc862f0666469ef` (MIT, `LICENSE`).
- [libogg](https://github.com/xiph/ogg), pinned submodule commit `bada45718453ac27b56773ae663f7e65112f6a6e` (BSD, `ogg/COPYING`).
- [libvorbis](https://github.com/xiph/vorbis), pinned submodule commit `0657aee69dec8508a0011f47f3b69d7538e9d262` (BSD, `vorbis/COPYING`).

Only the C sources and headers required by the remuxer are included. The remuxer
uses stdin/stdout in place of the two filename arguments, rejects empty input,
and checks stream I/O errors. Its packet and FSB5 serialization are unchanged.
Build with `make -C reconstruct/fsb` using the project's emscripten toolchain.
