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

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
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
	dstTable := prepareFileTable(dstPath)

	toCopy, toUpdate, toDelete := CompareScans(rootTable, dstTable)

	if err := Sync(rootPath, dstPath, toCopy, toUpdate, toDelete); err != nil {
		log.Fatal(err)
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

func Sync(
	rootPath string,
	dstPath string,
	toCopy []string,
	toUpdate []string,
	toDelete []string,
) error {
	total := len(toCopy) + len(toUpdate) + len(toDelete)

	fmt.Println("Синхронизация...")
	bar := progressbar.Default(int64(total))

	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	type logLine struct {
		c    *color.Color
		text string
	}
	logs := make([]logLine, 0, total)

	for _, file := range toCopy {
		if err := CopyFile(filepath.Join(rootPath, file), filepath.Join(dstPath, file)); err != nil {
			return err
		}
		logs = append(logs, logLine{green, fmt.Sprintf("[КОПИРОВАНИЕ] %s", file)})
		bar.Add(1)
	}

	for _, file := range toUpdate {
		if err := CopyFile(filepath.Join(rootPath, file), filepath.Join(dstPath, file)); err != nil {
			return err
		}
		logs = append(logs, logLine{yellow, fmt.Sprintf("[ОБНОВЛЕНИЕ] %s", file)})
		bar.Add(1)
	}

	for _, file := range toDelete {
		if err := DeleteFile(filepath.Join(dstPath, file)); err != nil {
			return err
		}
		logs = append(logs, logLine{red, fmt.Sprintf("[УДАЛЕНИЕ] %s", file)})
		bar.Add(1)
	}

	fmt.Println()
	for _, l := range logs {
		l.c.Println(l.text)
	}

	fmt.Println()
	fmt.Println("Синхронизация завершена!")

	return nil
}

func CopyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	sFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer sFile.Close()

	dstFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, sFile); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	return nil
}

func DeleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("error deleting file: %w", err)
	}

	return nil
}
