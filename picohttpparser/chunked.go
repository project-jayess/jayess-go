package picohttpparser

import (
	"fmt"
	"strconv"
	"strings"
)

func DecodeChunked(input string) (string, error) {
	var out strings.Builder
	rest := input
	for {
		lineEnd := strings.Index(rest, "\r\n")
		if lineEnd < 0 {
			return "", fmt.Errorf("%s: incomplete chunk size", IncompleteData)
		}
		sizeText := strings.TrimSpace(rest[:lineEnd])
		if extension, _, ok := strings.Cut(sizeText, ";"); ok {
			sizeText = extension
		}
		size, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil || size < 0 {
			return "", fmt.Errorf("%s: malformed chunk size", MalformedInput)
		}
		rest = rest[lineEnd+2:]
		if size == 0 {
			return out.String(), nil
		}
		if int64(len(rest)) < size+2 {
			return "", fmt.Errorf("%s: incomplete chunk data", IncompleteData)
		}
		out.WriteString(rest[:size])
		if rest[size:size+2] != "\r\n" {
			return "", fmt.Errorf("%s: malformed chunk terminator", MalformedInput)
		}
		rest = rest[size+2:]
	}
}
