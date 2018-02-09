package mongo

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestGetRecords(t *testing.T) {
	okexdiff := &OKExDiff{
		Config: &DBConfig{
			CollectionName: "DiffOKExHistory",
		},
	}
	okexdiff.Connect()

	defer okexdiff.Close()

	now := time.Now()
	start := now.Add(-12 * time.Hour)
	log.Printf("start:%v stop:%v", start, now)
	records, err := okexdiff.FindAll("eth", start, now)

	if err != nil {
		log.Printf("Error:%v", err)
		return
	}

	var totalDiff float64
	datas := ""
	for _, record := range records {
		datas = fmt.Sprintf("%s,%f", datas, record.Diff)
		totalDiff += record.Diff
	}

	log.Printf("Ave:%v", totalDiff/float64(len(records)))
	log.Printf("%s", datas)
}
