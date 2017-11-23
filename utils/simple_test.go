package utils

import (
	"testing"
	"sort"
	"log"
)

func TestStringCompare(t *testing.T){
	test := []string{"hello", "hallo", "bee", "fllll", "meeee", "maaa", "zzz"};
	sort.Strings(test)
	log.Printf("Result:%v", test)

}