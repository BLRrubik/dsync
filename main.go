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
	if len(os.Args) < 3 {
		fmt.Println("invalid args")
	}

	rootPath := os.Args[1]
	dstPath := os.Args[2]

	rootTable := prepareFileTable(rootPath)
	fmt.Println("rootPath:", rootPath)
	for file, hash := range rootTable {
		fmt.Println(file, hash)
	}
	dstTable := prepareFileTable(dstPath)
	fmt.Println("dstPath:", dstPath)
	for file, hash := range dstTable {
		fmt.Println(file, hash)
	}

	toCopy, toUpdate, toDelete := CompareScans(rootTable, dstTable)

	fmt.Println("toCopy:", toCopy)
	fmt.Println("toUpdate:", toUpdate)
	fmt.Println("toDelete:", toDelete)
}

func ListFiles(rootPath string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}

			files = append(files, relPath)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	return files, nil
}

func prepareFileTable(path string) map[string]string {
	files, err := ListFiles(path)
	if err != nil {
		log.Fatal(err)
	}

	filesTable := make(map[string]string)

	for data := range processHash(path, files) {
		filesTable[data.Path] = data.Hash
	}

	return filesTable
}

func processHash(rootPath string, files []string) chan FileHash {
	var (
		tasks   = make(chan string)
		results = make(chan FileHash)
		wg      = &sync.WaitGroup{}
	)

	go func() {
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for path := range tasks {
					hash, err := HashFile(rootPath + "/" + path)
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
	}()

	return results
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

func CompareScans(source, dest map[string]string) (toCopy, toUpdate, toDelete []string) {
	checkedFiles := make(map[string]struct{})

	for file, hash := range source {
		if destHash, ok := dest[file]; !ok {
			toCopy = append(toCopy, file)
		} else if hash != destHash {
			toUpdate = append(toUpdate, file)
		}

		checkedFiles[file] = struct{}{}
	}

	for file := range dest {
		if _, ok := checkedFiles[file]; ok {
			continue
		}

		toDelete = append(toDelete, file)
	}

	return toCopy, toUpdate, toDelete
}
