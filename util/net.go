package util

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func PrintDownloadPercent(done chan int64, path string, total int64) {

	var stop bool = false

	for {
		select {
		case <-done:
			stop = true
		default:

			file, err := os.Open(path)
			if err != nil {
				goto CONTINUE
			}

			fi, err := file.Stat()
			if err != nil {
				goto CONTINUE
			}

			size := fi.Size()

			if size == 0 {
				size = 1
			}

			var percent float64 = float64(size) / float64(total) * 100

			fmt.Printf("%.0f%%...", percent)
		}

		if stop {
			break
		}
	CONTINUE:
		time.Sleep(time.Second * 5)
	}
}

func DownloadFile(url string, dest string, progress bool) error {
	// Get the data

	headResp, err := http.Head(url)
	if err != nil {
		return err
	}
	defer headResp.Body.Close()

	done := make(chan int64)
	if progress {
		size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
		if err != nil {
			return err
		}
		go PrintDownloadPercent(done, dest, int64(size))
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	n, err := io.Copy(out, resp.Body)
	if progress {
		fmt.Println()
		done <- n
	}

	return err
}

func IsValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
