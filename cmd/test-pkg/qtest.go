package test_pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"
)

const releaseMBID = "d6b52521-0dfa-390f-970f-790174c22752"
const ua = "mb-track-rename/0.1 (contact: decbrks@pm.me)"

func main() {
	// Basic MBID sanity check (UUID format).
	mbidRe := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !mbidRe.MatchString(releaseMBID) {
		log.Fatalf("invalid MBID format: %s", releaseMBID)
	}

	client := &http.Client{Timeout: 15 * time.Second}

	u := fmt.Sprintf("https://musicbrainz.org/ws/2/release/%s?inc=media+recordings&fmt=json", releaseMBID)
	log.Printf("requesting: %s", u)

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

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		log.Fatalf("http %d: %s", res.StatusCode, string(b))
	}

	var v any
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		log.Fatal(err)
	}
	log.Printf("MusicBrainz response OK")
}
