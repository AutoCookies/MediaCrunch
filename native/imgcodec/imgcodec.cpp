#include "imgcodec.h"

#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <memory>
#include <string>
#include <vector>

#include <jpeglib.h>
#include <png.h>

namespace {

using PixelBuffer = std::vector<uint8_t>;

void write_error(char* errbuf, int errbuf_len, const std::string& msg) {
    if (errbuf == nullptr || errbuf_len <= 0) {
        return;
    }

    const int max_copy = errbuf_len - 1;
    const int copy_len = std::min<int>(max_copy, static_cast<int>(msg.size()));
    std::memcpy(errbuf, msg.c_str(), copy_len);
    errbuf[copy_len] = '\0';
}

std::string lowercase_ext(const std::string& path) {
    const std::string::size_type dot_pos = path.find_last_of('.');
    if (dot_pos == std::string::npos) {
        return "";
    }
    std::string ext = path.substr(dot_pos);
    std::transform(ext.begin(), ext.end(), ext.begin(), [](unsigned char c) { return std::tolower(c); });
    return ext;
}

bool decode_jpeg(const std::string& path, PixelBuffer& rgba, int& width, int& height, std::string& err) {
    FILE* fp = std::fopen(path.c_str(), "rb");
    if (!fp) {
        err = "failed to open JPEG input";
        return false;
    }

    jpeg_decompress_struct cinfo{};
    jpeg_error_mgr jerr{};
    cinfo.err = jpeg_std_error(&jerr);

    jpeg_create_decompress(&cinfo);
    jpeg_stdio_src(&cinfo, fp);

    if (jpeg_read_header(&cinfo, TRUE) != JPEG_HEADER_OK) {
        jpeg_destroy_decompress(&cinfo);
        std::fclose(fp);
        err = "invalid JPEG header";
        return false;
    }

    jpeg_start_decompress(&cinfo);
    width = static_cast<int>(cinfo.output_width);
    height = static_cast<int>(cinfo.output_height);
    const int channels = static_cast<int>(cinfo.output_components);

    if (width <= 0 || height <= 0 || (channels != 1 && channels != 3)) {
        jpeg_finish_decompress(&cinfo);
        jpeg_destroy_decompress(&cinfo);
        std::fclose(fp);
        err = "unsupported JPEG format";
        return false;
    }

    const size_t row_stride = static_cast<size_t>(width) * static_cast<size_t>(channels);
    std::vector<uint8_t> row(row_stride);
    rgba.resize(static_cast<size_t>(width) * static_cast<size_t>(height) * 4);

    while (cinfo.output_scanline < cinfo.output_height) {
        JSAMPROW row_ptr = row.data();
        jpeg_read_scanlines(&cinfo, &row_ptr, 1);
        const size_t y = static_cast<size_t>(cinfo.output_scanline - 1);
        uint8_t* out = rgba.data() + y * static_cast<size_t>(width) * 4;

        for (int x = 0; x < width; ++x) {
            if (channels == 3) {
                out[x * 4 + 0] = row[x * 3 + 0];
                out[x * 4 + 1] = row[x * 3 + 1];
                out[x * 4 + 2] = row[x * 3 + 2];
            } else {
                out[x * 4 + 0] = row[x];
                out[x * 4 + 1] = row[x];
                out[x * 4 + 2] = row[x];
            }
            out[x * 4 + 3] = 255;
        }
    }

    jpeg_finish_decompress(&cinfo);
    jpeg_destroy_decompress(&cinfo);
    std::fclose(fp);
    return true;
}

bool decode_png(const std::string& path, PixelBuffer& rgba, int& width, int& height, std::string& err) {
    FILE* fp = std::fopen(path.c_str(), "rb");
    if (!fp) {
        err = "failed to open PNG input";
        return false;
    }

    png_structp png = png_create_read_struct(PNG_LIBPNG_VER_STRING, nullptr, nullptr, nullptr);
    if (!png) {
        std::fclose(fp);
        err = "png_create_read_struct failed";
        return false;
    }

    png_infop info = png_create_info_struct(png);
    if (!info) {
        png_destroy_read_struct(&png, nullptr, nullptr);
        std::fclose(fp);
        err = "png_create_info_struct failed";
        return false;
    }

    if (setjmp(png_jmpbuf(png))) {
        png_destroy_read_struct(&png, &info, nullptr);
        std::fclose(fp);
        err = "PNG decode failed";
        return false;
    }

    png_init_io(png, fp);
    png_read_info(png, info);

    width = static_cast<int>(png_get_image_width(png, info));
    height = static_cast<int>(png_get_image_height(png, info));
    png_byte color_type = png_get_color_type(png, info);
    png_byte bit_depth = png_get_bit_depth(png, info);

    if (bit_depth == 16) {
        png_set_strip_16(png);
    }
    if (color_type == PNG_COLOR_TYPE_PALETTE) {
        png_set_palette_to_rgb(png);
    }
    if (color_type == PNG_COLOR_TYPE_GRAY && bit_depth < 8) {
        png_set_expand_gray_1_2_4_to_8(png);
    }
    if (png_get_valid(png, info, PNG_INFO_tRNS)) {
        png_set_tRNS_to_alpha(png);
    }
    if (color_type == PNG_COLOR_TYPE_GRAY || color_type == PNG_COLOR_TYPE_GRAY_ALPHA) {
        png_set_gray_to_rgb(png);
    }
    if (!(color_type & PNG_COLOR_MASK_ALPHA)) {
        png_set_add_alpha(png, 0xFF, PNG_FILLER_AFTER);
    }

    png_read_update_info(png, info);

    png_size_t rowbytes = png_get_rowbytes(png, info);
    rgba.resize(rowbytes * static_cast<size_t>(height));
    std::vector<png_bytep> rows(static_cast<size_t>(height));
    for (int y = 0; y < height; ++y) {
        rows[static_cast<size_t>(y)] = rgba.data() + static_cast<size_t>(y) * rowbytes;
    }

    png_read_image(png, rows.data());

    png_destroy_read_struct(&png, &info, nullptr);
    std::fclose(fp);
    return true;
}

}  // namespace

