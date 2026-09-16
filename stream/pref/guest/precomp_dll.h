// Guest-facing copy of official precomp_dll.h. Restore does not start
// worker threads; Go owns scratch files and any host concurrency.
class Switches {
public:
	Switches();

	int compression_method;
	unsigned int compression_otf_max_memory;
	unsigned int compression_otf_thread_count;
	long long *ignore_list;
	int ignore_list_len;
	bool intense_mode;
	bool fast_mode;
	bool brute_mode;
	bool pdf_bmp_mode;
	bool prog_only;
	bool use_mjpeg;
	bool use_brunsli;
	bool use_brotli;
	bool use_packjpg_fallback;
	bool debug_mode;
	unsigned int min_ident_size;
	bool use_pdf;
	bool use_zip;
	bool use_gzip;
	bool use_png;
	bool use_gif;
	bool use_jpg;
	bool use_mp3;
	bool use_swf;
	bool use_base64;
	bool use_bzip2;
	bool level_switch;
	bool use_zlib_level[81];
};

inline Switches::Switches() {
	compression_method = 0;
	compression_otf_max_memory = 2048;
	compression_otf_thread_count = 1;
	ignore_list = 0;
	ignore_list_len = 0;
	intense_mode = false;
	fast_mode = false;
	brute_mode = false;
	pdf_bmp_mode = false;
	prog_only = false;
	use_mjpeg = true;
	use_brunsli = true;
	use_brotli = false;
	use_packjpg_fallback = true;
	debug_mode = false;
	min_ident_size = 4;
	use_pdf = true;
	use_zip = true;
	use_gzip = true;
	use_png = true;
	use_gif = true;
	use_jpg = true;
	use_mp3 = true;
	use_swf = true;
	use_base64 = true;
	use_bzip2 = true;
	level_switch = false;
	for (int i = 0; i < 81; i++) {
		use_zlib_level[i] = true;
	}
}

#ifndef DLL
#define DLL
#endif

DLL void get_copyright_msg(char *msg);
DLL bool precompress_file(char *in_file, char *out_file, char *msg, Switches switches);
DLL bool recompress_file(char *in_file, char *out_file, char *msg, Switches switches);
