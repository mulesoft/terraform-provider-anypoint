package exchange

// This file holds the PURE readers for the parts of an Exchange asset payload whose
// location is not obvious: api_version lives in the attributes list, and classifier /
// main_file live on the uploaded file rather than at the top level.
//
// They live in the client package so the resource and the data source read them the
// same way — an asset inspected through the data source reports the same values a
// managed one does.

// ExtractAttributeValue returns the value of a named attribute, or "" when absent.
func ExtractAttributeValue(attributes []interface{}, key string) string {
	for _, attr := range attributes {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}
		if attrKey, _ := attrMap["key"].(string); attrKey == key {
			if val, ok := attrMap["value"].(string); ok {
				return val
			}
		}
	}
	return ""
}

// ExtractFileMetadata returns the classifier and main file of the user-uploaded file.
//
// Exchange explodes an upload into several derived files, so the generated ones and the
// always-present pom are skipped; the first remaining file carrying a classifier is the
// one the practitioner actually published.
func ExtractFileMetadata(files []AssetFile) (classifier string, mainFile string) {
	for _, f := range files {
		if f.IsGenerated {
			continue
		}
		if f.Packaging == "pom" && f.Classifier == "" {
			continue
		}
		if f.Classifier != "" {
			return f.Classifier, f.MainFile
		}
	}
	return "", ""
}
