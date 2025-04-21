package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"net/http"

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
	query = strings.ToLower(query)

	for _, g := range games {
		if strings.Contains(strings.ToLower(g.Title), query) {
			filtered = append(filtered, g)
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

func ShowGames(games []GameEntry) {
	const pageSize = 50
	reader := bufio.NewReader(os.Stdin)

	filtered := games
	query := ""
	page := 0

	downloaded := make(map[string]bool)
	downloadQueue := make(chan GameEntry, 10) // concurrent-safe

	// start background downloader worker
	go func() {
		for game := range downloadQueue {
			if downloaded[game.Title] {
				fmt.Printf("⚠️  Already downloading: %s\n", game.Title)
				continue
			}
			downloaded[game.Title] = true
			DownloadAndExtract(game)
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

		fmt.Println("\n(n)ext page, (p)revious page, (f)ilter, (q)uit, or enter number to download:")
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
			fmt.Print("Enter filter (empty to reset): ")
			query, _ = reader.ReadString('\n')
			query = strings.TrimSpace(query)
			filtered = FilterGamesByTitle(games, query)
			page = 0
		case "q":
			close(downloadQueue)
			return
		default:
			// attempt to parse number
			index, err := strconv.Atoi(cmd)
			if err == nil && index >= 0 && index < len(filtered) {
				fmt.Printf("📥 Queued for download: %s\n", filtered[index].Title)
				downloadQueue <- filtered[index]
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
