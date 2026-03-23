package utils

import "encoding/json"

func MarshalToJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func UnmarshalFromJSON(jsonStr string, result any) error {
	return json.Unmarshal([]byte(jsonStr), result)
}
