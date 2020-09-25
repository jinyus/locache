package locache

import (
	"fmt"
	"os"
	"strings"
)

func findFilesByExt(directory string, extension string) []os.FileInfo {
	f, err := os.Open(directory)
	if err != nil {
		fmt.Printf("could not open directory: %v\n", err)
		return nil
	}
	defer f.Close()

	entries, err := f.Readdir(0) // 0 => no limit; read all entries
	if err != nil {
		fmt.Printf("error reading dir: %v\n", err)
		// Don't return: Readdir may return partial results.
	}

	var found []os.FileInfo
	for _, file := range entries {
		if strings.HasSuffix(file.Name(), extension) && !file.IsDir() {
			found = append(found, file)
		}
	}
	return found
}