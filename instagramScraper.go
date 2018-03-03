package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly"
)

// Define your target instagram account here
const instagramAccount string = `kaimook.bnk48official`

// Ajax paging URL
const nextPageURLTemplate string = `https://www.instagram.com/graphql/query/?query_hash=472f257a40c653c64c666ce877d59d2b&variables={"id":"%s","first":12,"after":"%s"}`

// Structure - paging cursor
type pageInfo struct {
	EndCursor string `json:"end_cursor"`
	NextPage  bool   `json:"has_next_page"`
}

// Structure - first entry
type entryNode struct {
	EntryData struct {
		ProfilePage []struct {
			User struct {
				ID    string `json:"id"`
				Media struct {
					Nodes []struct {
						ImageURL     string `json:"display_src"`
						ThumbnailURL string `json:"thumbnail_src"`
						IsVideo      bool   `json:"is_video"`
						Date         int    `json:"date"`
						Dimensions   struct {
							Width  int `json:"width"`
							Height int `json:"height"`
						}
						Likes struct {
							Count int `json:"count"`
						} `json:"likes"`
					}
					PageInfo pageInfo `json:"page_info"`
				} `json:"media"`
			} `json:"user"`
		} `json:"ProfilePage"`
	} `json:"entry_data"`
}

// Structure - next entry
type entryEdgeNode struct {
	Data struct {
		User struct {
			Container struct {
				PageInfo pageInfo `json:"page_info"`
				Edges    []struct {
					Node struct {
						ImageURL     string `json:"display_url"`
						ThumbnailURL string `json:"thumbnail_src"`
						IsVideo      bool   `json:"is_video"`
						Date         int    `json:"taken_at_timestamp"`
						Dimensions   struct {
							Width  int `json:"width"`
							Height int `json:"height"`
						}
						Likes struct {
							Count int `json:"count"`
						} `json:"edge_media_preview_like"`
					}
				} `json:"edges"`
			} `json:"edge_owner_to_timeline_media"`
		}
	} `json:"data"`
}

// Statistic Data - for csv file
var statData = [][]string{{"Image Name", "Likes"}}

func main() {

	var actualUserID string
	outputDir := fmt.Sprintf("./instagram_scrapped_image/")

	c := colly.NewCollector(
		colly.CacheDir("./_instagram_cache/"),
	)

	c.OnHTML("body > script:first-of-type", func(e *colly.HTMLElement) {
		jsonData := e.Text[strings.Index(e.Text, "{") : len(e.Text)-1]
		data := entryNode{}
		err := json.Unmarshal([]byte(jsonData), &data)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("saving output to ", outputDir)
		os.MkdirAll(outputDir, os.ModePerm)
		page := data.EntryData.ProfilePage[0]
		actualUserID = page.User.ID
		for _, obj := range page.User.Media.Nodes {
			if obj.IsVideo {
				continue
			}
			newStat := []string{filepath.Base(obj.ImageURL), fmt.Sprintf("%v", obj.Likes.Count)}
			statData = append(statData, newStat)
			c.Visit(obj.ImageURL)
		}
		if page.User.Media.PageInfo.NextPage {
			c.Visit(fmt.Sprintf(nextPageURLTemplate, actualUserID, page.User.Media.PageInfo.EndCursor))
		}
	})

	c.OnResponse(func(r *colly.Response) {
		if strings.Index(r.Headers.Get("Content-Type"), "image") > -1 {
			log.Println("Saving Image: " + r.FileName())
			r.Save(outputDir + r.FileName())
			return
		}

		if strings.Index(r.Headers.Get("Content-Type"), "json") == -1 {
			return
		}

		data := entryEdgeNode{}
		err := json.Unmarshal(r.Body, &data)
		if err != nil {
			log.Fatal(err)
		}

		for _, obj := range data.Data.User.Container.Edges {
			if obj.Node.IsVideo {
				continue
			}
			newStat := []string{filepath.Base(obj.Node.ImageURL), fmt.Sprintf("%v", obj.Node.Likes.Count)}
			statData = append(statData, newStat)
			c.Visit(obj.Node.ImageURL)
		}
		if data.Data.User.Container.PageInfo.NextPage {
			c.Visit(fmt.Sprintf(nextPageURLTemplate, actualUserID, data.Data.User.Container.PageInfo.EndCursor))
		} else {
			log.Println("Done Scraping")

			// Save CSV File -----------------------------
			file, err := os.Create("result.csv")
			checkError("Cannot create file", err)
			defer file.Close()

			writer := csv.NewWriter(file)
			defer writer.Flush()

			for _, value := range statData {
				err := writer.Write(value)
				checkError("Cannot write to file", err)
			}
		}
	})

	c.Visit("https://instagram.com/" + instagramAccount)
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}
