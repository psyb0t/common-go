package utils

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

func GetAsJSONBytes(v any) []byte {
	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logrus.Warnf("failed to marshal to JSON: %s", err)

		return nil
	}

	return jsonBytes
}

func GetAsJSONString(v any) string {
	return string(GetAsJSONBytes(v))
}
