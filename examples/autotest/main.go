package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	goface "github.com/joy999/go-face"
)

func main() {
	basePath := "./models"
	if len(os.Args) > 1 {
		basePath = os.Args[1]
	}

	fmt.Println("=== go-face InitAuto stress test ===")
	fmt.Println("Base path:", basePath)
	fmt.Println("GOROOT:", runtime.GOROOT())
	fmt.Println("GOARCH:", runtime.GOARCH)
	fmt.Println("NumCPU:", runtime.NumCPU())
	fmt.Println("Start:", time.Now().Format("15:04:05"))

	// 循环反复初始化，持续运行一段时间
	const maxIter = 100
	var successCount, failCount int
	var lastModel string

	for i := 1; i <= maxIter; i++ {
		modelPath, err := goface.InitAuto(basePath)
		if err == nil {
			successCount++
			lastModel = modelPath
			fmt.Printf("[%3d/%d] OK  model=%s  time=%s\n", i, maxIter, modelPath, time.Now().Format("15:04:05"))

			// 如果初始化成功，创建 session 并做一次检测
			sess, err := goface.NewSession(goface.DefaultSessionOption())
			if err == nil {
				w, h := 640, 480
				img := make([]byte, w*h*3)
				faces, _ := sess.Detect(img, w, h, goface.FormatBGR)
				fmt.Printf("         Detect: %d face(s)\n", len(faces))
				sess.Close()
			}
			goface.Deinit()
		} else {
			failCount++
			fmt.Printf("[%3d/%d] FAIL  err=%v  time=%s\n", i, maxIter, err, time.Now().Format("15:04:05"))
			goface.Deinit()
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("=== stress test complete ===")
	fmt.Printf("Iterations: %d  Success: %d  Fail: %d  LastModel: %s\n", maxIter, successCount, failCount, lastModel)
	fmt.Println("End:", time.Now().Format("15:04:05"))
}
