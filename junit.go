package main

import (
	"encoding/xml"
	"io"
	"os"
	"path"

	"github.com/bmatcuk/doublestar"
)

type TestSuites struct {
	TestSuites []TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	File string  `xml:"filepath,attr"`
	Time float64 `xml:"time,attr"`
}

func loadJUnitXML(xmlData []byte) *TestSuites {
	var testSuites TestSuites
	err := xml.Unmarshal(xmlData, &testSuites)
	if err != nil {
		fatalMsg("failed to unmarshal junit xml: %v\n", err)
	}
	return &testSuites
}

func addFileTimesFromIOReader(fileTimes map[string][]float64, reader io.Reader) {
	xmlData, err := io.ReadAll(reader)

	if err != nil {
		fatalMsg("failed to read junit xml: %v\n", err)
	}

	testSuites := loadJUnitXML(xmlData)

	for _, testSuite := range testSuites.TestSuites {
		filePath := path.Clean(testSuite.File)
		printMsg("adding test time for %s\n", filePath)
		printMsg("  time: %f\n", testSuite.Time)
		fileTimes[filePath] = append(fileTimes[filePath], testSuite.Time)
	}
}

func getFileTimesFromJUnitXML(fileTimes map[string][]float64) {
	if junitXMLPath != "" {
		filenames, err := doublestar.Glob(junitXMLPath)
		if err != nil {
			fatalMsg("failed to match jUnit filename pattern: %v", err)
		}
		for _, junitFilename := range filenames {
			file, err := os.Open(junitFilename)
			if err != nil {
				fatalMsg("failed to open junit xml: %v\n", err)
			}
			printMsg("using test times from JUnit report %s\n", junitFilename)
			addFileTimesFromIOReader(fileTimes, file)
			file.Close()
		}
	} else {
		printMsg("using test times from JUnit report at stdin\n")
		addFileTimesFromIOReader(fileTimes, os.Stdin)
	}
}
