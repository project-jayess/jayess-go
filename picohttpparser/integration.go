package picohttpparser

type IntegrationFeature string

const (
	JayessObjectConversion IntegrationFeature = "jayess-object-conversion"
	MongooseHTTPInput      IntegrationFeature = "mongoose-http-input"
	CriticalPathParsing    IntegrationFeature = "critical-path-parsing"
)

func IntegrationFeatures() []IntegrationFeature {
	return []IntegrationFeature{
		JayessObjectConversion,
		MongooseHTTPInput,
		CriticalPathParsing,
	}
}

func RequestObject(request Request) map[string]any {
	return map[string]any{
		"method":  request.Method,
		"path":    request.Path,
		"version": request.Version,
		"headers": headersObject(request.Headers),
	}
}

func ResponseObject(response Response) map[string]any {
	return map[string]any{
		"version":    response.Version,
		"statusCode": response.StatusCode,
		"reason":     response.Reason,
		"headers":    headersObject(response.Headers),
	}
}

func headersObject(headers []Header) []map[string]string {
	values := make([]map[string]string, 0, len(headers))
	for _, header := range headers {
		values = append(values, map[string]string{
			"name":  header.Name,
			"value": header.Value,
		})
	}
	return values
}
