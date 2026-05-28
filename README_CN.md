# go-face

基于 [InspireFace](https://github.com/HyperInspire/InspireFace) 的 Go 人脸识别库。  
所有核心逻辑（检测、特征提取、对齐、后端探测）均封装在轻量 C 层中；CGO **仅用于 I/O 交换**，Go 不控制流程。

## 功能特性

- **多格式原始图像输入** — 支持 RGB、BGR、RGBA、BGRA、NV12、NV21、I420、GRAY
- **一键检测** — 单次调用返回人脸框 + 姿态角 + 512 维特征向量 + 对齐后人脸图
- **RK3588 硬件加速** — 显式指定 `BackendRGA` 可启用 Rockchip RGA 图像预处理，不可用时自动回退 CPU
- **自动模型选择** — `InitAuto` 按优先级自动选择当前平台最优模型（NPU → 高性能 CPU → 轻量 CPU）
- **双模式 API**
  - *高层 API* — 地道 Go 风格（`Session.Detect(...)`）
  - *底层 API* — 直接操作 C 指针，零拷贝场景
- **预置原生库与模型** — 内置 macOS(Apple Silicon)、Linux x86、Linux aarch64、Linux RK3588 库及模型包
- **HTTP 服务示例** — 开箱即用的 InsightFace 风格 REST API

## 目录

- [平台支持](#平台支持)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [API 参考](#api-参考)
- [图像格式](#图像格式)
- [性能优化](#性能优化)
- [常见问题排查](#常见问题排查)
- [架构设计](#架构设计)
- [许可证](#许可证)

## 平台支持

| 平台 | 推理后端 | 构建标签 | 说明 |
|----------|---------|-----------|-------|
| macOS (arm64) | MNN/CPU | *(无)* | 仅用于开发/编译验证 |
| Linux x86 (amd64) | MNN/CPU | *(无)* | 使用 `Pikachu` 或 `Megatron` 模型包 |
| Linux aarch64 (通用) | MNN/CPU | *(无)* | 使用 `Pikachu` 或 `Megatron` 模型包 |
| Linux RK3588 | RKNN + RGA | `rk3588` | 使用 `Gundam_RK3588` 模型包 |

> RK3588 上可通过设置 `Backend: goface.BackendRGA` 启用 Rockchip RGA 硬件加速进行图像预处理。若 RGA 不可用，库会透明降级为 CPU 处理。
>
> **RK3588 前置条件：** 运行前需安装系统 NPU 驱动包：
> ```bash
> sudo apt update
> sudo apt install rknpu2-rk3588
> ```
> 用户态库 `librknnrt.so` 已打包在 `third_party/inspireface/lib/linux_aarch64_rk3588/` 中，但目标设备仍需安装内核态 NPU 驱动（`/dev/rknpu`）。
>
> ⚠️ 预置的 `Pikachu` 模型包主要验证于 **Linux aarch64**。macOS 用户可能遇到 `HERR_ARCHIVE_LOAD_MODEL_FAILURE`（MNN 运行时版本漂移），这不影响代码逻辑——请在目标 Linux / RK3588 设备上部署测试。

## 快速开始

### HTTP 服务（InsightFace 风格 API）

最快捷的体验方式是运行内置的 HTTP 服务示例：

```bash
cd examples/server
go run main.go
```

上传照片获取人脸信息：
```bash
curl -X POST http://localhost:8080/detect \
  -F "image=@你的照片.jpg" \
  -F "feature=true"
```

完整 API 文档见 [`examples/server/README.md`](examples/server/README.md)（`/detect`、`/compare`、`/health`）。

### 1. 模型包（已内置）

本仓库已在 `models/` 目录下内置四套模型包：

| 模型包 | 适用设备 | 推理后端 | 特征维度 | 大小 | 下载 |
|------|--------|---------|-------------|------|------|
| `Pikachu` | CPU (x86/arm64) | MNN | **512 维** | ~16 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Pikachu) |
| `Megatron` | CPU (x86/arm64) | MNN | **512 维** | ~58 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron) |
| `Megatron_TRT` | NVIDIA GPU | TensorRT | **512 维** | ~67 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Megatron_TRT) |
| `Gundam_RK3588` | RK3588 NPU | RKNN + RGA | **512 维** | ~35 MB | [v1.0.0](https://github.com/joy999/go-face/releases/download/v1.0.0/Gundam_RK3588) |

四套模型均输出 **512 维 ArcFace 特征向量**，因此可跨平台互相比较。

> 如需其他模型包（RK356X、RV1106 等），请从 [InspireFace releases](https://github.com/HyperInspire/InspireFace/releases) 页面下载，或使用官方脚本 `command/download_models_general.sh`。
>
> 如果本地没有模型文件，`InitAuto` 会自动从 [GitHub releases](https://github.com/joy999/go-face/releases) 下载最适合当前平台的模型包。

### 2. 在 Go 项目中使用

```bash
go get github.com/joy999/go-face
```

#### 自动模型选择（推荐）

```go
modelPath, err := goface.InitAuto("./models")
if err != nil {
    log.Fatal(err)
}
fmt.Println("已加载模型:", modelPath)
defer goface.Deinit()
```

`InitAuto` 按以下优先级尝试加载：
1. **RK3588 平台**：`Gundam_RK3588` → `Megatron` → `Pikachu`
2. **其他平台**：`Megatron` → `Pikachu`

如果模型目录为空或不存在，`InitAuto` 会自动从 GitHub releases 下载最佳匹配的模型包，然后再初始化。

也可以手动下载指定模型：

```go
if err := goface.DownloadModel("./models", "Pikachu"); err != nil {
    log.Fatal(err)
}
```

#### 手动初始化

```go
package main

import (
    "fmt"
    "log"
    goface "github.com/joy999/go-face"
)

func main() {
    // 1. 全局初始化（每个进程只需一次）
    if err := goface.Init(goface.InitOption{
        ModelPath: "./models/Pikachu",
        Backend:   goface.BackendAuto,
    }); err != nil {
        log.Fatal(err)
    }
    defer goface.Deinit()

    // 2. 创建会话
    sess, err := goface.NewSession(goface.DefaultSessionOption())
    if err != nil {
        log.Fatal(err)
    }
    defer sess.Close()

    // 3. 原始图像字节（来自摄像头 / gstreamer / 解码器）
    width, height := 640, 480
    img := make([]byte, width*height*3) // 本例为 BGR

    // 4. 检测人脸
    faces, err := sess.Detect(img, width, height, goface.FormatBGR)
    if err != nil {
        log.Fatal(err)
    }

    for i, f := range faces {
        fmt.Printf("人脸 #%d 区域=%v 置信度=%.3f\n", i, f.Rect, f.Confidence)
        if f.Feature != nil {
            fmt.Printf("  特征维度=%d\n", len(f.Feature))
        }
        if f.FaceImage != nil {
            fmt.Printf("  对齐人脸=%dx%dx%d\n",
                f.FaceImage.Width, f.FaceImage.Height, f.FaceImage.Channels)
        }
    }

    // 5. 比较两张人脸
    if len(faces) >= 2 {
        sim := goface.CompareFeatures(faces[0].Feature, faces[1].Feature)
        fmt.Printf("相似度=%.3f (阈值=%.3f)\n",
            sim, goface.RecommendedThreshold())
    }
}
```

### 3. 编译

```bash
# macOS / 通用 Linux x86 / 通用 Linux aarch64
go build

# RK3588（启用 RKNN + RGA 库）
go build -tags rk3588
```

### 4. 运行时库路径

预置的 `.dylib`/`.so` 已通过 `@rpath` / `-Wl,-rpath` 引用，通常无需额外配置 `LD_LIBRARY_PATH`。若希望系统级安装，可将库文件复制到 `/usr/local/lib`（Linux 下 `/usr/lib`）并执行 `ldconfig`。

## 项目结构

```
go-face/
├── goface.c, goface.h          # C 核心层（所有 InspireFace 逻辑）
├── face.go                      # CGO 绑定 + Go 高层/底层 API
├── types.go                     # Go 类型定义
├── auto.go                      # 自动模型选择 API
├── auto_default.go              # 非 RK3588 平台的自动选择逻辑
├── auto_rk3588.go               # RK3588 平台的自动选择逻辑（构建标签: rk3588）
├── face_test.go                 # 单元测试
├── face_e2e_test.go             # 端到端流水线测试
├── bench_feature_test.go        # 特征提取性能基准测试（CPU 与 RK3588）
├── cgo_darwin.go                # macOS 链接标志
├── cgo_linux.go                 # Linux x86 链接标志
├── cgo_linux_arm64.go           # Linux aarch64 链接标志
├── cgo_linux_rk3588.go          # RK3588 链接标志（构建标签: rk3588）
├── go.mod                       # Go 1.21
├── README.md / README_CN.md
├── models/
│   ├── Pikachu/                 # 轻量 CPU 模型包（512 维，MNN）
│   ├── Megatron/                # 高性能 CPU 模型包（512 维，MNN）
│   ├── Megatron_TRT/            # TensorRT GPU 模型包（512 维）
│   └── Gundam_RK3588/           # RK3588 模型包（512 维，RKNN+RGA）
├── third_party/
│   └── inspireface/
│       ├── include/             # InspireFace C 头文件
│       └── lib/                 # 各平台预编译库
│           ├── darwin_arm64/
│           ├── linux_x86/
│           └── linux_aarch64_rk3588/
├── examples/
│   ├── simple/main.go           # 高层 API 用法
│   ├── lowlevel/main.go         # 底层 API 用法（零拷贝）
│   ├── autotest/main.go         # InitAuto 压力测试
│   └── server/                  # HTTP REST 服务（InsightFace 风格）
│       ├── main.go
│       └── README.md
```

## API 参考

### 全局生命周期

```go
// 初始化 SDK（每个进程只需一次）
err := goface.Init(goface.InitOption{
    ModelPath: "./models/Pikachu",  // 模型包路径（tar 归档文件或含对应 .tar 的目录）
    Backend:   goface.BackendAuto,  // BackendAuto / BackendCPU / BackendRGA
    LogLevel:  2,                   // 0=error, 1=warn, 2=info, 3=debug
})
defer goface.Deinit()
```

### 自动模型选择

```go
// 根据当前平台自动选择最优可用模型。
// RK3588 平台：Gundam_RK3588 → Megatron → Pikachu
// 其他平台：  Megatron → Pikachu
modelPath, err := goface.InitAuto("./models")
if err != nil {
    log.Fatal(err)
}
fmt.Println("已加载模型:", modelPath)
defer goface.Deinit()
```

### 会话选项

```go
opt := goface.DefaultSessionOption()
// 或自定义：
opt := goface.SessionOption{
    MaxFaces:          10,   // 最大检测人脸数
    DetectPixelLevel:  320,  // 检测分辨率：160、320、640；-1 为默认
    EnableRecognition: 1,    // 提取 512 维特征向量
    EnableFacePose:    1,    // 头部姿态（roll/yaw/pitch）
    EnableQuality:     0,    // 人脸质量评估
    EnableLiveness:    0,    // RGB 活体检测
    EnableMaskDetect:  0,    // 口罩检测
}
sess, err := goface.NewSession(opt)
defer sess.Close()
```

### 人脸检测

```go
// 默认旋转（Rotation0）
faces, err := sess.Detect(data, width, height, goface.FormatBGR)

// 指定摄像头旋转角度
faces, err := sess.DetectWithRotation(data, width, height, goface.FormatNV12, goface.Rotation90)

// Face 结构字段：
//   Rect       image.Rectangle
//   Roll/Yaw/Pitch  float32
//   Confidence float32
//   Feature    []float32   // 512 维，若未启用人脸识别则为 nil
//   FaceImage  *FaceImage  // 对齐裁剪后的人脸图，若不可用则为 nil
```

### 特征比对

```go
sim := goface.CompareFeatures(featA, featB)   // 余弦相似度 [-1, 1]
thr := goface.RecommendedThreshold()            // 约 0.48（视模型而定）
if sim > thr {
    // 同一人
}
```

### 错误码

```go
msg := goface.StrError(code)  // 将 InspireFace 错误码转为可读字符串
```

### 底层 API

适合需要自主管理内存或构建自定义对象池的高级用户：

```go
ptr, _ := goface.LowLevelSessionCreate(goface.DefaultSessionOption())
defer goface.LowLevelSessionDestroy(ptr)

result, count, err := goface.LowLevelDetect(ptr, img, w, h, goface.FormatBGR, goface.Rotation0)
if err != nil { ... }
defer goface.LowLevelResultFree(result)

for i := 0; i < count; i++ {
    var f goface.Face
    goface.LowLevelResultGetFace(result, i, &f)
    // 使用 f ...
}
```

## 图像格式

| Go 常量 | 说明 | 典型来源 |
|-------------|-------------|----------------|
| `FormatRGB` | 交错 RGB | OpenCV RGB |
| `FormatBGR` | 交错 BGR | OpenCV 默认 |
| `FormatRGBA`| RGB + Alpha | 带透明通道的 PNG |
| `FormatBGRA`| BGR + Alpha | 图形 API |
| `FormatNV12`| YUV NV12 | GStreamer `nv12`、iOS 摄像头 |
| `FormatNV21`| YUV NV21 | Android 摄像头 |
| `FormatI420`| YUV I420 | 视频编解码器 |
| `FormatGRAY`| 单通道 | 红外 / 深度相机 |

## Vendor 模式与原生库

`go mod vendor` **不会**复制没有 `.go` 文件的目录，并且会跳过超过 1 MB 的文件。这会导致以下问题：

- **`third_party/inspireface/include/` 下的 C 头文件** 默认不会被 vendor。我们在 v1.0.3+ 中通过将头文件目录转为占位 Go 包的方式修复了此问题，确保它们能被正确 vendor。
- **`third_party/inspireface/lib/` 下的 `.so` / `.dylib` 文件** 由于每个都有数 MB，**永远不会**被 vendor 复制。

### 自动安装（推荐）

执行完 `go mod vendor` 后，运行辅助工具自动下载并解压原生库：

```bash
# 若启用了 vendor/，需加 -mod=mod 才能从网络获取该工具
#（或者直接从本地 go-face 仓库运行）
go run -mod=mod github.com/joy999/go-face/cmd/install-libs

# 从本地 clone 运行：
go run /path/to/go-face/cmd/install-libs
```

该工具会自动检测 `go-face` 是位于你的 `vendor/` 目录中还是在模块缓存里，并将 `lib.tar.gz` 解压到正确的 `third_party/inspireface/lib/` 位置。

如需安装到系统路径（方便所有构建共用）：

```bash
go run github.com/joy999/go-face/cmd/install-libs -system
sudo ldconfig   # Linux 上需要执行
```

### 手工部署

如果你处于防火墙内或希望完全手动控制，可一次性下载 [`lib.tar.gz`](https://github.com/joy999/go-face/releases/download/init/lib.tar.gz)，然后自行放置库文件。

**A. Vendor 模式 — 放到 `vendor/github.com/joy999/go-face/` 内**

```bash
# 1. 找到 go-face 在 vendor 中的目录
PKG_DIR=$(go list -f '{{.Dir}}' github.com/joy999/go-face)
echo $PKG_DIR
# → /your/project/vendor/github.com/joy999/go-face

# 2. 将 lib.tar.gz 解压到该包的 third_party 目录下
mkdir -p "$PKG_DIR/third_party/inspireface/lib"
tar -xzf lib.tar.gz -C "$PKG_DIR/third_party/inspireface/lib"
```

**B. Module-cache 模式 — 放到 `$GOMODCACHE` 中**

```bash
# 1. 找到 go-face 在模块缓存中的目录
PKG_DIR=$(go list -f '{{.Dir}}' github.com/joy999/go-face)
echo $PKG_DIR
# → /home/user/go/pkg/mod/github.com/joy999/go-face@v1.0.3

# 2. 将 lib.tar.gz 解压到同一位置
tar -xzf lib.tar.gz -C "$PKG_DIR/third_party/inspireface/lib"
```

**C. 系统路径安装 — 放到 `/usr/local/lib`（Linux）或 `/usr/lib`**

只拷贝当前平台需要的库文件即可：

| 平台 | 需拷贝的文件 |
|----------|---------------|
| macOS (arm64) | `darwin_arm64/libInspireFace.dylib` → `/usr/local/lib/` |
| Linux x86 (amd64) | `linux_x86/libInspireFace.so` → `/usr/local/lib/` |
| Linux aarch64 / RK3588 | `linux_aarch64_rk3588/libInspireFace.so` + `linux_aarch64_rk3588/librknnrt.so` → `/usr/local/lib/` |

拷贝完成后刷新动态链接器缓存：

```bash
sudo ldconfig
```

CGO 中已添加的 `-L/usr/local/lib` 兜底路径（v1.0.3+）会自动找到这些库。

**D. 从本地 `replace` 的 clone 中复制**

如果你使用了 `replace github.com/joy999/go-face => /path/to/local/go-face`，本地仓库中的 `.so` 文件已经是真实内容（不是 LFS 指针），可直接复制：

```bash
# 复制到项目 vendor 目录
cp -r /path/to/local/go-face/third_party/inspireface/lib/* \
      ./vendor/github.com/joy999/go-face/third_party/inspireface/lib/

# 或者安装到系统路径
cp /path/to/local/go-face/third_party/inspireface/lib/linux_aarch64_rk3588/*.so /usr/local/lib/
sudo ldconfig
```

## 性能优化

- **复用 Session**。`Session` 内部分配了缓存和 GPU/NPU 内存。建议每个 goroutine 持有一个 Session，或使用对象池；不要每个请求都创建/销毁。
- **匹配合适的检测分辨率**。`DetectPixelLevel: 160` 最快；`640` 最准；`320` 是大多数场景的甜点值。
- **按需开启功能**。每个选项（`EnableLiveness`、`EnableMaskDetect` 等）都会增加内存占用和延迟。关闭不需要的功能。
- **RK3588 上使用 `BackendRGA`** 以启用 Rockchip RGA 硬件加速。若 RGA 不可用，库会自动回退 CPU。`BackendAuto` 当前默认使用 CPU 处理。
- **使用 `InitAuto`**。它会自动选择当前硬件上性能最高的可用模型。
- **NV12/NV21 零拷贝**。若你的流水线已输出 NV12（如 GStreamer `nv12`），直接传入，无需转为 BGR/RGB。

## 常见问题排查

| 现象 | 可能原因 | 解决方法 |
|---------|--------------|-----|
| `HERR_ARCHIVE_LOAD_MODEL_FAILURE` (252) | 模型包版本不匹配或缺少 `.mnn`/`.rknn` 扩展名 | 使用与平台匹配的模型包；本仓库内置的模型包已修正 |
| `Model path does not exist` | `ModelPath` 指向了不存在的路径 | 确保路径与内置模型包之一匹配（如 `./models/Pikachu`） |
| `no session available`（HTTP 服务） | 池内所有 Session 都在忙 | 增大 `POOL_SIZE` 或减少并发请求 |
| `HERR_INVALID_IMAGE_STREAM_PARAM` | 宽/高/格式不匹配 | 确认 `len(data)` 与所选格式的 `width*height*channels` 一致 |
| vendor 编译时 `inspireface.h: No such file or directory` | `go mod vendor` 默认不复制头文件目录 | 升级到 v1.0.3+，重新执行 `go mod vendor`；头文件已通过占位 Go 包被正确复制 |
| vendor 编译时 `cannot find -lInspireFace` | `.so` 文件超过 1 MB，被 `go mod vendor` 跳过 | vendor 后运行 `go run github.com/joy999/go-face/cmd/install-libs`，或将库安装到 `/usr/local/lib` |
| macOS: `image not found` | `@rpath` 解析失败 | `export DYLD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/darwin_arm64` |
| Linux x86: `cannot open shared object file` | 加载器未识别 `rpath` | `export LD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/linux_x86` |
| Linux aarch64: `cannot open shared object file` | 加载器未识别 `rpath` | `export LD_LIBRARY_PATH=$(pwd)/third_party/inspireface/lib/linux_aarch64_rk3588` |

## 架构设计

```
┌─────────────────────────────────────────────┐
│  Go 应用层                                    │
│  (Session.Detect / LowLevelDetect)           │
├─────────────────────────────────────────────┤
│  Go I/O 层（类型、切片包装）                  │
├─────────────────────────────────────────────┤
│  CGO — 纯 I/O 桥梁                           │
│  (C.go → Go, Go → C 结构体)                  │
├─────────────────────────────────────────────┤
│  goface.c / goface.h                         │
│  • 初始化 / 反初始化                          │
│  • 会话创建 / 销毁                            │
│  • 检测（流 → 追踪 → 提取 → 裁剪）            │
│  • 特征比对                                   │
├─────────────────────────────────────────────┤
│  InspireFace C API                           │
│  (libInspireFace.so / .dylib)               │
└─────────────────────────────────────────────┘
```

所有流程控制（循环、条件判断、资源清理）均留在 `goface.c` 内部。Go 从不直接调用单个 InspireFace API。

## 许可证

本仓库中的**代码**采用 MIT 许可证发布。

> ⚠️ InspireFace 模型包遵循与 InsightFace 相同的许可证：**仅限学术研究使用**，商业应用需获得授权。详见 [InspireFace License](https://github.com/HyperInspire/InspireFace#license)。