namespace mc {
bool encode_webp_rgba(
    const uint8_t* rgba,
    int width,
    int height,
    int stride,
    int quality,
    std::vector<uint8_t>& encoded,
    std::string& err
);
}  // namespace mc

extern "C" int mc_transcode_webp(
    const char* in_path,
    const char* out_path,
    int quality,
    char* errbuf,
    int errbuf_len
) {
    if (!in_path || !out_path) {
        write_error(errbuf, errbuf_len, "input and output paths must not be null");
        return 1;
    }
    if (quality < 1 || quality > 100) {
        write_error(errbuf, errbuf_len, "quality must be between 1 and 100");
        return 2;
    }

    const std::string in(in_path);
    const std::string out(out_path);

    PixelBuffer rgba;
    int width = 0;
    int height = 0;
    std::string decode_err;

    const std::string ext = lowercase_ext(in);
    bool decoded = false;
    if (ext == ".jpg" || ext == ".jpeg") {
        decoded = decode_jpeg(in, rgba, width, height, decode_err);
    } else if (ext == ".png") {
        decoded = decode_png(in, rgba, width, height, decode_err);
    } else {
        write_error(errbuf, errbuf_len, "unsupported input extension");
        return 3;
    }

    if (!decoded) {
        write_error(errbuf, errbuf_len, decode_err);
        return 4;
    }

    std::vector<uint8_t> encoded;
    std::string encode_err;
    if (!mc::encode_webp_rgba(rgba.data(), width, height, width * 4, quality, encoded, encode_err)) {
        write_error(errbuf, errbuf_len, encode_err);
        return 5;
    }

    std::ofstream ofs(out, std::ios::binary | std::ios::trunc);
    if (!ofs) {
        write_error(errbuf, errbuf_len, "failed to open output file for writing");
        return 6;
    }
    ofs.write(reinterpret_cast<const char*>(encoded.data()), static_cast<std::streamsize>(encoded.size()));
    if (!ofs.good()) {
        write_error(errbuf, errbuf_len, "failed while writing output file");
        return 7;
    }

    return 0;
}
