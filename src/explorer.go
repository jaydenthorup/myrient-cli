package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type GameEntry struct {
	Title string
	Size  string
	URL   string
}

func FetchGamesFromCategory(categoryURL string) ([]GameEntry, error) {
	resp, err := http.Get(categoryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var games []GameEntry

	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		linkSel := s.Find("td.link a")
		sizeSel := s.Find("td.size")

		href, exists := linkSel.Attr("href")
		title, hasTitle := linkSel.Attr("title")
		size := strings.TrimSpace(sizeSel.Text())

		if exists && hasTitle && strings.HasSuffix(href, ".zip") {
			games = append(games, GameEntry{
				Title: title,
				Size:  size,
				URL:   categoryURL + href,
			})
		}
	})

	return games, nil
}

func FilterGamesByTitle(games []GameEntry, query string) []GameEntry {
	if query == "" {
		return games
	}

	var filtered []GameEntry

	// Try to compile as regex, fallback to substring if invalid
	re, err := regexp.Compile(query)
	if err == nil {
		for _, g := range games {
			if re.MatchString(g.Title) {
				filtered = append(filtered, g)
			}
		}
	} else {
		q := strings.ToLower(query)
		for _, g := range games {
			if strings.Contains(strings.ToLower(g.Title), q) {
				filtered = append(filtered, g)
			}
		}
	}

	return filtered
}

func PaginateGames(games []GameEntry, page int, pageSize int) []GameEntry {
	start := page * pageSize
	end := start + pageSize
	if start >= len(games) {
		return []GameEntry{}
	}
	if end > len(games) {
		end = len(games)
	}
	return games[start:end]
}

func loadDownloadedLog() map[string]bool {
	downloaded := make(map[string]bool)
	file, err := os.Open(downloadedLogPath)
	if err != nil {
		return downloaded
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		downloaded[scanner.Text()] = true
	}
	return downloaded
}

// Update isGameDownloaded to check downloaded log
func isGameDownloaded(game GameEntry) bool {
	downloaded := loadDownloadedLog()
	if downloaded[game.Title] {
		return true
	}
	dir := getDownloadDir()
	zipPath := filepath.Join(dir, sanitizeFilename(game.Title))
	partPath := zipPath + ".part"
	// If zip or extracted folder exists, consider as downloaded
	if _, err := os.Stat(zipPath); err == nil {
		return true
	}
	if _, err := os.Stat(partPath); err == nil {
		return false // partial file, not complete
	}
	// Check for extracted folder
	extractDir := filepath.Join(dir, strings.TrimSuffix(sanitizeFilename(game.Title), ".zip"))
	if fi, err := os.Stat(extractDir); err == nil && fi.IsDir() {
		return true
	}
	return false
}

func ShowGames(games []GameEntry) {
	const pageSize = 50
	reader := bufio.NewReader(os.Stdin)

	filtered := games
	query := ""
	page := 0

	downloaded := make(map[string]bool)
	var downloadedMutex sync.Mutex
	downloadQueue := make(chan GameEntry, 10) // concurrent-safe

	// start single background downloader worker
	go func() {
		for game := range downloadQueue {
			if isGameDownloaded(game) {
				fmt.Printf("⚠️  Already downloaded: %s\n", game.Title)
				continue
			}
			downloadedMutex.Lock()
			already := downloaded[game.Title]
			if !already {
				downloaded[game.Title] = true
			}
			downloadedMutex.Unlock()
			DownloadAndExtract(context.Background(), game)
		}
	}()

	for {
		if len(filtered) == 0 {
			fmt.Println("\nNo games found. Use (f) to enter a new filter or (q) to quit.")
		} else {
			paged := PaginateGames(filtered, page, pageSize)
			fmt.Printf("\n--- Page %d (%d–%d of %d) ---\n",
				page+1,
				page*pageSize+1,
				min((page+1)*pageSize, len(filtered)),
				len(filtered),
			)

			for i, game := range paged {
				globalIndex := page*pageSize + i
				fmt.Printf("[%d] %s (%s)\n", globalIndex, game.Title, game.Size)
			}
		}

		// Show regex filter instructions
		fmt.Println("\nRegex filter tips:")
		fmt.Println("  - Enter a regex pattern to match titles (e.g. ^Halo.*USA)")
		fmt.Println("  - Use | for OR (e.g. Mario|Zelda)")
		fmt.Println("  - Use (?i) for case-insensitive (e.g. (?i)halo)")
		fmt.Println("  - Leave empty to reset filter")

		fmt.Println("\n(n)ext page, (p)revious page, (f)ilter, (a)ll, (q)uit, or enter number to download:")
		fmt.Print("> ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(strings.ToLower(cmd))

		switch cmd {
		case "n":
			if (page+1)*pageSize < len(filtered) {
				page++
			} else {
				fmt.Println("No more pages.")
			}
		case "p":
			if page > 0 {
				page--
			} else {
				fmt.Println("Already at the first page.")
			}
		case "f":
			fmt.Print("Enter filter (supports regex, empty to reset): ")
			query, _ = reader.ReadString('\n')
			query = strings.TrimSpace(query)
			filtered = FilterGamesByTitle(games, query)
			page = 0
		case "a":
			attempted := len(filtered)
			queued := 0
			fmt.Printf("\nSummary: Attempted to queue %d games.\n\n", attempted)
			for _, game := range filtered {
				if isGameDownloaded(game) {
					continue
				}
				downloadedMutex.Lock()
				already := downloaded[game.Title]
				downloadedMutex.Unlock()
				if !already {
					fmt.Printf("📥 Queued for download: %s\n", game.Title)
					downloadQueue <- game
					queued++
				}
			}
			if queued == 0 {
				fmt.Println("No new games to queue for download.")
			} else {
				fmt.Printf("Queued %d new games for download.\n", queued)
			}
		case "q":
			close(downloadQueue)
			return
		default:
			// attempt to parse number
			index, err := strconv.Atoi(cmd)
			if err == nil && index >= 0 && index < len(filtered) {
				if isGameDownloaded(filtered[index]) {
					fmt.Printf("⚠️  Already downloaded: %s\n", filtered[index].Title)
					continue
				}
				downloadedMutex.Lock()
				already := downloaded[filtered[index].Title]
				downloadedMutex.Unlock()
				if !already {
					fmt.Printf("📥 Queued for download: %s\n", filtered[index].Title)
					downloadQueue <- filtered[index]
				} else {
					fmt.Printf("⚠️  Already downloading: %s\n", filtered[index].Title)
				}
			} else {
				fmt.Println("Invalid command or index.")
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
