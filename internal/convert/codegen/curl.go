package codegen

import (
	"fearless/internal/convert/curl"
	"fearless/internal/core/model"
)

func init() {
	Register(TargetCurl, GenerateCurl)
}

// GenerateCurl generates a runnable cURL command snippet from a model.Request.
func GenerateCurl(req *model.Request) (string, error) {
	return curl.Export(req)
}
