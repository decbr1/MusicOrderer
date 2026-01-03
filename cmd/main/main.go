package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ua = "mb-track-rename/0.1 (contact: decbrks@pm.me)"

type releaseSearchResp struct {
	Releases []struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Country    string `json:"country"`
		Date       string `json:"date"`
		TrackCount int    `json:"track-count"`
		Title      string `json:"title"`
	} `json:"releases"`
}

type releaseLookupResp struct {
	Title string `json:"title"`
	Media []struct {
		Position int `json:"position"`
		Tracks   []struct {
			Position     int    `json:"position"`
			Title        string `json:"title"`
			ArtistCredit []struct {
				Name string `json:"name"`
			} `json:"artist-credit"`
		} `json:"tracks"`
	} `json:"media"`
}

type track struct {
	medium int
	pos    int
	title  string
	artist string
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	repl := strings.NewReplacer( // remove windows-illegal and generally annoying chars
		"/", "／", "\\", "＼",
		":", " -", "*", "", "?", "", "\"",
		"<", "(", ">", ")", "|", "-", "'",
	)
	return repl.Replace(s)
}

// normalize for fuzzy matching:  lowercase, remove non-alphanumeric
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, s)
	return s
}

func doGET(client *http.Client, url string, out any) {
	req, err := http.NewRequest("GET", url, nil)
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

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		log.Fatal(err)
	}
}

func pickReleaseIDFromRGID(client *http.Client, rgid string) (string, error) {
	u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/?query=rgid:%s&fmt=json&limit=100", rgid)
	//u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/%s?inc=media+recordings+artist-credits&fmt=json", rgid)

	var sr releaseSearchResp
	doGET(client, u, &sr)

	if len(sr.Releases) == 0 {
		return "", fmt.Errorf("no releases found for rgid %s", rgid)
	}

	bestIdx := 0
	bestScore := -1
	for i, r := range sr.Releases {
		s := 0
		if r.Status == "Official" {
			s += 1000
		}
		if r.TrackCount == 14 {
			s += 200
		}
		if r.Country == "US" {
			s += 50
		}
		if r.Date != "" {
			s += 10
		}
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}

	return sr.Releases[bestIdx].ID, nil
}

func lookupTracks(client *http.Client, releaseID string) ([]track, string, error) {
	//u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/%s?inc=media+recordings&fmt=json", releaseID)
	u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/%s?inc=media+recordings+artist-credits&fmt=json", releaseID)

	var rl releaseLookupResp
	doGET(client, u, &rl)

	var tracks []track
	for _, m := range rl.Media {
		for _, t := range m.Tracks {
			artist := "Unknown"
			if len(t.ArtistCredit) > 0 {
				artist = t.ArtistCredit[0].Name
			}
			tracks = append(tracks, track{
				medium: m.Position,
				pos:    t.Position,
				title:  t.Title,
				artist: artist,
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
		return nil, rl.Title, fmt.Errorf("no tracks on release %s", releaseID)
	}
	return tracks, rl.Title, nil
}

func main() {
	dir := flag.String("dir", ".", "directory containing audio files")
	mbid := flag.String("mbid", "", "MusicBrainz RELEASE MBID (best option)")
	rgid := flag.String("rgid", "", "MusicBrainz RELEASE-GROUP MBID (script will pick a release)")
	artistInFilename := flag.Bool("artist-in-filename", false, "include artist name in output filenames")

	flag.Parse()

	if *mbid == "" && *rgid == "" {
		log.Fatal(`
		incorrect usage. replace (command) with one of the following: { go run . || ./main || main.exe }
			usage: (command) -mbid <release-mbid> -dir <folder>
			or:    (command) -rgid <release-group-mbid> -dir <folder>`)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	releaseID := *mbid
	if releaseID == "" {
		id, err := pickReleaseIDFromRGID(client, *rgid)
		if err != nil {
			log.Fatal(err)
		}
		releaseID = id
	}

	time.Sleep(1 * time.Second)

	tracks, title, err := lookupTracks(client, releaseID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Using release %s: %s\n", releaseID, title)

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatal(err)
	}

	// build list of audio files (skip dirs, hidden files)
	var files []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		files = append(files, e.Name())
	}

	if len(files) != len(tracks) {
		fmt.Printf("Warning: %d files but %d tracks.  Proceeding anyway.\n", len(files), len(tracks))
	}

	used := make(map[string]bool)

	for i, tr := range tracks {
		n := i + 1
		normTrack := normalize(tr.title)

		// find best match by title similarity
		bestFile := ""
		bestScore := -1
		for _, f := range files {
			if used[f] {
				continue
			}
			// strip extension and normalize
			base := strings.TrimSuffix(f, filepath.Ext(f))
			normFile := normalize(base)

			// simple substring match score
			score := 0
			if strings.Contains(normFile, normTrack) {
				score = 100
			} else if strings.Contains(normTrack, normFile) {
				score = 50
			} else {
				// count common substrings (crude)
				for _, word := range strings.Fields(normTrack) {
					if len(word) > 2 && strings.Contains(normFile, word) {
						score += 10
					}
				}
			}

			if score > bestScore {
				bestScore = score
				bestFile = f
			}
		}

		if bestFile == "" || bestScore == 0 {
			fmt.Printf("warning: no match for track %d %q\n", n, tr.title)
			continue
		}

		used[bestFile] = true

		ext := filepath.Ext(bestFile)
		var newName string
		if *artistInFilename {
			newName = fmt.Sprintf("%02d %s - %s%s", n, sanitizeFilename(tr.artist), sanitizeFilename(tr.title), ext)
		} else {
			newName = fmt.Sprintf("%02d %s%s", n, sanitizeFilename(tr.title), ext)
		}

		oldPath := filepath.Join(*dir, bestFile)
		newPath := filepath.Join(*dir, newName)

		if oldPath == newPath {
			continue
		}

		fmt.Printf("%02d:  %q -> %q\n", n, bestFile, newName)
		if err := os.Rename(oldPath, newPath); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}
