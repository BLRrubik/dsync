package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type FileHash struct {
	Path string
	Hash string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("invalid args")
	}

	rootPath := os.Args[1]

	files, err := ListFiles(rootPath)
	if err != nil {
		log.Fatal(err)
	}

	var (
		tasks   = make(chan string)
		results = make(chan FileHash)
		wg      = &sync.WaitGroup{}
	)

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for path := range tasks {
				hash, err := HashFile(path)
				if err != nil {
					continue
				}

				results <- FileHash{Path: path, Hash: hash}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for _, file := range files {
		tasks <- file
	}

	close(tasks)

	for data := range results {
		fmt.Println(data.Path, data.Hash)
	}
}

func ListFiles(rootPath string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	return files, nil
}

func HashFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("error hashing file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
