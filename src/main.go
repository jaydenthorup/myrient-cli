package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("== Redump Myrient Browser ==")
	InitProgress()
	defer WaitForProgress()

	cats, err := FetchCategories()
	if err != nil {
		log.Fatalf("Failed to download category: %v", err)
	}

	selected := ShowCategoryMenu(cats)

	games, err := FetchGamesFromCategory(selected.URL)

	if err != nil {
		log.Fatalf("Failed to fetch games: %v", err)
	}

	ShowGames(games)
}
