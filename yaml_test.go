package ipfix

import (
	"os"
	"testing"
)

func TestWriteYAML(t *testing.T) {
	srcFile, _ := os.Open("./ipfix-information-elements.csv")
	defer srcFile.Close()
	m, err := ReadCSV(srcFile)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.CreateTemp("", "ipfix_iana_fields-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	err = WriteYAML(file, m)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReadYAML(t *testing.T) {
	srcFile, _ := os.Open("./ipfix-information-elements.csv")
	defer srcFile.Close()
	m, err := ReadCSV(srcFile)
	if err != nil {
		t.Fatal(err)
	}

	destFile, err := os.CreateTemp("", "ipfix_iana_fields-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer destFile.Close()

	err = WriteYAML(destFile, m)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(destFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, err = ReadYAML(file)
	if err != nil {
		t.Fatal(err)
	}
}
