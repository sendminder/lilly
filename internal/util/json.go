package util

import "encoding/json"

// key, value로 마샬링
func CreateJsonData(key string, value interface{}) ([]byte, error) {
	data := map[string]interface{}{
		key: value,
	}

	// JSON 마샬링
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
