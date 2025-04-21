package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Category struct {
	Title string
	URL   string
}

const baseURL = "https://myrient.erista.me/files/Redump/"

func FetchCategories() ([]Category, error) {
	resp, err := http.Get(baseURL)
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

	var categories []Category

	doc.Find("tr > td > a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		title, hasTitle := s.Attr("title")
		if exists && hasTitle && strings.HasSuffix(href, "/") {
			categories = append(categories, Category{
				Title: title,
				URL:   baseURL + href,
			})
		}
	})

	return categories, nil
}

func ShowCategoryMenu(categories []Category) Category {
	var filtered []Category = categories

	for {
		fmt.Println("\nEnter a category, or ENTER to show all:")
		var input string
		fmt.Print("> ")
		fmt.Scanln(&input)

		filtered = nil
		for _, cat := range categories {
			if strings.Contains(strings.ToLower(cat.Title), strings.ToLower(input)) {
				filtered = append(filtered, cat)
			}
		}

		if len(filtered) == 0 {
			fmt.Println("No results. Try again.")
			continue
		}

		fmt.Println("\nCategories:")
		for i, cat := range filtered {
			fmt.Printf("[%d] %s\n", i, cat.Title)
		}

		fmt.Print("Select a category number: ")
		var index int
		_, err := fmt.Scanln(&index)
		if err != nil || index < 0 || index >= len(filtered) {
			fmt.Println("Invalid choice!")
			continue
		}

		return filtered[index]
	}
}
