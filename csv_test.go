package ipfix

import (
	"os"
	"testing"
)

func TestReadCSV(t *testing.T) {
	srcFile, _ := os.Open("./ipfix-information-elements.csv")
	defer srcFile.Close()
	_, err := ReadCSV(srcFile)
	if err != nil {
		t.Fatal(err)
	}
}
