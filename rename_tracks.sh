#!/bin/bash

declare -a NEW_ORDER=(
  "Super Bowl LIX Halftime Show - Live.mp3"
  "wacced out murals.mp3"
  "squabble up.mp3"
  "luther (with sza).mp3"
  "man at the garden.mp3"
  "hey now (feat. dody6).mp3"
  "reincarnated.mp3"
  "tv off (feat. lefty gunplay).mp3"
  "dodger blue (feat. wallie the sensei, siete7x, roddy ricch).mp3"
  "peekaboo (feat.azchike).mp3"
  "heart pt. 6.mp3"
  "gnx (feat. hitta j3, youngthreat, peysoh).mp3"
  "gloria (with sza).mp3"
)

i=1
for filename in "${NEW_ORDER[@]}"; do
  num=$(printf "%02d" "$i")
  newname="${num}. ${filename}"
  if [[ -f "$filename" ]]; then
    echo "Renaming '$filename' -> '$newname'"
    mv -- "$filename" "$newname"
  else
    echo "Warning: '$filename' not found!"
  fi
  ((i++))
done
