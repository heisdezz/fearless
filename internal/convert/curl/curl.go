package curl

import (
	"fearless/internal/core/model"
)

// Import takes a raw cURL command and parses it into a model.Item and model.Request,
// returning any warnings for dropped or adapted flags.
func Import(cmd string) (*Result, error) {
	return Parse(cmd)
}

// ImportRequest is a convenience function that imports a cURL command and returns
// the model.Request and warnings.
func ImportRequest(cmd string) (*model.Request, []string, error) {
	res, err := Parse(cmd)
	if err != nil {
		return nil, nil, err
	}
	return res.Request, res.Warnings, nil
}

// ImportItem is a convenience function that imports a cURL command and returns
// the model.Item and warnings.
func ImportItem(cmd string) (*model.Item, []string, error) {
	res, err := Parse(cmd)
	if err != nil {
		return nil, nil, err
	}
	return res.Item, res.Warnings, nil
}
