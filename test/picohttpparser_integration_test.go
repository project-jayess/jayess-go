package test

import (
	"testing"

	"jayess-go/picohttpparser"
)

func TestPicoHTTPParserRequestCanConvertToJayessObject(t *testing.T) {
	request, err := picohttpparser.ParseRequest("POST /api HTTP/1.1\r\nContent-Type: application/json\r\n\r\n")
	if err != nil {
		t.Fatalf("expected request to parse, got %v", err)
	}

	object := picohttpparser.RequestObject(request)
	if object["method"] != "POST" || object["path"] != "/api" {
		t.Fatalf("unexpected Jayess request object: %#v", object)
	}
	headers, ok := object["headers"].([]map[string]string)
	if !ok || len(headers) != 1 || headers[0]["name"] != "Content-Type" {
		t.Fatalf("unexpected Jayess request headers object: %#v", object["headers"])
	}
}

func TestPicoHTTPParserIntegrationFeaturesIncludeMongooseAndCriticalPaths(t *testing.T) {
	features := picohttpparser.IntegrationFeatures()
	for _, want := range []picohttpparser.IntegrationFeature{
		picohttpparser.JayessObjectConversion,
		picohttpparser.MongooseHTTPInput,
		picohttpparser.CriticalPathParsing,
	} {
		if !picoHTTPParserHasIntegrationFeature(features, want) {
			t.Fatalf("expected integration feature %s in %#v", want, features)
		}
	}
}

func picoHTTPParserHasIntegrationFeature(values []picohttpparser.IntegrationFeature, want picohttpparser.IntegrationFeature) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
