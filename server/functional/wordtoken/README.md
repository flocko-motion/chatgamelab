# words_de.txt

2048 German words, one per line: `a–z` only, 4–10 letters, unique four-letter prefixes.

Source: [dys2p/wordlists-de](https://github.com/dys2p/wordlists-de) `de-2048-v1.txt`,
multi-licensed Unlicense / CC0 / BSD-3; used here under CC0.

Changes against the source: 34 words unfit for schools removed (violence, alcohol,
crime, illness, body shaming) and replaced by words from the same project's
`de-7776-v1.txt` that keep the prefixes unique.

Removed: alkohol angriff attacke bedroht beil bier blut bombe brust bunker dick dieb
dolch fett ganove gauner gewalt gier gift grab gruft habgier illegal kneipe krebs
kriegen peitsche pille rache razzia verrat virus wein zocken

Added: alphabet amateur bison dackel diamant dromedar duett eislauf ferkel gockel
hermelin iltis kaiman kondor kormoran labrador leguan lerche mammut marzipan neumond
panorama pizza radeln rodeln seeadler skilift tundra urwald viadukt wirsing xylofon
ziesel zylinder

`wordtoken_test.go` enforces the format rules on every change.
