package constants

import (
	"bufio"
	"embed"
	"strings"
	"sync"
)

//go:embed profanity-words.txt
var profanityFile embed.FS

var (
	profanityWords map[string]struct{}
	profanityOnce  sync.Once
)

func loadProfanityWords() {
	profanityWords = make(map[string]struct{})

	f, err := profanityFile.Open("profanity-words.txt")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			profanityWords[strings.ToLower(word)] = struct{}{}
		}
	}
}

// IsProfane checks if the given name contains a profane word.
// It checks the full name (case-insensitive) and each word in the name individually.
func IsProfane(name string) bool {
	profanityOnce.Do(loadProfanityWords)

	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}

	// Check full name
	if _, found := profanityWords[lower]; found {
		return true
	}

	// Check each word individually
	for _, word := range strings.Fields(lower) {
		if _, found := profanityWords[word]; found {
			return true
		}
	}

	return false
}
