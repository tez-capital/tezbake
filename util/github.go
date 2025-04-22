package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tez-capital/tezbake/constants"
)

type release struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

func (rel *release) FindAsset(want string) (url string, size int64, err error) {
	for _, a := range rel.Assets {
		if a.Name == want {
			return a.BrowserDownloadURL, a.Size, nil
		}
	}
	return "", 0, fmt.Errorf("asset %q not present in release", want)
}

func FetchGithubRelease(ctx context.Context, wantPrerelease bool, tag string) (*release, error) {
	base := "https://api.github.com/repos/" + constants.TezbakeRepository + "/releases"
	var url string
	switch {
	case wantPrerelease && tag != "" && tag != "latest":
		url = base + "/tags/" + tag
	case wantPrerelease:
		url = base + "?per_page=10" // search among the 10 most recent
	default:
		url = base + "/latest"
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound && wantPrerelease && tag != "" {
		return nil, fmt.Errorf("tag %q not found", tag)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", res.Status)
	}

	if wantPrerelease && (tag == "" || tag == "latest") {
		// We got a slice of releases; pick first with prerelease=true
		var all []release
		if err := json.NewDecoder(res.Body).Decode(&all); err != nil {
			return nil, err
		}
		for _, r := range all {
			if r.Prerelease {
				return &r, nil
			}
		}
		return nil, errors.New("no prerelease found in recent history")
	}

	// Single release object
	var r release
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}
