#include <cstdint>
#include <string>
#include <vector>

#include <webp/encode.h>

namespace mc {

bool encode_webp_rgba(
    const uint8_t* rgba,
    int width,
    int height,
    int stride,
    int quality,
    std::vector<uint8_t>& encoded,
    std::string& err
) {
    uint8_t* out_data = nullptr;
    const size_t out_size = WebPEncodeRGBA(rgba, width, height, stride, static_cast<float>(quality), &out_data);
    if (out_size == 0 || out_data == nullptr) {
        err = "WebPEncodeRGBA failed";
        return false;
    }

    encoded.assign(out_data, out_data + out_size);
    WebPFree(out_data);
    return true;
}

}  // namespace mc
