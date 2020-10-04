package locache

import (
	"fmt"
	"os"
	"strings"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func findFilesByExt(directory string, extension string) []os.FileInfo {
	println("looking for files in:", directory, " ext: ", extension)
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
