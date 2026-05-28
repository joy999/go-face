package goface

import (
	"image/jpeg"
	"os"
	"sort"
	"testing"
	"time"
)

func BenchmarkFeatureExtraction_CPU(b *testing.B) {
	modelDir := findModelDir("Pikachu")
	if modelDir == "" {
		b.Skip("Pikachu model not found in models/")
	}

	err := Init(InitOption{
		ModelPath: modelDir,
		Backend:   BackendCPU,
		LogLevel:  0,
	})
	if err != nil {
		b.Skipf("Init skipped: %v", err)
	}
	defer Deinit()

	sess, err := NewSession(DefaultSessionOption())
	if err != nil {
		b.Fatalf("NewSession failed: %v", err)
	}
	defer sess.Close()

	imgPath := "/tmp/test_face.jpg"
	f, err := os.Open(imgPath)
	if err != nil {
		b.Fatalf("open image failed: %v", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		b.Fatalf("jpeg decode failed: %v", err)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	bgr := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := (y*width + x) * 3
			bgr[idx+0] = byte(b >> 8)
			bgr[idx+1] = byte(g >> 8)
			bgr[idx+2] = byte(r >> 8)
		}
	}

	faces, err := sess.Detect(bgr, width, height, FormatBGR)
	if err != nil {
		b.Fatalf("warm-up detect failed: %v", err)
	}
	if len(faces) == 0 {
		b.Fatal("warm-up: no face detected")
	}
	featDim := len(faces[0].Feature)
	b.Logf("Warm-up OK: detected %d face(s), feature dim = %d", len(faces), featDim)

	const iterations = 100
	times := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		faces, err := sess.Detect(bgr, width, height, FormatBGR)
		elapsed := time.Since(start)
		if err != nil {
			b.Fatalf("iteration %d failed: %v", i, err)
		}
		if len(faces) == 0 {
			b.Fatalf("iteration %d: no face detected", i)
		}
		times[i] = elapsed
	}

	var total time.Duration
	min := times[0]
	max := times[0]
	for _, t := range times {
		total += t
		if t < min { min = t }
		if t > max { max = t }
	}
	avg := total / iterations

	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	p50 := times[iterations*50/100]
	p95 := times[iterations*95/100]
	p99 := times[iterations*99/100]

	b.Logf("=== CPU Feature Extraction Benchmark (%d iters, %d-D) ===", iterations, featDim)
	b.Logf("Total   : %v", total)
	b.Logf("Average : %v", avg)
	b.Logf("Min     : %v", min)
	b.Logf("Max     : %v", max)
	b.Logf("P50     : %v", p50)
	b.Logf("P95     : %v", p95)
	b.Logf("P99     : %v", p99)
	b.Logf("Throughput: %.1f faces/sec", float64(iterations)/total.Seconds())
}

func BenchmarkFeatureExtraction_RK3588(b *testing.B) {
	modelDir := findModelDir("Gundam_RK3588")
	if modelDir == "" {
		b.Skip("Gundam_RK3588 model not found in models/")
	}

	err := Init(InitOption{
		ModelPath: modelDir,
		Backend:   BackendAuto,
		LogLevel:  0,
	})
	if err != nil {
		b.Skipf("Init skipped: %v", err)
	}
	defer Deinit()

	sess, err := NewSession(DefaultSessionOption())
	if err != nil {
		b.Fatalf("NewSession failed: %v", err)
	}
	defer sess.Close()

	imgPath := "/tmp/test_face.jpg"
	f, err := os.Open(imgPath)
	if err != nil {
		b.Fatalf("open image failed: %v", err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		b.Fatalf("jpeg decode failed: %v", err)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	bgr := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := (y*width + x) * 3
			bgr[idx+0] = byte(b >> 8)
			bgr[idx+1] = byte(g >> 8)
			bgr[idx+2] = byte(r >> 8)
		}
	}

	faces, err := sess.Detect(bgr, width, height, FormatBGR)
	if err != nil {
		b.Fatalf("warm-up detect failed: %v", err)
	}
	if len(faces) == 0 {
		b.Fatal("warm-up: no face detected")
	}
	if len(faces[0].Feature) == 0 {
		b.Fatal("warm-up: feature vector is empty")
	}
	featDim := len(faces[0].Feature)
	b.Logf("Warm-up OK: detected %d face(s), feature dim = %d", len(faces), featDim)

	const iterations = 100
	times := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		faces, err := sess.Detect(bgr, width, height, FormatBGR)
		elapsed := time.Since(start)
		if err != nil {
			b.Fatalf("iteration %d failed: %v", i, err)
		}
		if len(faces) == 0 {
			b.Fatalf("iteration %d: no face detected", i)
		}
		times[i] = elapsed
	}

	var total time.Duration
	min := times[0]
	max := times[0]
	for _, t := range times {
		total += t
		if t < min { min = t }
		if t > max { max = t }
	}
	avg := total / iterations

	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	p50 := times[iterations*50/100]
	p95 := times[iterations*95/100]
	p99 := times[iterations*99/100]

	b.Logf("=== RK3588 Feature Extraction Benchmark (%d iters, %d-D) ===", iterations, featDim)
	b.Logf("Total   : %v", total)
	b.Logf("Average : %v", avg)
	b.Logf("Min     : %v", min)
	b.Logf("Max     : %v", max)
	b.Logf("P50     : %v", p50)
	b.Logf("P95     : %v", p95)
	b.Logf("P99     : %v", p99)
	b.Logf("Throughput: %.1f faces/sec", float64(iterations)/total.Seconds())
}
