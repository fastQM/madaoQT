package mongo

import (
	"log"
	"testing"
)

func TestGetRecords(t *testing.T) {
	okexdiff := &OKExDiff{
		Config: &DBConfig{
			CollectionName: "DiffOKExHistory",
		},
	}
	okexdiff.Connect()

	defer okexdiff.Close()

	records, err := okexdiff.FindAll(map[string]interface{}{
		"coin": "eth",
	})

	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	log.Printf("Count:%v", len(records))
}
