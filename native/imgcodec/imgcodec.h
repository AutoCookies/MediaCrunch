#ifndef MEDIACRUNCH_IMGCODEC_H
#define MEDIACRUNCH_IMGCODEC_H

#ifdef __cplusplus
extern "C" {
#endif

int mc_transcode_webp(
    const char* in_path,
    const char* out_path,
    int quality,
    char* errbuf,
    int errbuf_len
);

#ifdef __cplusplus
}
#endif

#endif
