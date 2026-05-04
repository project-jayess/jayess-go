package picohttpparser

import "strings"

type IncrementalParser struct {
	buffer string
}

func (parser *IncrementalParser) Feed(chunk string) bool {
	parser.buffer += chunk
	return parser.Complete()
}

func (parser IncrementalParser) Complete() bool {
	return strings.Contains(parser.buffer, "\r\n\r\n") || strings.Contains(parser.buffer, "\n\n")
}

func (parser IncrementalParser) Request() (Request, error) {
	return ParseRequest(parser.buffer)
}

func (parser IncrementalParser) Response() (Response, error) {
	return ParseResponse(parser.buffer)
}
