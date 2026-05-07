package runtime

import (
	"net/url"
	"path/filepath"
	"strings"
)

type ParsedURL struct {
	Scheme   string
	Host     string
	Path     string
	Query    map[string][]string
	Fragment string
}

func URLParse(input string) (ParsedURL, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return ParsedURL{}, err
	}
	return ParsedURL{
		Scheme:   parsed.Scheme,
		Host:     parsed.Host,
		Path:     parsed.Path,
		Query:    map[string][]string(parsed.Query()),
		Fragment: parsed.Fragment,
	}, nil
}

func URLFormat(parsed ParsedURL) string {
	values := url.Values(parsed.Query)
	out := url.URL{Scheme: parsed.Scheme, Host: parsed.Host, Path: parsed.Path, RawQuery: values.Encode(), Fragment: parsed.Fragment}
	return out.String()
}

func URLParseQuery(input string) (map[string][]string, error) {
	values, err := url.ParseQuery(input)
	return map[string][]string(values), err
}

func URLStringifyQuery(values map[string][]string) string {
	return url.Values(values).Encode()
}

func URLEncode(input string) string {
	return url.QueryEscape(input)
}

func URLDecode(input string) (string, error) {
	return url.QueryUnescape(input)
}

func URLFileURLToPath(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	return filepath.FromSlash(parsed.Path), nil
}

func URLPathToFileURL(path string) string {
	clean := filepath.ToSlash(path)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return (&url.URL{Scheme: "file", Path: clean}).String()
}
