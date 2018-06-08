package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// CreateFileWithData creates a new file at the given path and writes
// the provided bytes
func CreateFileWithData(fileName string, data []byte) error {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// ReadNewlineSeparatedList reads data from the reader interface until
// reaching EOF and returns a slice with data from each line
func ReadNewlineSeparatedList(rd io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(rd)
	var into []string
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		if token == "" {
			continue
		}
		into = append(into, token)
	}
	return into, scanner.Err()
}
