package storage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const testLibUrl = "https://raw.githubusercontent.com/MikeMirzayanov/testlib/master/testlib.h"
const testLibCachePath = "cache/testlib.h"

func GetTestlib() ([]byte, error) {
	if err := ensureTestlibExistsInCache(); err != nil {
		return nil, fmt.Errorf("failed to ensure testlib exists in cache: %w", err)
	}

	testlibBytes, err := os.ReadFile(testLibCachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read testlib from cache: %w", err)
	}

	return testlibBytes, nil
}

func ensureTestlibExistsInCache() error {
	// Check if the file already exists
	if _, err := os.Stat(testLibCachePath); err == nil {
		log.Println("Testlib already exists in cache")
		return nil
	} else if !os.IsNotExist(err) {
		log.Println("Error while checking if testlib exists in cache")
		return err // Return here on other errors besides "not exists"
	}

	// Create the file
	out, err := os.Create(testLibCachePath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	resp, err := http.Get(testLibUrl)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
