package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	goface "github.com/joy999/go-face"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Pose struct {
	Roll  float32 `json:"roll"`
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`
}

type FaceResult struct {
	Rect       Rect      `json:"rect"`
	Confidence float32   `json:"confidence"`
	Pose       Pose      `json:"pose"`
	Feature    []float32 `json:"feature,omitempty"`
	FaceImage  string    `json:"face_image,omitempty"` // base64 RGB
}

type DetectResponse struct {
	Code   int          `json:"code"`
	Msg    string       `json:"msg"`
	Count  int          `json:"count"`
	Faces  []FaceResult `json:"faces"`
	Width  int          `json:"width"`
	Height int          `json:"height"`
}

type CompareResponse struct {
	Code       int     `json:"code"`
	Msg        string  `json:"msg"`
	Similarity float32 `json:"similarity"`
	Threshold  float32 `json:"threshold"`
	SamePerson bool    `json:"same_person"`
}

// ---------------------------------------------------------------------------
// Session Pool
// ---------------------------------------------------------------------------

type SessionPool struct {
	pool []*goface.Session
	mu   sync.Mutex
}

func NewSessionPool(size int) (*SessionPool, error) {
	p := &SessionPool{pool: make([]*goface.Session, 0, size)}
	for i := 0; i < size; i++ {
		s, err := goface.NewSession(goface.DefaultSessionOption())
		if err != nil {
			return nil, fmt.Errorf("create session %d: %w", i, err)
		}
		p.pool = append(p.pool, s)
	}
	return p, nil
}

func (p *SessionPool) Acquire() *goface.Session {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pool) == 0 {
		s, err := goface.NewSession(goface.DefaultSessionOption())
		if err != nil {
			log.Printf("[pool] hot-create session failed: %v", err)
			return nil
		}
		return s
	}
	s := p.pool[len(p.pool)-1]
	p.pool = p.pool[:len(p.pool)-1]
	return s
}

func (p *SessionPool) Release(s *goface.Session) {
	if s == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = append(p.pool, s)
}

func (p *SessionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range p.pool {
		s.Close()
	}
	p.pool = nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeToBGR(img image.Image) ([]byte, int, int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	data := make([]byte, w*h*3)

	switch m := img.(type) {
	case *image.RGBA:
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				offset := m.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
				r := m.Pix[offset]
				g := m.Pix[offset+1]
				b := m.Pix[offset+2]
				idx := (y*w + x) * 3
				data[idx+0] = b // B
				data[idx+1] = g // G
				data[idx+2] = r // R
			}
		}
	case *image.NRGBA:
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				offset := m.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
				r := m.Pix[offset]
				g := m.Pix[offset+1]
				b := m.Pix[offset+2]
				idx := (y*w + x) * 3
				data[idx+0] = b
				data[idx+1] = g
				data[idx+2] = r
			}
		}
	case *image.YCbCr:
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, _ := m.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
				idx := (y*w + x) * 3
				data[idx+0] = byte(b >> 8)
				data[idx+1] = byte(g >> 8)
				data[idx+2] = byte(r >> 8)
			}
		}
	default:
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
				idx := (y*w + x) * 3
				data[idx+0] = byte(b >> 8)
				data[idx+1] = byte(g >> 8)
				data[idx+2] = byte(r >> 8)
			}
		}
	}
	return data, w, h
}

func processImage(sess *goface.Session, file io.Reader, withFeature, withFaceImage bool) (*DetectResponse, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	data, w, h := decodeToBGR(img)
	faces, err := sess.Detect(data, w, h, goface.FormatBGR)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}

	resp := &DetectResponse{
		Code:   0,
		Msg:    "ok",
		Count:  len(faces),
		Faces:  make([]FaceResult, 0, len(faces)),
		Width:  w,
		Height: h,
	}

	for _, f := range faces {
		fr := FaceResult{
			Rect: Rect{
				X: f.Rect.Min.X, Y: f.Rect.Min.Y,
				Width: f.Rect.Dx(), Height: f.Rect.Dy(),
			},
			Confidence: f.Confidence,
			Pose:       Pose{Roll: f.Roll, Yaw: f.Yaw, Pitch: f.Pitch},
		}
		if withFeature {
			fr.Feature = f.Feature
		}
		if withFaceImage && f.FaceImage != nil {
			fr.FaceImage = base64.StdEncoding.EncodeToString(f.FaceImage.Data)
		}
		resp.Faces = append(resp.Faces, fr)
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Demo mode helpers (no real inference, for API validation only)
// ---------------------------------------------------------------------------

func demoDetect(w int, h int, withFeature, withFaceImage bool) *DetectResponse {
	resp := &DetectResponse{
		Code: 0, Msg: "ok (demo mode)", Count: 1,
		Width: w, Height: h,
		Faces: []FaceResult{{
			Rect:       Rect{X: w / 4, Y: h / 4, Width: w / 2, Height: h / 2},
			Confidence: 0.987,
			Pose:       Pose{Roll: 5.2, Yaw: -3.1, Pitch: 1.5},
		}},
	}
	if withFeature {
		feat := make([]float32, 512)
		for i := range feat {
			feat[i] = float32(i%100) / 100.0
		}
		resp.Faces[0].Feature = feat
	}
	if withFaceImage {
		// 112x112x3 RGB dummy image
		img := make([]byte, 112*112*3)
		for i := range img {
			img[i] = byte(i % 256)
		}
		resp.Faces[0].FaceImage = base64.StdEncoding.EncodeToString(img)
	}
	return resp
}

func demoCompare() *CompareResponse {
	return &CompareResponse{
		Code:       0,
		Msg:        "ok (demo mode)",
		Similarity: 0.823,
		Threshold:  0.48,
		SamePerson: true,
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func detectHandler(pool *SessionPool, demo bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, DetectResponse{Code: -1, Msg: "POST only"})
			return
		}

		file, _, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, DetectResponse{Code: -1, Msg: "missing image field"})
			return
		}
		defer file.Close()

		withFeature := r.FormValue("feature") != "false"
		withFaceImage := r.FormValue("face_image") == "true"

		if demo {
			img, _, err := image.Decode(file)
			iw, ih := 640, 480
			if err == nil && img != nil {
				b := img.Bounds()
				iw, ih = b.Dx(), b.Dy()
			}
			writeJSON(w, http.StatusOK, demoDetect(iw, ih, withFeature, withFaceImage))
			return
		}

		sess := pool.Acquire()
		if sess == nil {
			writeJSON(w, http.StatusServiceUnavailable, DetectResponse{Code: -1, Msg: "no session available"})
			return
		}
		defer pool.Release(sess)

		resp, err := processImage(sess, file, withFeature, withFaceImage)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, DetectResponse{Code: -1, Msg: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func compareHandler(pool *SessionPool, demo bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, CompareResponse{Code: -1, Msg: "POST only"})
			return
		}

		_, _, err := r.FormFile("image1")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CompareResponse{Code: -1, Msg: "missing image1"})
			return
		}

		_, _, err = r.FormFile("image2")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CompareResponse{Code: -1, Msg: "missing image2"})
			return
		}

		if demo {
			writeJSON(w, http.StatusOK, demoCompare())
			return
		}

		// Real mode: read both images and compare first face from each
		_ = pool // TODO: implement real compare using pool
		writeJSON(w, http.StatusNotImplemented, CompareResponse{Code: -1, Msg: "compare not yet implemented in real mode"})
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	modelPath := os.Getenv("MODEL_PATH")
	if modelPath == "" {
		modelPath = "./models/Pikachu"
	}
	listenAddr := os.Getenv("LISTEN")
	if listenAddr == "" {
		listenAddr = ":8080"
	}
	poolSize, _ := strconv.Atoi(os.Getenv("POOL_SIZE"))
	if poolSize <= 0 {
		poolSize = 4
	}
	demo := os.Getenv("DEMO") == "1"

	var pool *SessionPool

	if !demo {
		log.Printf("[server] initializing SDK with model: %s", modelPath)
		if err := goface.Init(goface.InitOption{
			ModelPath: modelPath,
			Backend:   goface.BackendAuto,
			LogLevel:  1,
		}); err != nil {
			log.Printf("[server] goface.Init failed: %v", err)
			log.Printf("[server] hint: set DEMO=1 to run in demo mode (simulated responses, no real inference)")
			os.Exit(1)
		}
		defer goface.Deinit()

		log.Printf("[server] creating session pool (size=%d)", poolSize)
		var err error
		pool, err = NewSessionPool(poolSize)
		if err != nil {
			log.Fatalf("[server] NewSessionPool failed: %v", err)
		}
		defer pool.Close()
	} else {
		log.Printf("[server] running in DEMO mode (no real inference)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/detect", detectHandler(pool, demo))
	mux.HandleFunc("/compare", compareHandler(pool, demo))

	log.Printf("[server] listening on %s", listenAddr)
	log.Printf("[server] endpoints: POST /detect, POST /compare, GET /health")
	if demo {
		log.Printf("[server] WARNING: demo mode returns synthetic data only")
	}
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("[server] ListenAndServe: %v", err)
	}
}
