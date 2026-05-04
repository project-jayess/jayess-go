package picohttpparser

import (
	"fmt"
	"strconv"
	"strings"
)

type Header struct {
	Name  string
	Value string
}

type Request struct {
	Method  string
	Path    string
	Version string
	Headers []Header
}

type Response struct {
	Version    string
	StatusCode int
	Reason     string
	Headers    []Header
}

func ParseRequest(input string) (Request, error) {
	head, err := headerBlock(input)
	if err != nil {
		return Request{}, err
	}
	lines := splitLines(head)
	if len(lines) == 0 {
		return Request{}, fmt.Errorf("%s: missing request line", MalformedInput)
	}
	parts := strings.Fields(lines[0])
	if len(parts) != 3 || !strings.HasPrefix(parts[2], "HTTP/") {
		return Request{}, fmt.Errorf("%s: malformed request line", MalformedInput)
	}
	headers, err := parseHeaders(lines[1:])
	if err != nil {
		return Request{}, err
	}
	return Request{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: headers,
	}, nil
}

func ParseResponse(input string) (Response, error) {
	head, err := headerBlock(input)
	if err != nil {
		return Response{}, err
	}
	lines := splitLines(head)
	if len(lines) == 0 {
		return Response{}, fmt.Errorf("%s: missing response line", MalformedInput)
	}
	parts := strings.SplitN(lines[0], " ", 3)
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "HTTP/") {
		return Response{}, fmt.Errorf("%s: malformed response line", MalformedInput)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return Response{}, fmt.Errorf("%s: malformed response status", MalformedInput)
	}
	reason := ""
	if len(parts) == 3 {
		reason = parts[2]
	}
	headers, err := parseHeaders(lines[1:])
	if err != nil {
		return Response{}, err
	}
	return Response{
		Version:    parts[0],
		StatusCode: code,
		Reason:     reason,
		Headers:    headers,
	}, nil
}

func parseHeaders(lines []string) ([]Header, error) {
	headers := make([]Header, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("%s: malformed header", MalformedInput)
		}
		headers = append(headers, Header{
			Name:  strings.TrimSpace(name),
			Value: strings.TrimSpace(value),
		})
	}
	return headers, nil
}

func headerBlock(input string) (string, error) {
	if end := strings.Index(input, "\r\n\r\n"); end >= 0 {
		return input[:end], nil
	}
	if end := strings.Index(input, "\n\n"); end >= 0 {
		return input[:end], nil
	}
	return "", fmt.Errorf("%s: missing header terminator", IncompleteData)
}

func splitLines(input string) []string {
	return strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
}
