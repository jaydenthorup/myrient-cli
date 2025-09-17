package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 2 * time.Minute}
var downloadedLogPath = "downloaded.log"

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

func logDownloaded(gameTitle string) {
	file, err := os.OpenFile(downloadedLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer file.Close()
		file.WriteString(gameTitle + "\n")
	}
}

func DownloadAndExtract(ctx context.Context, game GameEntry) {
	dir := getDownloadDir()
	err := ensureDirExists(dir)
	if err != nil {
		fmt.Printf("❌ Failed to create download dir: %v\n", err)
		return
	}

	zipPath := filepath.Join(dir, sanitizeFilename(game.Title))
	partPath := zipPath + ".part"

	// If already fully downloaded, skip
	if _, err := os.Stat(zipPath); err == nil {
		fmt.Printf("⚠️  Already downloaded: %s\n", game.Title)
		return
	}

	var startOffset int64 = 0
	var out *os.File
	if info, err := os.Stat(partPath); err == nil {
		// Resume from partial file
		startOffset = info.Size()
		out, err = os.OpenFile(partPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("❌ Could not open partial file: %v\n", err)
			return
		}
		fmt.Printf("⏩ Resuming download for %s at %d bytes\n", game.Title, startOffset)
	} else {
		out, err = os.Create(partPath)
		if err != nil {
			fmt.Printf("❌ File creation failed: %v\n", err)
			return
		}
	}
	defer out.Close()

	// HTTP request with Range header
	req, err := http.NewRequestWithContext(ctx, "GET", game.URL, nil)
	if err != nil {
		fmt.Printf("❌ Download request error: %v\n", err)
		return
	}
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Download error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		fmt.Printf("❌ HTTP error: %d\n", resp.StatusCode)
		return
	}

	bar := NewDownloadBar(trimTitle(game.Title), resp.ContentLength+startOffset)
	reader := bar.ProxyReader(resp.Body)
	defer reader.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		fmt.Printf("❌ Write error: %v\n", err)
		return
	}

	bar.SetTotal(resp.ContentLength+startOffset, true)

	// Rename .part to final zip
	if err := os.Rename(partPath, zipPath); err != nil {
		fmt.Printf("❌ Rename failed: %v\n", err)
		return
	}

	// Log successful download
	logDownloaded(game.Title)

	err = unzipSingleWithProgress(ctx, zipPath, dir, game.Title)
	if err != nil {
		fmt.Printf("❌ Unzip failed: %v\n", err)
		return
	}

	os.Remove(zipPath)
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

func unzipSingleWithProgress(ctx context.Context, zipPath, dest string, gameTitle string) error {
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

	// Use io.Copy with context cancellation
	copyDone := make(chan error, 1)
	go func() {
		_, err = io.Copy(outFile, proxyReader)
		copyDone <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-copyDone:
		if err != nil {
			return err
		}
	}

	bar.SetTotal(int64(f.UncompressedSize64), true)
	return nil
}
