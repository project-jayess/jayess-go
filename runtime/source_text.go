package runtime

import (
	"strings"
	"unicode/utf8"
)

type SourceLocation struct {
	ByteOffset int
	Line       int
	Column     int
}

type SourceText struct {
	text       string
	lineStarts []int
}

func NewSourceText(text string) SourceText {
	return SourceText{text: text, lineStarts: sourceLineStarts(text)}
}

func (source SourceText) Text() string {
	return source.text
}

func (source SourceText) ByteLen() int {
	return len(source.text)
}

func (source SourceText) RuneLen() int {
	return utf8.RuneCountInString(source.text)
}

func (source SourceText) Slice(start int, end int) (string, bool) {
	if start < 0 || end < start || end > len(source.text) {
		return "", false
	}
	if !isRuneBoundary(source.text, start) || !isRuneBoundary(source.text, end) {
		return "", false
	}
	return source.text[start:end], true
}

func (source SourceText) Location(byteOffset int) (SourceLocation, bool) {
	if byteOffset < 0 || byteOffset > len(source.text) {
		return SourceLocation{}, false
	}
	if !isRuneBoundary(source.text, byteOffset) {
		return SourceLocation{}, false
	}
	lineIndex := source.lineIndex(byteOffset)
	return SourceLocation{
		ByteOffset: byteOffset,
		Line:       lineIndex + 1,
		Column:     utf8.RuneCountInString(source.text[source.lineStarts[lineIndex]:byteOffset]) + 1,
	}, true
}

func ConcatSourceText(parts ...SourceText) SourceText {
	var builder strings.Builder
	for _, part := range parts {
		builder.WriteString(part.text)
	}
	return NewSourceText(builder.String())
}

func (source SourceText) lineIndex(byteOffset int) int {
	low := 0
	high := len(source.lineStarts) - 1
	for low <= high {
		mid := low + (high-low)/2
		if source.lineStarts[mid] <= byteOffset {
			if mid == len(source.lineStarts)-1 || source.lineStarts[mid+1] > byteOffset {
				return mid
			}
			low = mid + 1
			continue
		}
		high = mid - 1
	}
	return 0
}

func isRuneBoundary(text string, offset int) bool {
	return offset == 0 || offset == len(text) || utf8.RuneStart(text[offset])
}

func sourceLineStarts(text string) []int {
	starts := []int{0}
	for index, r := range text {
		if r == '\n' {
			starts = append(starts, index+1)
		}
	}
	return starts
}
