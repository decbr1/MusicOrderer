# Music Reorder, rer.

A Go tool for automatically renaming music files to match official track listings from MusicBrainz.

## Overview

This project consists of two complementary tools:
1. **`main`** - Renames audio files in a directory based on MusicBrainz metadata
2. **`mbfind`** - Searches MusicBrainz to find release IDs for an artist/album

## Usage

the files in the `releases/` directory probably wont work as of right now. oh well!<br>
use `go run .` from a git clone instead for now

### Finding Release IDs

Use `mbfind` to search for an album and get its MusicBrainz IDs:

```bash
go run ./mbfind -artist "Michael Jackson" -album "Dangerous"
```

This outputs a ranked list of matching releases with their:
- Release Group ID (rgid)
- Release ID (mbid)
- Type (Album, Single, etc.)
- First release date

### Renaming Files

Once you have a release ID, use the main tool to rename your files:

```bash
# Using a release ID (most accurate)
go run . -mbid <release-mbid> -dir "/path/to/music/folder"

# Using a release group ID (tool picks best release)
go run . -rgid <release-group-mbid> -dir "/path/to/music/folder"
```

## Example

```bash
$ go run . -rgid d6b52521-0dfa-390f-970f-790174c22752 -dir "/path/to/Michael Jackson - Dangerous/"

Using release ae5efacd-f75f-432a-9f22-b35d3169d21f: Dangerous
01:  "Michael Jackson - Jam.flac" -> "01. Jam.flac"
02:  "Michael Jackson - Why You Wanna Trip On Me.flac" -> "02. Why You Wanna Trip on Me.flac"
03:  "Michael Jackson - In The Closet.flac" -> "03. In the Closet.flac"
...
14:  "Michael Jackson - Dangerous.flac" -> "14. Dangerous.flac"
```

## Features

- **Fuzzy matching** - Intelligently matches existing filenames to track titles
- **Safe renaming** - Sanitizes filenames to remove problematic characters
- **Multi-disc support** - Handles albums spanning multiple discs
- **Preserves extensions** - Keeps original file extensions (.flac, .mp3, etc.)
- **Smart release selection** - When using rgid, prioritizes official releases from preferred regions

## Flags

### main
- `-mbid` - MusicBrainz Release MBID (recommended for accuracy)
- `-rgid` - MusicBrainz Release Group MBID (script selects best release)
- `-dir` - Directory containing audio files (default: current directory)
- `--artist-in-filename` - Add artist name prefix to filenames

### mbfind
- `-artist` - Artist name (required)
- `-album` - Album title (required)
- `-limit` - Number of results to display (default: 10)


## How It Works

1. Queries the MusicBrainz API for official track listings
2. Scans your directory for audio files
3. Matches files to tracks using normalized title comparison
4. Renames files to format: `## Track Title.ext`

## Requirements

- Go 1.16 or higher
- Internet connection (for MusicBrainz API access)

## Notes

- The tool skips hidden files (starting with `.`)
- If file count doesn't match track count, it proceeds with a warning
- Original file extensions are preserved
- Windows-incompatible characters are replaced with safe alternatives

## License
GNU GPLv3<br>
Contact: decbrks@pm.me