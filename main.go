package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const releaseMBID = ""
const ua = "mb-track-rename/0.1 (contact: decbrks@pm.me)"

type mbRelease struct {
	Media []struct {
		Position int `json:"position"`
		Tracks   []struct {
			Position int    `json:"position"`
			Number   string `json:"number"`
			Title    string `json:"title"`
		} `json:"tracks"`
	} `json:"media"`
}

type track struct {
	medium int
	pos    int
	title  string
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	// windows-illegal + generally annoying characters
	repl := strings.NewReplacer(
		"/", "／", "\\",
		":", " -", "*", "", "-",
		"?", "", "\"", "'", "|",
		"<", "(", ">", ")", "＼",
	)
	return repl.Replace(s)
}

func main() {
	client := &http.Client{Timeout: 15 * time.Second}

	u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/%s?inc=media+recordings&fmt=json", releaseMBID)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		log.Fatalf("http %d: %s", res.StatusCode, string(b))
	}

	var r mbRelease
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatal(err)
	}

	var tracks []track
	for _, m := range r.Media {
		for _, t := range m.Tracks {
			tracks = append(tracks, track{
				medium: m.Position,
				pos:    t.Position,
				title:  t.Title,
			})
		}
	}

	sort.Slice(tracks, func(i, j int) bool {
		if tracks[i].medium != tracks[j].medium {
			return tracks[i].medium < tracks[j].medium
		}
		return tracks[i].pos < tracks[j].pos
	})

	if len(tracks) == 0 {
		log.Fatal("no tracks returned from MusicBrainz (wrong MBID?)")
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	reNum := regexp.MustCompile(`^(\d{1,3})\b`)

	existingByNum := map[int]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		m := reNum.FindStringSubmatch(name)
		if len(m) != 2 {
			continue
		}
		var n int
		_, _ = fmt.Sscanf(m[1], "%d", &n)
		if n > 0 {
			// first one wins
			if _, ok := existingByNum[n]; !ok {
				existingByNum[n] = name
			}
		}
	}

	for i, tr := range tracks {
		n := i + 1
		old, ok := existingByNum[n]
		if !ok {
			fmt.Printf("warning: no local file starting with %02d\n", n)
			continue
		}

		ext := filepath.Ext(old)
		newName := fmt.Sprintf("%02d. %s%s", n, sanitizeFilename(tr.title), ext)

		if old == newName {
			continue
		}

		fmt.Printf("Renaming %q -> %q\n", old, newName)
		if err := os.Rename(old, newName); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}
