package jsonutil

import (
	"encoding/json"
	"log/slog"
)

func GetAsJSONBytes(v any) []byte {
	jsonBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		slog.Warn("failed to marshal to JSON", "error", err)

		return nil
	}

	return jsonBytes
}

func GetAsJSONString(v any) string {
	return string(GetAsJSONBytes(v))
}
