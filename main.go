package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar"
)

var useJUnitXML bool
var useLineCount bool
var junitXMLPath string
var testFilePattern = ""
var excludeFilePattern = ""
var testFileSpaceSeparatedList = ""
var splitIndex int
var splitTotal int

func printMsg(msg string, args ...interface{}) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, msg)
	} else {
		fmt.Fprintf(os.Stderr, msg, args...)
	}
}

func fatalMsg(msg string, args ...interface{}) {
	printMsg(msg, args...)
	os.Exit(1)
}

func removeDeletedFiles(fileTimes map[string][]float64, currentFileSet map[string]bool) {
	for file := range fileTimes {
		if !currentFileSet[file] {
			delete(fileTimes, file)
		}
	}
}

func addNewFiles(fileTimes map[string]float64, currentFileSet map[string]bool) {
	averageFileTime := 0.0
	if len(fileTimes) > 0 {
		for _, time := range fileTimes {
			averageFileTime += time
		}
		averageFileTime /= float64(len(fileTimes))
	} else {
		averageFileTime = 1.0
	}

	for file := range currentFileSet {
		if _, isSet := fileTimes[file]; isSet {
			continue
		}
		if useJUnitXML {
			printMsg("missing file time for %s\n", file)
		}
		fileTimes[file] = averageFileTime
	}
}

func parseFlags() {
	flag.StringVar(&testFilePattern, "glob", "spec/**/*_spec.rb", "Glob pattern to find test files. Make sure to single-quote to avoid shell expansion.")
	flag.StringVar(&excludeFilePattern, "exclude-glob", "", "Glob pattern to exclude test files. Make sure to single-quote.")
	flag.StringVar(&testFileSpaceSeparatedList, "tests", "", "Space-separated list of files. Will be appended to the files found (and excluded) by the globs. Make sure to single-quote to avoid shell expansion.")

	flag.IntVar(&splitIndex, "split-index", -1, "This test container's index (or set CIRCLE_NODE_INDEX)")
	flag.IntVar(&splitTotal, "split-total", -1, "Total number of containers (or set CIRCLE_NODE_TOTAL)")

	flag.BoolVar(&useJUnitXML, "junit", false, "Use a JUnit XML report for test times")
	flag.StringVar(&junitXMLPath, "junit-path", "", "Path to a JUnit XML report (leave empty to read from stdin; use glob pattern to load multiple files)")

	flag.BoolVar(&useLineCount, "line-count", false, "Use line count to estimate test times")

	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Show this help text")

	flag.Parse()

	if showHelp {
		printMsg("Splits test files into containers of even duration\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if splitTotal == 0 || splitIndex < 0 || splitIndex > splitTotal {
		fatalMsg("-split-index and -split-total (and environment variables) are missing or invalid\n")
	}
}

func main() {
	parseFlags()

	currentFileSet := make(map[string]bool)

	if testFilePattern != "" {
		// We are not using filepath.Glob,
		// because it doesn't support '**' (to match all files in all nested directories)
		currentFiles, err := doublestar.Glob(testFilePattern)
		if err != nil {
			printMsg("failed to enumerate current file set: %v", err)
			os.Exit(1)
		}
		for _, file := range currentFiles {
			currentFileSet[file] = true
		}
	}

	if excludeFilePattern != "" {
		excludedFiles, err := doublestar.Glob(excludeFilePattern)
		if err != nil {
			printMsg("failed to enumerate excluded file set: %v", err)
			os.Exit(1)
		}
		for _, file := range excludedFiles {
			delete(currentFileSet, file)
		}
	}

	if testFileSpaceSeparatedList != "" {
		for _, file := range strings.Fields(testFileSpaceSeparatedList) {
			currentFileSet[file] = true
		}
	}

	fileTimes := make(map[string][]float64)

	if useLineCount {
		estimateFileTimesByLineCount(currentFileSet, fileTimes)
	} else if useJUnitXML {
		getFileTimesFromJUnitXML(fileTimes)
	}

	removeDeletedFiles(fileTimes, currentFileSet)

	// Print the file times map
	for file, time := range fileTimes {
		fmt.Printf("%s: %v\n", file, time)
	}

	// Create a new file time map with average times
	averageFileTimes := make(map[string]float64)

	for file, times := range fileTimes {
		averageFileTime := 0.0
		for _, time := range times {
			averageFileTime += time
		}
		averageFileTime /= float64(len(times))
		averageFileTimes[file] = averageFileTime
	}

	addNewFiles(averageFileTimes, currentFileSet)

	buckets, bucketTimes := splitFiles(averageFileTimes, splitTotal)
	if useJUnitXML {
		printMsg("expected test time: %0.1fs\n", bucketTimes[splitIndex])
	}

	fmt.Println(strings.Join(buckets[splitIndex], " "))
}
