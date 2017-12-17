package task

import (
	"encoding/json"
	"log"
	"testing"
)

func TestConfigToJson(t *testing.T) {
	configJSON := "{\"area\":{\"ltc\":{\"open\":3, \"close\":1.5}}, \"limitclose\":0.03, \"limitopen\":0.005}"

	var config AnalyzerConfig
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		log.Printf("Err:%v", err)
		return
	}

	log.Printf("Config:%v", config)

}
