# go-face

A Go face-recognition library powered by [InspireFace](https://github.com/HyperInspire/InspireFace).  
All core logic (detection, feature extraction, alignment, backend probing) lives in a thin C layer; CGO is used **only** for I/O exchange.

## Features

- **Multi-format raw image input** — RGB, BGR, RGBA, BGRA, NV12, NV21, I420, GRAY
- **One-shot detection** — bounding box + pose angles + 512-dim feature vector + aligned face image
- **RK3588 hardware acceleration** — explicit `BackendRGA` enables Rockchip RGA image preprocessing, with automatic CPU fallback
- **Auto model selection** — `InitAuto` picks the best available model (NPU → high-perf CPU → lightweight CPU)
- **Dual API**
  - *High-level* — idiomatic Go (`Session.Detect(...)`)
  - *Low-level* — direct C-pointer access for zero-copy scenarios
- **Pre-bundled native libraries & models** — macOS (Apple Silicon), Linux x86, Linux aarch64, Linux RK3588
- **HTTP server example** — InsightFace-style REST API out of the box

## Table of Contents

- [Platform Support](#platform-support)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [API Reference](#api-reference)
- [Image Formats](#image-formats)
- [Performance Tips](#performance-tips)
- [Troubleshooting](#troubleshooting)
- [Architecture](#architecture)
- [License](#license)

## Platform Support

| Platform | Backend | Build tag | Notes |
|----------|---------|-----------|-------|
| macOS (arm64) | MNN/CPU | *(none)* | Development / compile-check only |
| Linux x86 (amd64) | MNN/CPU | *(none)* | Use `Pikachu` or `Megatron` pack |
| Linux aarch64 (generic) | MNN/CPU | *(none)* | Use `Pikachu` or `Megatron` pack |
| Linux RK3588 | RKNN + RGA | `rk3588` | Use `Gundam_RK3588` pack |

> On RK3588 you can enable the Rockchip RGA hardware accelerator for image preprocessing by setting `Backend: goface.BackendRGA`. If RGA is not available the library transparently falls back to CPU.
>
> **RK3588 Prerequisites:** Install the system NPU driver package before running:
> ```bash
> sudo apt update
> sudo apt install rknpu2-rk3588
> ```
> The user-space library `librknnrt.so` is already bundled under `third_party/inspireface/lib/linux_aarch64_rk3588/`, but the kernel-space NPU driver (`/dev/rknpu`) must still be installed on the target device.
>
> ⚠️ The bundled `Pikachu` model pack is primarily validated on **Linux aarch64**. macOS users may encounter `HERR_ARCHIVE_LOAD_MODEL_FAILURE` due to MNN runtime version drift; this is harmless for development — deploy and test on your target Linux / RK3588 device.

## Quick Start

### HTTP Server (InsightFace-style API)

The fastest way to try the library is the bundled HTTP server example:

```bash
cd examples/server
go run main.go
```

Upload a photo and get face info:
```bash
curl -X POST http://localhost:8080/detect \
  -F "image=@your_photo.jpg" \
  -F "feature=true"
```

See [`examples/server/README.md`](examples/server/README.md) for the full API reference (`/detect`, `/compare`, `/health`).

### 1. Model packs (already bundled)

This repository already includes four model packs under `models/`:

| Pack | Device | Backend | Feature dim | Size | Download |
|------|--------|---------|-------------|------|----------|
| `Pikachu` | CPU (x86/arm64) | MNN | **512-d** | ~16 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Pikachu) |
| `Megatron` | CPU (x86/arm64) | MNN | **512-d** | ~58 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron) |
| `Megatron_TRT` | NVIDIA GPU | TensorRT | **512-d** | ~67 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron_TRT) |
| `Gundam_RK3588` | RK3588 NPU | RKNN + RGA | **512-d** | ~35 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Gundam_RK3588) |

All packs output **512-dimensional** face embeddings (ArcFace), so they are fully compatible for cross-platform feature comparison.

> If you need other model packs (RK356X, RV1106, etc.), download them from the [InspireFace releases](https://github.com/HyperInspire/InspireFace/releases) page or use the official `command/download_models_general.sh` script.
>
> If the model packs are not present locally, `InitAuto` will automatically download the best matching model from the [GitHub releases](https://github.com/joy999/go-face/releases) page.

### 2. Use in your Go project

```bash
go get github.com/joy999/go-face
```

#### Automatic model selection (recommended)

```go
modelPath, err := goface.InitAuto("./models")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Loaded model:", modelPath)
defer goface.Deinit()
```

`InitAuto` tries models in this priority:
1. **RK3588**: `Gundam_RK3588` → `Megatron` → `Pikachu`
2. **Other platforms**: `Megatron` → `Pikachu`

If the model directory is empty or missing, `InitAuto` will automatically download the best matching model from GitHub releases before initializing.

You can also download a specific model manually:

```go
if err := goface.DownloadModel("./models", "Pikachu"); err != nil {
    log.Fatal(err)
}
```

#### Manual initialization

```go
package main

import (
    "fmt"
    "log"
    goface "github.com/joy999/go-face"
)

func main() {
    // 1. Global init (once per process)
    if err := goface.Init(goface.InitOption{
        ModelPath: "./models/Pikachu",
        Backend:   goface.BackendAuto,
    }); err != nil {
        log.Fatal(err)
    }
    defer goface.Deinit()

    // 2. Create a session
    sess, err := goface.NewSession(goface.DefaultSessionOption())
    if err != nil {
        log.Fatal(err)
    }
    defer sess.Close()

    // 3. Raw image bytes from camera / gstreamer / decoder
    width, height := 640, 480
    img := make([]byte, width*height*3) // BGR in this example

    // 4. Detect
    faces, err := sess.Detect(img, width, height, goface.FormatBGR)
    if err != nil {
        log.Fatal(err)
    }

    for i, f := range faces {
        fmt.Printf("Face #%d rect=%v conf=%.3f\n", i, f.Rect, f.Confidence)
        if f.Feature != nil {
            fmt.Printf("  feature dim=%d\n", len(f.Feature))
        }
        if f.FaceImage != nil {
            fmt.Printf("  aligned face=%dx%dx%d\n",
                f.FaceImage.Width, f.FaceImage.Height, f.FaceImage.Channels)
        }
    }

    // 5. Compare two faces
    if len(faces) >= 2 {
        sim := goface.CompareFeatures(faces[0].Feature, faces[1].Feature)
        fmt.Printf("similarity=%.3f (threshold=%.3f)\n",
            sim, goface.RecommendedThreshold())
    }
}
```

### 3. Build

```bash
# macOS / generic Linux x86 / generic Linux aarch64
go build

# RK3588 ( enables RKNN + RGA libraries )
go build -tags rk3588
```

### 4. Run-time library path

The bundled `.dylib`/`.so` files are referenced with `@rpath` / `-Wl,-rpath` so no extra `LD_LIBRARY_PATH` is required in most cases. If you prefer system-wide installation, copy the libraries to `/usr/local/lib` (or `/usr/lib` on Linux) and run `ldconfig`.

## Project Structure

```
go-face/
├── goface.c, goface.h          # C core layer (all InspireFace logic)
├── face.go                      # CGO bindings + Go high/low-level API
├── types.go                     # Go type definitions
├── auto.go                      # Auto model selection API
├── auto_default.go              # Auto selection for non-RK3588 platforms
├── auto_rk3588.go               # Auto selection for RK3588 (build tag: rk3588)
├── face_test.go                 # Unit tests
├── face_e2e_test.go             # End-to-end pipeline tests
├── bench_feature_test.go        # Feature extraction benchmarks (CPU & RK3588)
├── cgo_darwin.go                # macOS linker flags
├── cgo_linux.go                 # Linux x86 linker flags
├── cgo_linux_arm64.go           # Linux aarch64 linker flags
├── cgo_linux_rk3588.go          # RK3588 linker flags (build tag: rk3588)
├── go.mod                       # Go 1.21
├── README.md / README_CN.md
├── models/
│   ├── Pikachu/                 # Lightweight CPU model pack (512-dim, MNN)
│   ├── Megatron/                # High-performance CPU model pack (512-dim, MNN)
│   ├── Megatron_TRT/            # TensorRT GPU model pack (512-dim)
│   └── Gundam_RK3588/           # RK3588 model pack (512-dim, RKNN+RGA)
├── third_party/
│   └── inspireface/
│       ├── include/             # InspireFace C headers
│       └── lib/                 # Pre-compiled libraries per platform
│           ├── darwin_arm64/
│           ├── linux_x86/
│           └── linux_aarch64_rk3588/
├── examples/
│   ├── simple/main.go           # High-level API usage
│   ├── lowlevel/main.go         # Low-level API usage (zero-copy)
│   ├── autotest/main.go         # InitAuto stress test
│   └── server/                  # HTTP REST server (InsightFace-style)
│       ├── main.go
│       └── README.md
```

## API Reference

### Global Lifecycle

```go
// Initialize the SDK (once per process)
err := goface.Init(goface.InitOption{
    ModelPath: "./models/Pikachu",  // path to model pack (tar archive or directory with corresponding .tar)
    Backend:   goface.BackendAuto,  // BackendAuto / BackendCPU / BackendRGA
    LogLevel:  2,                   // 0=error, 1=warn, 2=info, 3=debug
})
defer goface.Deinit()
```

### Auto Model Selection

```go
// Automatically pick the best available model for the current platform.
// On RK3588: Gundam_RK3588 → Megatron → Pikachu
// On others:  Megatron → Pikachu
modelPath, err := goface.InitAuto("./models")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Loaded model:", modelPath)
defer goface.Deinit()
```

### Session Options

```go
opt := goface.DefaultSessionOption()
// or customize:
opt := goface.SessionOption{
    MaxFaces:          10,   // max faces to detect
    DetectPixelLevel:  320,  // detector input resolution: 160, 320, 640; -1 = default
    EnableRecognition: 1,    // extract 512-dim feature vector
    EnableFacePose:    1,    // head pose (roll/yaw/pitch)
    EnableQuality:     0,    // face quality assessment
    EnableLiveness:    0,    // RGB anti-spoofing
    EnableMaskDetect:  0,    // mask detection
}
sess, err := goface.NewSession(opt)
defer sess.Close()
```

### Detection

```go
// Detect with default rotation (Rotation0)
faces, err := sess.Detect(data, width, height, goface.FormatBGR)

// Detect with camera rotation
faces, err := sess.DetectWithRotation(data, width, height, goface.FormatNV12, goface.Rotation90)

// Face struct fields:
//   Rect       image.Rectangle
//   Roll/Yaw/Pitch  float32
//   Confidence float32
//   Feature    []float32   // 512-dim, nil if recognition disabled
//   FaceImage  *FaceImage  // aligned cropped face, nil if unavailable
```

### Feature Comparison

```go
sim := goface.CompareFeatures(featA, featB)   // cosine similarity [-1, 1]
thr := goface.RecommendedThreshold()            // ~0.48 (model-dependent)
if sim > thr {
    // same person
}
```

### Error Codes

```go
msg := goface.StrError(code)  // convert InspireFace error code to human-readable string
```

### Low-Level API

For advanced users who want to avoid intermediate allocations or manage their own session pool:

```go
ptr, _ := goface.LowLevelSessionCreate(goface.DefaultSessionOption())
defer goface.LowLevelSessionDestroy(ptr)

result, count, err := goface.LowLevelDetect(ptr, img, w, h, goface.FormatBGR, goface.Rotation0)
if err != nil { ... }
defer goface.LowLevelResultFree(result)

for i := 0; i < count; i++ {
    var f goface.Face
    goface.LowLevelResultGetFace(result, i, &f)
    // use f ...
}
```

## Image Formats

| Go constant | Description | Typical source |
|-------------|-------------|----------------|
| `FormatRGB` | interleaved RGB | OpenCV RGB |
| `FormatBGR` | interleaved BGR | OpenCV default |
| `FormatRGBA`| RGB + Alpha | PNG with alpha |
| `FormatBGRA`| BGR + Alpha | Graphics APIs |
| `FormatNV12`| YUV NV12 | GStreamer `nv12`, iOS camera |
| `FormatNV21`| YUV NV21 | Android camera |
| `FormatI420`| YUV I420 | Video codecs |
| `FormatGRAY`| single channel | IR / depth cameras |

## Vendoring & Native Libraries

`go mod vendor` does **not** copy directories without `.go` files, and it also skips files larger than 1 MB. This means:

- **C headers** under `third_party/inspireface/include/` are omitted by default. We fixed this by turning the header directories into placeholder Go packages so they are vendored correctly (v1.0.3+).
- **`.so` / `.dylib` files** under `third_party/inspireface/lib/` are **never** vendored because they are several megabytes each.

If you use `go mod vendor`, run the helper after vendoring to download the native libraries:

```bash
# If you have vendor/ enabled, use -mod=mod so the tool can be fetched
# (or run from a local clone of go-face)
go run -mod=mod github.com/joy999/go-face/cmd/install-libs

# If you already have a local clone:
go run /path/to/go-face/cmd/install-libs
```

The tool automatically detects whether `go-face` lives in your `vendor/` directory or in the module cache, and extracts `lib.tar.gz` to the correct `third_party/inspireface/lib/` location.

If you prefer a system-wide installation (so the libraries are available for all builds), use:

```bash
go run github.com/joy999/go-face/cmd/install-libs -system
sudo ldconfig   # on Linux
```

> **Note:** If you are behind a firewall or GitHub is unreachable, download [`lib.tar.gz`](https://github.com/joy999/go-face/releases/download/init/lib.tar.gz) manually and extract it to `third_party/inspireface/lib/` (or `/usr/local/lib` for `-system`).

## Performance Tips

- **Reuse sessions**. `Session` allocates internal caches and GPU/NPU memory. Create one per goroutine or use a pool; do not create/destroy per request.
- **Match pixel level to workload**. `DetectPixelLevel: 160` is fastest; `640` is most accurate. `320` is the sweet spot for most scenes.
- **Enable only what you need**. Each option (`EnableLiveness`, `EnableMaskDetect`, etc.) increases memory and latency. Disable unused features.
- **Use `BackendRGA` on RK3588** to enable Rockchip RGA hardware acceleration. If RGA is unavailable the library falls back to CPU automatically. `BackendAuto` currently defaults to CPU processing.
- **Use `InitAuto`**. It automatically selects the highest-performance model available on the current hardware.
- **NV12/NV21 zero-copy**. If your pipeline already outputs NV12 (e.g. GStreamer `nv12`), pass it directly instead of converting to BGR/RGB.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `HERR_ARCHIVE_LOAD_MODEL_FAILURE` (252) | Model pack version mismatch or missing `.mnn`/`.rknn` extension | Use the model pack matching your platform; bundled packs are already fixed |
| `Model path does not exist` | `ModelPath` points to a non-existent path | Ensure the path matches one of the bundled model packs (e.g. `./models/Pikachu`) |
| `no session available` (HTTP server) | All sessions in pool are busy | Increase `POOL_SIZE` or reduce concurrent requests |
| `HERR_INVALID_IMAGE_STREAM_PARAM` | Width/height/format mismatch | Verify `len(data)` matches `width*height*channels` for the chosen format |
| `inspireface.h: No such file or directory` (vendor builds) | `go mod vendor` omits header directories | Upgrade to v1.0.3+, re-run `go mod vendor`; headers are now copied via placeholder packages |
| `cannot find -lInspireFace` (vendor builds) | `.so` files > 1 MB are skipped by `go mod vendor` | Run `go run github.com/joy999/go-face/cmd/install-libs` after vendoring, or install libs to `/usr/local/lib` |
| macOS: `image not found` at runtime | `@rpath` resolution failure | `export DYLD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/darwin_arm64` |
| Linux x86: `cannot open shared object file` | `rpath` not honored by loader | `export LD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/linux_x86` |
| Linux aarch64: `cannot open shared object file` | `rpath` not honored by loader | `export LD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/linux_aarch64_rk3588` |

## Architecture

```
┌─────────────────────────────────────────────┐
│  Go Application                              │
│  (Session.Detect / LowLevelDetect)           │
├─────────────────────────────────────────────┤
│  Go I/O layer (types, slice wrapping)        │
├─────────────────────────────────────────────┤
│  CGO — pure I/O bridge                      │
│  (C.go → Go, Go → C struct)                 │
├─────────────────────────────────────────────┤
│  goface.c / goface.h                         │
│  • init / deinit                             │
│  • session create / destroy                  │
│  • detect (stream → track → extract → crop) │
│  • feature compare                           │
├─────────────────────────────────────────────┤
│  InspireFace C API                           │
│  (libInspireFace.so / .dylib)               │
└─────────────────────────────────────────────┘
```

All control flow (loops, conditionals, resource cleanup) stays inside `goface.c`. Go never calls individual InspireFace APIs directly.

## License

The **code** in this repository is released under the MIT License.

> ⚠️ The **models** bundled in InspireFace packs follow the same license as InsightFace: academic use only, commercial use prohibited without permission. See [InspireFace License](https://github.com/HyperInspire/InspireFace#license) for details.
