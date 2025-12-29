package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const ua = "mbid-finder/0.1 (contact: you@example.com)"

type searchResp struct {
	Releases []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Date  string `json:"date"`
	} `json:"releases"`
}

func main() {
	artist := "Radiohead"
	album := "Kid A"

	q := fmt.Sprintf(`release:%q AND artist:%q`, album, artist)
	u := "https://musicbrainz.org/ws/2/release/?query=" + url.QueryEscape(q) + "&fmt=json&limit=5"

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", ua)

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)

	var r searchResp
	_ = json.Unmarshal(b, &r)

	for _, rel := range r.Releases {
		fmt.Println(rel.ID, "-", rel.Title, rel.Date)
	}
}
