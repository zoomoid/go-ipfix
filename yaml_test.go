/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
