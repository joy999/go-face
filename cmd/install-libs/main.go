package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const downloadURL = "https://github.com/joy999/go-face/releases/download/init/lib.tar.gz"

var (
	systemMode = flag.Bool("system", false, "Install libs to system path (/usr/local/lib) instead of package dir")
	force      = flag.Bool("force", false, "Force re-download even if libs already exist")
)

func main() {
	flag.Parse()

	if *systemMode {
		installToSystem()
		return
	}

	pkgDir, err := goPackageDir("github.com/joy999/go-face")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot locate github.com/joy999/go-face: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure you have run 'go mod tidy' and the package is available.")
		os.Exit(1)
	}

	libDir := filepath.Join(pkgDir, "third_party", "inspireface", "lib")

	// Check if libs already look valid (real .so > 1KB, not LFS pointer)
	if !*force && hasValidLibs(libDir) {
		fmt.Println("Libraries already exist. Use -force to re-download.")
		return
	}

	fmt.Println("Downloading", downloadURL)
	tmpFile, err := downloadToTemp(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	fmt.Println("Extracting to", libDir)
	if err := extractTarGz(tmpFile, libDir); err != nil {
		fmt.Fprintf(os.Stderr, "Extract failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done. Libraries installed to", libDir)
}

func installToSystem() {
	plat := runtime.GOOS + "/" + runtime.GOARCH
	files := platformFiles(plat)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Unsupported platform: %s\n", plat)
		os.Exit(1)
	}

	fmt.Println("Downloading", downloadURL)
	tmpFile, err := downloadToTemp(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile)

	// Extract only needed files to /usr/local/lib
	fmt.Println("Installing to /usr/local/lib ...")
	if err := extractSystemLibs(tmpFile, files); err != nil {
		fmt.Fprintf(os.Stderr, "Install failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done. Run 'sudo ldconfig' if needed.")
}

func goPackageDir(pkg string) (string, error) {
	out, err := exec.Command("go", "list", "-f", "{{.Dir}}", pkg).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", ee.Stderr)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func hasValidLibs(libDir string) bool {
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subDir := filepath.Join(libDir, e.Name())
		files, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			// Real .so is > 1MB; LFS pointer is ~132 bytes
			if info.Size() > 1024 {
				return true
			}
		}
	}
	return false
}

func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "go-face-libs-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func extractTarGz(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// Skip macOS resource fork files
		if strings.Contains(hdr.Name, "/._") {
			continue
		}

		target := filepath.Join(dst, hdr.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
		if err := os.Chmod(target, os.FileMode(hdr.Mode)); err != nil {
			return err
		}
	}
	return nil
}

func platformFiles(plat string) []string {
	switch plat {
	case "darwin/arm64":
		return []string{"darwin_arm64/libInspireFace.dylib"}
	case "linux/arm64":
		return []string{
			"linux_aarch64_rk3588/libInspireFace.so",
			"linux_aarch64_rk3588/librknnrt.so",
		}
	case "linux/amd64":
		return []string{"linux_x86/libInspireFace.so"}
	default:
		return nil
	}
}

func extractSystemLibs(src string, wanted []string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	wantMap := make(map[string]bool)
	for _, w := range wanted {
		wantMap[w] = true
	}

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if strings.Contains(hdr.Name, "/._") {
			continue
		}
		if !wantMap[hdr.Name] {
			continue
		}

		base := filepath.Base(hdr.Name)
		target := filepath.Join("/usr/local/lib", base)
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
		if err := os.Chmod(target, 0755); err != nil {
			return err
		}
		fmt.Println("Installed", target)
	}
	return nil
}
