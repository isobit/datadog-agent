package rungo

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"regexp"
)

// This is the main Golang version download page.
// It contains a bunch of HTML, which has to be grepped through,
// but this is the same approach that Gimme uses:
// https://github.com/travis-ci/gimme
// https://github.com/travis-ci/gimme/blob/31ad563474d6ee1dabdabe1d1d2bbdeb6444fd92/gimme#L542
const goVersionListURL string = "https://golang.org/dl"

var goVersionRegex *regexp.Regexp = regexp.MustCompile(`go([A-Za-z0-9.]*)\.src`)

// Gets a list of all current Go versions by downloading the Golang download page
// and scanning it for Go versions.
// Includes beta and RC versions, as well as normal point releases.
// See https://golang.org/dl (all versions are listed under "Archived versions")
func ListGoVersions(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", goVersionListURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing GET request to %s: %w", goVersionListURL, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request to %s: %w", goVersionListURL, err)
	}

	defer resp.Body.Close()

	// Read in the body line by line, and use the regex on each line.
	// We probably could read in the entire page to a buffer;
	// as of 2021-11-15, it's about 1 MiB.
	// However, there are a multitude of line breaks
	// at natural locations, so it's just easier to do that.
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	allVersionsSet := make(map[string]struct{})
	for scanner.Scan() {
		matches := goVersionRegex.FindAllStringSubmatch(scanner.Text(), -1)
		for _, groups := range matches {
			version := groups[1]
			allVersionsSet[version] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning response from %s: %w", goVersionListURL, err)
	}

	i := 0
	allVersions := make([]string, len(allVersionsSet))
	for v := range allVersionsSet {
		allVersions[i] = v
		i += 1
	}

	return allVersions, nil
}
