package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestPanicCheckAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, PanicCheckAnalyzer,
		"mainok",
		"mainbad",
		"nonmain",
	)
}
