package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const ua = "mb-track-rename/0.1 (contact: decbrks@pm.me)"

type artistCreditName struct {
	Name string `json:"name"`
}

type rgSearchResp struct {
	Count         int `json:"count"`
	ReleaseGroups []struct {
		ID           string             `json:"id"`
		Title        string             `json:"title"`
		PrimaryType  string             `json:"primary-type"`
		FirstRelease string             `json:"first-release-date"`
		ArtistCredit []artistCreditName `json:"artist-credit"`
	} `json:"release-groups"`
}

type releaseInRG struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Country        string `json:"country"`
	Date           string `json:"date"`
	Title          string `json:"title"`
	TrackCount     int    `json:"track-count"`
	Packaging      string `json:"packaging"`
	Disambiguation string `json:"disambiguation"`
}

type releasesInRGResp struct {
	Releases []releaseInRG `json:"releases"`
}

func doGET(client *http.Client, u string, out any) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		log.Fatalf("http %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		log.Fatal(err)
	}
}

func artistString(ac []artistCreditName) string {
	var parts []string
	for _, a := range ac {
		if a.Name != "" {
			parts = append(parts, a.Name)
		}
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, ", ")
}

func pickBestReleaseMBID(client *http.Client, rgid string) (string, error) {
	u := fmt.Sprintf("https://musicbrainz.org/ws/2/release?query=rgid:%s&fmt=json&limit=100", url.QueryEscape(rgid))
	var rr releasesInRGResp
	doGET(client, u, &rr)

	if len(rr.Releases) == 0 {
		return "", fmt.Errorf("no releases found for rgid %s", rgid)
	}

	score := func(r releaseInRG) int {
		s := 0
		if r.Status == "Official" {
			s += 1000
		}
		if r.Country == "US" {
			s += 50
		}
		if r.Date != "" {
			s += 10
		}
		if r.Disambiguation == "" {
			s += 5
		}
		return s
	}

	best := rr.Releases[0]
	bestS := score(best)
	for _, r := range rr.Releases[1:] {
		if s := score(r); s > bestS {
			best = r
			bestS = s
		}
	}
	return best.ID, nil
}

func main() {
	artist := flag.String("artist", "", "artist name")
	album := flag.String("album", "", "album name")
	limit := flag.Int("limit", 10, "number of results to print")
	flag.Parse()

	if strings.TrimSpace(*artist) == "" || strings.TrimSpace(*album) == "" {
		log.Fatal("usage: go run ./cmd/mbfind -artist <name> -album <title>")
	}

	client := &http.Client{Timeout: 20 * time.Second}

	q := fmt.Sprintf(`artist:%q AND releasegroup:%q`, strings.TrimSpace(*artist), strings.TrimSpace(*album))
	u := fmt.Sprintf(
		"https://musicbrainz.org/ws/2/release-group?query=%s&fmt=json&limit=%d",
		url.QueryEscape(q),
		*limit,
	)

	var sr rgSearchResp
	doGET(client, u, &sr)

	if len(sr.ReleaseGroups) == 0 {
		log.Fatal("no release-groups found")
	}

	type row struct {
		rgid        string
		title       string
		artist      string
		typ         string
		firstDate   string
		releaseMBID string
	}

	var rows []row
	for _, rg := range sr.ReleaseGroups {
		mbid := ""
		if rg.ID != "" {
			time.Sleep(1 * time.Second)
			id, err := pickBestReleaseMBID(client, rg.ID)
			if err == nil {
				mbid = id
			}
		}

		rows = append(rows, row{
			rgid:        rg.ID,
			title:       rg.Title,
			artist:      artistString(rg.ArtistCredit),
			typ:         rg.PrimaryType,
			firstDate:   rg.FirstRelease,
			releaseMBID: mbid,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].typ != rows[j].typ {
			if rows[i].typ == "Album" {
				return true
			}
			if rows[j].typ == "Album" {
				return false
			}
		}
		if rows[i].firstDate != rows[j].firstDate {
			return rows[i].firstDate > rows[j].firstDate
		}
		return rows[i].title < rows[j].title
	})

	for i, r := range rows {
		fmt.Printf("%02d) %s \u2014 %s\n", i+1, r.artist, r.title)
		fmt.Printf("    type: %s  first: %s\n", r.typ, r.firstDate)
		fmt.Printf("    rgid: %s\n", r.rgid)
		if r.releaseMBID != "" {
			fmt.Printf("    mbid: %s\n", r.releaseMBID)
		} else {
			fmt.Printf("    mbid: (not found)\n")
		}
	}
}
