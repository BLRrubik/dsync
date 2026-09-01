package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		fmt.Println(file)
	}
}

func ListFiles(rootPath string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, strings.TrimPrefix(path, rootPath+"/"))
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	return files, nil
}
