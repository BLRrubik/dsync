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
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("invalid args")
	}

	rootPath := os.Args[1]

	files, err := ListFiles(rootPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		hash, err := HashFile(rootPath + "/" + file)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(file, hash)
	}
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
				return fmt.Errorf("could not determine relative path: %w", err)
			}

			files = append(files, relPath)
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
