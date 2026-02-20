#include "avifcodec.h"

#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <string>
#include <vector>

#include <jpeglib.h>
#include <png.h>
#include <libheif/heif.h>

namespace {
using PixelBuffer = std::vector<uint8_t>;

void write_error(char* errbuf, int errbuf_len, const std::string& msg) {
  if (!errbuf || errbuf_len <= 0) return;
  int n = std::min(errbuf_len - 1, static_cast<int>(msg.size()));
  std::memcpy(errbuf, msg.c_str(), n);
  errbuf[n] = '\0';
}
std::string lowercase_ext(const std::string& path) {
  auto p = path.find_last_of('.');
  if (p == std::string::npos) return "";
  std::string ext = path.substr(p);
  std::transform(ext.begin(), ext.end(), ext.begin(), [](unsigned char c){ return std::tolower(c); });
  return ext;
}

bool decode_jpeg(const std::string& path, PixelBuffer& rgba, int& width, int& height, std::string& err) {
  FILE* fp = std::fopen(path.c_str(), "rb");
  if (!fp) { err = "failed to open JPEG input"; return false; }
  jpeg_decompress_struct cinfo{}; jpeg_error_mgr jerr{}; cinfo.err = jpeg_std_error(&jerr);
  jpeg_create_decompress(&cinfo); jpeg_stdio_src(&cinfo, fp);
  if (jpeg_read_header(&cinfo, TRUE) != JPEG_HEADER_OK) { jpeg_destroy_decompress(&cinfo); std::fclose(fp); err = "invalid JPEG header"; return false; }
  jpeg_start_decompress(&cinfo);
  width = static_cast<int>(cinfo.output_width); height = static_cast<int>(cinfo.output_height);
  int channels = static_cast<int>(cinfo.output_components);
  if (width<=0||height<=0||(channels!=1&&channels!=3)) { jpeg_finish_decompress(&cinfo); jpeg_destroy_decompress(&cinfo); std::fclose(fp); err = "unsupported JPEG format"; return false; }
  std::vector<uint8_t> row(static_cast<size_t>(width)*channels); rgba.resize(static_cast<size_t>(width)*height*4);
  while (cinfo.output_scanline < cinfo.output_height) {
    JSAMPROW row_ptr = row.data(); jpeg_read_scanlines(&cinfo, &row_ptr, 1);
    size_t y = static_cast<size_t>(cinfo.output_scanline - 1); uint8_t* out = rgba.data() + y * static_cast<size_t>(width) * 4;
    for (int x=0; x<width; ++x) {
      if (channels==3){ out[x*4+0]=row[x*3+0]; out[x*4+1]=row[x*3+1]; out[x*4+2]=row[x*3+2]; }
      else { out[x*4+0]=row[x]; out[x*4+1]=row[x]; out[x*4+2]=row[x]; }
      out[x*4+3]=255;
    }
  }
  jpeg_finish_decompress(&cinfo); jpeg_destroy_decompress(&cinfo); std::fclose(fp); return true;
}

bool decode_png(const std::string& path, PixelBuffer& rgba, int& width, int& height, std::string& err) {
  FILE* fp = std::fopen(path.c_str(), "rb");
  if (!fp) { err = "failed to open PNG input"; return false; }
  png_structp png = png_create_read_struct(PNG_LIBPNG_VER_STRING, nullptr, nullptr, nullptr);
  if (!png) { std::fclose(fp); err = "png_create_read_struct failed"; return false; }
  png_infop info = png_create_info_struct(png);
  if (!info) { png_destroy_read_struct(&png, nullptr, nullptr); std::fclose(fp); err = "png_create_info_struct failed"; return false; }
  if (setjmp(png_jmpbuf(png))) { png_destroy_read_struct(&png, &info, nullptr); std::fclose(fp); err = "PNG decode failed"; return false; }
  png_init_io(png, fp); png_read_info(png, info);
  width = static_cast<int>(png_get_image_width(png, info)); height = static_cast<int>(png_get_image_height(png, info));
  png_byte color_type = png_get_color_type(png, info); png_byte bit_depth = png_get_bit_depth(png, info);
  if (bit_depth == 16) png_set_strip_16(png);
  if (color_type == PNG_COLOR_TYPE_PALETTE) png_set_palette_to_rgb(png);
  if (color_type == PNG_COLOR_TYPE_GRAY && bit_depth < 8) png_set_expand_gray_1_2_4_to_8(png);
  if (png_get_valid(png, info, PNG_INFO_tRNS)) png_set_tRNS_to_alpha(png);
  if (color_type == PNG_COLOR_TYPE_GRAY || color_type == PNG_COLOR_TYPE_GRAY_ALPHA) png_set_gray_to_rgb(png);
  if (!(color_type & PNG_COLOR_MASK_ALPHA)) png_set_add_alpha(png, 0xFF, PNG_FILLER_AFTER);
  png_read_update_info(png, info);
  png_size_t rowbytes = png_get_rowbytes(png, info); rgba.resize(rowbytes * static_cast<size_t>(height)); std::vector<png_bytep> rows(static_cast<size_t>(height));
  for (int y = 0; y < height; ++y) rows[static_cast<size_t>(y)] = rgba.data() + static_cast<size_t>(y) * rowbytes;
  png_read_image(png, rows.data()); png_destroy_read_struct(&png, &info, nullptr); std::fclose(fp); return true;
}

bool encode_avif_rgba(const uint8_t* rgba, int width, int height, int quality, const std::string& out_path, std::string& err) {
  struct heif_context* ctx = heif_context_alloc();
  if (!ctx) { err = "heif_context_alloc failed"; return false; }
  struct heif_image* img = nullptr;
  heif_error h = heif_image_create(width, height, heif_colorspace_RGB, heif_chroma_444, &img);
  if (h.code != heif_error_Ok) { heif_context_free(ctx); err = "heif_image_create failed"; return false; }
  h = heif_image_add_plane(img, heif_channel_R, width, height, 8);
  if (h.code == heif_error_Ok) h = heif_image_add_plane(img, heif_channel_G, width, height, 8);
  if (h.code == heif_error_Ok) h = heif_image_add_plane(img, heif_channel_B, width, height, 8);
  if (h.code != heif_error_Ok) { heif_image_release(img); heif_context_free(ctx); err = "heif_image_add_plane failed"; return false; }
  int rs=0, gs=0, bs=0;
  uint8_t* rp = heif_image_get_plane(img, heif_channel_R, &rs);
  uint8_t* gp = heif_image_get_plane(img, heif_channel_G, &gs);
  uint8_t* bp = heif_image_get_plane(img, heif_channel_B, &bs);
  for (int y=0; y<height; ++y) {
    for (int x=0; x<width; ++x) {
      const uint8_t* px = rgba + (y*width + x)*4;
      rp[y*rs + x] = px[0];
      gp[y*gs + x] = px[1];
      bp[y*bs + x] = px[2];
    }
  }

  struct heif_encoder* enc = nullptr;
  h = heif_context_get_encoder_for_format(ctx, heif_compression_AV1, &enc);
  if (h.code != heif_error_Ok || !enc) { heif_image_release(img); heif_context_free(ctx); err = "no AV1 encoder available in libheif"; return false; }
  heif_encoder_set_lossy_quality(enc, quality);
  h = heif_context_encode_image(ctx, img, enc, nullptr, nullptr);
  if (h.code != heif_error_Ok) { heif_encoder_release(enc); heif_image_release(img); heif_context_free(ctx); err = "heif_context_encode_image failed"; return false; }
  h = heif_context_write_to_file(ctx, out_path.c_str());
  heif_encoder_release(enc); heif_image_release(img); heif_context_free(ctx);
  if (h.code != heif_error_Ok) { err = "heif_context_write_to_file failed"; return false; }
  return true;
}
}

extern "C" int mc_transcode_avif(const char* in_path, const char* out_path, int quality, char* errbuf, int errbuf_len) {
  if (!in_path || !out_path) { write_error(errbuf, errbuf_len, "input and output paths must not be null"); return 1; }
  if (quality < 1 || quality > 100) { write_error(errbuf, errbuf_len, "quality must be between 1 and 100"); return 2; }
  std::string in(in_path), out(out_path), decerr;
  PixelBuffer rgba; int width = 0, height = 0;
  std::string ext = lowercase_ext(in);
  bool ok = false;
  if (ext == ".jpg" || ext == ".jpeg") ok = decode_jpeg(in, rgba, width, height, decerr);
  else if (ext == ".png") ok = decode_png(in, rgba, width, height, decerr);
  else { write_error(errbuf, errbuf_len, "unsupported input extension"); return 3; }
  if (!ok) { write_error(errbuf, errbuf_len, decerr); return 4; }
  std::string encerr;
  if (!encode_avif_rgba(rgba.data(), width, height, quality, out, encerr)) { write_error(errbuf, errbuf_len, encerr); return 5; }
  return 0;
}
