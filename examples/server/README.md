# Face Recognition HTTP Server

A lightweight HTTP server built on `go-face`, providing face detection and comparison APIs similar to InsightFace.

## Endpoints

### `GET /health`
Health check.

```bash
curl http://localhost:8080/health
```

### `POST /detect`
Upload an image and get face detection / recognition results.

**Parameters (multipart/form-data):**
- `image` — Image file (JPEG / PNG)
- `feature` — `"true"` (default) to return 512-dim feature vector; `"false"` to omit
- `face_image` — `"true"` to return aligned face image (base64 RGB); default `"false"`

**Example:**
```bash
curl -X POST http://localhost:8080/detect \
  -F "image=@photo.jpg" \
  -F "feature=true" \
  -F "face_image=true"
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "count": 2,
  "width": 1920,
  "height": 1080,
  "faces": [
    {
      "rect": {"x": 340, "y": 520, "width": 180, "height": 220},
      "confidence": 0.987,
      "pose": {"roll": 5.2, "yaw": -3.1, "pitch": 1.5},
      "feature": [0.12, -0.05, ...],
      "face_image": "iVBORw0KGgo..."
    }
  ]
}
```

### `POST /compare`
Upload two images and compare the most prominent face in each.

**Parameters (multipart/form-data):**
- `image1` — First image file
- `image2` — Second image file

**Example:**
```bash
curl -X POST http://localhost:8080/compare \
  -F "image1=@faceA.jpg" \
  -F "image2=@faceB.jpg"
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "similarity": 0.823,
  "threshold": 0.48,
  "same_person": true
}
```

## Run

### macOS / Linux (generic)
```bash
cd examples/server
go run main.go
```

### RK3588
```bash
cd examples/server
go run -tags rk3588 main.go
```

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `MODEL_PATH` | `./models/Pikachu` | Path to InspireFace model pack |
| `LISTEN` | `:8080` | HTTP listen address |
| `POOL_SIZE` | `4` | Session pool size (concurrency) |
| `DEMO` | `0` | Set to `1` to run in demo mode (simulated responses, no real inference) |

### Example with custom model
```bash
MODEL_PATH=../../models/Gundam_RK3588 POOL_SIZE=8 go run -tags rk3588 main.go
```
