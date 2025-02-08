package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	version    = "1.2.0"
	releaseURL = "https://github.com/duckdb/duckdb/releases/download/v%s/static-lib-%s.zip"
)

var platforms = map[string]map[string]string{
	"darwin": {
		"arm64": "osx-arm64",
		"amd64": "osx-amd64",
	},
	"linux": {
		"arm64": "linux-arm64",
		"amd64": "linux-amd64",
	},
	"windows": {
		"amd64": "windows-mingw",
	},
}

func main() {
	flagOs := flag.String("os", runtime.GOOS, "Target OS name for setup")
	flagArch := flag.String("arch", runtime.GOARCH, "Target arch name for setup")
	flag.Parse()
	goos := *flagOs
	arch := *flagArch

	log.Printf("Detected GOOS: %s, %s\n", goos, arch)

	platformMap, ok := platforms[goos]
	if !ok {
		log.Fatal(fmt.Sprintf("unsupported GOOS: %s", goos))
	}

	mapping, ok := platformMap[arch]
	if !ok {
		log.Fatal(fmt.Sprintf("unsupported GOARCH: %s for GOOS: %s", arch, goos))
	}

	url := fmt.Sprintf(releaseURL, version, mapping)
	log.Printf("Downloading from %s...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, err := os.CreateTemp("", "duckdb-static-lib-*.zip")
	if err != nil {
		log.Fatal(err)
	}
	zipFile := out.Name()
	if _, err := io.Copy(out, resp.Body); err != nil {
		log.Fatal(err)
	}
	out.Close()

	log.Println("Download complete, extracting zip file.")

	r, err := zip.OpenReader(zipFile)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer rc.Close()
		os.MkdirAll(filepath.Dir(f.Name), 0755)
		dst, err := os.Create(f.Name)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(dst, rc); err != nil {
			log.Fatal(err)
		}
		dst.Close()
	}
	os.Remove(zipFile)

	log.Println("Extraction done, finalizing dependencies.")

	depDir := fmt.Sprintf("deps/%s_%s", goos, arch)
	os.MkdirAll(depDir, 0755)
	os.WriteFile(filepath.Join(depDir, "vendor.go"), []byte(fmt.Sprintf("package %s_%s", goos, arch)), 0644)
	os.Rename("libduckdb_bundle.a", filepath.Join(depDir, "libduckdb.a"))
	os.Rename("duckdb.h", "duckdb.h")

	log.Println("Setup process completed.")
}
