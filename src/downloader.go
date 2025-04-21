package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func getDownloadDir() string {
	custom := os.Getenv("MYRIENT_DOWNLOADS_PATH")
	if custom != "" {
		return custom
	}
	return filepath.Join(".", ".downloads")
}

func ensureDirExists(path string) error {
	return os.MkdirAll(path, 0755)
}

func DownloadAndExtract(game GameEntry) {
	go func() {
		dir := getDownloadDir()
		err := ensureDirExists(dir)
		if err != nil {
			fmt.Printf("❌ Failed to create download dir: %v\n", err)
			return
		}

		zipPath := filepath.Join(dir, sanitizeFilename(game.Title))

		if _, err := os.Stat(zipPath); err == nil {
			fmt.Printf("⚠️  Already downloaded: %s\n", game.Title)
			return
		}

		resp, err := http.Get(game.URL)
		if err != nil {
			fmt.Printf("❌ Download error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("❌ HTTP error: %d\n", resp.StatusCode)
			return
		}

		bar := NewDownloadBar(trimTitle(game.Title), resp.ContentLength)

		reader := bar.ProxyReader(resp.Body)
		defer reader.Close()

		out, err := os.Create(zipPath)
		if err != nil {
			fmt.Printf("❌ File creation failed: %v\n", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, reader)
		if err != nil {
			fmt.Printf("❌ Write error: %v\n", err)
			return
		}

		bar.SetTotal(resp.ContentLength, true)

		err = unzipSingleWithProgress(zipPath, dir, game.Title)
		if err != nil {
			fmt.Printf("❌ Unzip failed: %v\n", err)
			return
		}

		os.Remove(zipPath)
	}()
}

func trimTitle(title string) string {
	if len(title) > 30 {
		return title[:27] + "..."
	}
	return title
}

func sanitizeFilename(name string) string {
	// Remove characters that are invalid in paths
	return strings.ReplaceAll(name, "/", "_")
}

func unzipSingleWithProgress(zipPath, dest string, gameTitle string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) != 1 {
		return fmt.Errorf("Expected 1 file inside ZIP, got %d", len(r.File))
	}

	f := r.File[0]
	fpath := filepath.Join(dest, f.Name)

	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return err
	}

	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	bar := NewDownloadBar(trimTitle(gameTitle+" (unzip)"), int64(f.UncompressedSize64))
	proxyReader := bar.ProxyReader(rc)

	_, err = io.Copy(outFile, proxyReader)
	if err != nil {
		return err
	}

	bar.SetTotal(int64(f.UncompressedSize64), true)
	return nil
}
