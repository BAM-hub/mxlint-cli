package lint

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mxlint/mxlint-cli/cache"
)

const NOQA = "# noqa"
const NOQA_ALIAS = "#noqa"

func printTestsuite(ts Testsuite) {
	fmt.Printf("## %s\n", ts.Name)
	for _, tc := range ts.Testcases {
		result := "PASS"
		if tc.Failure != nil {
			result = "FAIL"
		}
		if tc.Skipped != nil {
			result = "SKIP"
		}
		fmt.Printf("%s (%.5fs) %s\n", result, tc.Time, tc.Name)
	}
	fmt.Println("")
}

// EvalAllWithResults evaluates all rules and returns the results
// This is similar to EvalAll but returns the results instead of just printing them
func EvalAllWithResults(rulesPath string, modelSourcePath string, xunitReport string, jsonFile string) (interface{}, error) {
	testsuites := make([]Testsuite, 0)
	rules, err := ReadRulesMetadata(rulesPath)
	mxCache, err := cache.GetDiffCache(modelSourcePath)

	if err != nil {
		return nil, err
	}
	failuresCount := 0
	for _, rule := range rules {
		testsuite, err := evalTestsuite(rule, modelSourcePath, mxCache)
		if err != nil {
			return nil, err
		}
		printTestsuite(*testsuite)
		failuresCount += testsuite.Failures
		testsuites = append(testsuites, *testsuite)
	}

	if xunitReport != "" {
		file, err := os.Create(xunitReport)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		encoder := xml.NewEncoder(file)
		encoder.Indent("", "  ")
		testsuitesContainer := TestSuites{Testsuites: testsuites}
		if err := encoder.Encode(testsuitesContainer); err != nil {
			panic(err)
		}
	}

	if jsonFile != "" {
		file, err := os.Create(jsonFile)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		testsuitesContainer := TestSuites{Testsuites: testsuites, Rules: rules}
		if err := encoder.Encode(testsuitesContainer); err != nil {
			panic(err)
		}
	}

	for _, ts := range testsuites {
		if ts.Failures > 0 {
			log.Warningf("Rule %s: %d failures", ts.Name, ts.Failures)
			for _, tc := range ts.Testcases {
				if tc.Failure != nil {
					log.Warningf("  Document %s: %s", tc.Name, tc.Failure.Message)
				}
			}
		}
	}

	// Return the results
	testsuitesContainer := TestSuites{Testsuites: testsuites, Rules: rules}

	if failuresCount > 0 {
		return testsuitesContainer, fmt.Errorf("%d failures", failuresCount)
	} else {
		log.Infof("Lint summary: All rules passed successfully!")
		log.Infof("Total rules evaluated: %d", len(rules))
		log.Infof("Total files checked: %d", countTotalTestcases(testsuites))
	}
	return testsuitesContainer, nil
}

func EvalAll(rulesPath string, modelSourcePath string, xunitReport string, jsonFile string) error {
	testsuites := make([]Testsuite, 0)
	rules, err := ReadRulesMetadata(rulesPath)
	if err != nil {
		return err
	}

	mxCache, err := cache.GetDiffCache(modelSourcePath)

	if err != nil {
		return err
	}

	failuresCount := 0
	for _, rule := range rules {
		testsuite, err := evalTestsuite(rule, modelSourcePath, mxCache)
		if err != nil {
			return err
		}
		printTestsuite(*testsuite)
		failuresCount += testsuite.Failures
		testsuites = append(testsuites, *testsuite)
	}

	if xunitReport != "" {
		file, err := os.Create(xunitReport)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		encoder := xml.NewEncoder(file)
		encoder.Indent("", "  ")
		testsuitesContainer := TestSuites{Testsuites: testsuites}
		if err := encoder.Encode(testsuitesContainer); err != nil {
			panic(err)
		}
	}

	if jsonFile != "" {
		file, err := os.ReadFile(jsonFile)

		if os.IsNotExist(err) || len(file) == 0 {
			// new file creation
			newFile, err := os.Create(jsonFile)

			if err != nil {
				panic(err)
			}
			defer newFile.Close()

			encoder := json.NewEncoder(newFile)
			encoder.SetIndent("", "  ")
			testsuitesContainer := TestSuites{Testsuites: testsuites, Rules: rules}
			if err := encoder.Encode(testsuitesContainer); err != nil {
				panic(err)
			}
		}

		var jsonData TestSuites
		err = json.Unmarshal(file, &jsonData)

		if err != nil {
			return fmt.Errorf("%v", err)
		}

		for index, testsuite := range testsuites {
			// if the new data has tests this meas this was diffed in cache
			if testsuite.Tests > 0 {
				for _, testCase := range testsuite.Testcases {
					testCaseIndex := findTestCase(jsonData.Testsuites[index].Testcases, testCase.Name)
					if testCaseIndex != -1 {

						oldTestSuite := jsonData.Testsuites[index]
						oldTestCase := jsonData.Testsuites[index].Testcases[testCaseIndex]

						if testCase.IsDeleted {
							// just remove it and keep going
							oldTestSuite.Time -= testCase.Time
							length := len(oldTestSuite.Testcases) - 1
							oldTestSuite.Testcases[testCaseIndex] = oldTestSuite.Testcases[length]
							oldTestSuite.Testcases = oldTestSuite.Testcases[:length]
							jsonData.Testsuites[index] = oldTestSuite
							continue
						}

						// remove the old faliure
						if oldTestCase.Failure != nil {
							oldTestSuite.Failures--
						}

						// add the new faliure if exists
						if testCase.Failure != nil {
							oldTestSuite.Failures++
						}

						oldTestSuite.Time -= oldTestCase.Time
						oldTestSuite.Time += testCase.Time

						jsonData.Testsuites[index].Testcases[testCaseIndex] = testCase
						jsonData.Testsuites[index] = oldTestSuite
						break
					} else {
						oldTestSuite := jsonData.Testsuites[index]
						oldTestSuite.Time += testCase.Time

						if testCase.Failure != nil {
							oldTestSuite.Failures++
						}

						jsonData.Testsuites[index] = oldTestSuite
						jsonData.Testsuites[index].Testcases = append(jsonData.Testsuites[index].Testcases, testCase)
					}

				}

			}

		}
		cache.InvalidateCahce(modelSourcePath, false, false)
		newFile, err := os.Create(jsonFile)

		if err != nil {
			panic(err)
		}
		defer newFile.Close()

		encoder := json.NewEncoder(newFile)
		encoder.SetIndent("", "  ")
		testsuitesContainer := TestSuites{Testsuites: jsonData.Testsuites, Rules: rules}
		if err := encoder.Encode(testsuitesContainer); err != nil {
			panic(err)
		}
	}

	for _, ts := range testsuites {
		if ts.Failures > 0 {
			log.Warningf("Rule %s: %d failures", ts.Name, ts.Failures)
			for _, tc := range ts.Testcases {
				if tc.Failure != nil {
					log.Warningf("  Document %s: %s", tc.Name, tc.Failure.Message)
				}
			}
		}
	}

	if failuresCount > 0 {
		log.Errorf("Lint summary: Found %d failures:", failuresCount)
		log.Errorf("Failures by rule:")
		for _, ts := range testsuites {
			if ts.Failures > 0 {
				log.Errorf("- %s: %d failures", ts.Name, ts.Failures)
			}
		}
		return fmt.Errorf("%d failures", failuresCount)
	} else {
		log.Infof("Lint summary: All rules passed successfully!")
		log.Infof("Total rules evaluated: %d", len(rules))
		log.Infof("Total files checked: %d", countTotalTestcases(testsuites))
	}

	// cache.InvalidateCahce(modelSourcePath)
	return nil
}

func findTestCase(slice []Testcase, value string) int {

	for index, item := range slice {
		if item.Name == value {
			return index
		}
	}
	return -1
}

// countTotalTestcases returns the total number of testcases across all testsuites
func countTotalTestcases(testsuites []Testsuite) int {
	count := 0
	for _, ts := range testsuites {
		count += len(ts.Testcases)
	}
	return count
}

func evalTestsuite(rule Rule, modelSourcePath string, changedFiles cache.MxCacheDiffWrapper) (*Testsuite, error) {

	log.Debugf("evaluating rule %s", rule.Path)

	queryString := "data." + rule.PackageName
	testcases := make([]Testcase, 0)
	failuresCount := 0
	skippedCount := 0
	totalTime := 0.0
	inputFiles, err := expandPaths(rule.Pattern, modelSourcePath, changedFiles, false)
	if err != nil {
		return nil, err
	}

	testcase := &Testcase{}

	for _, inputFile := range inputFiles {

		if inputFile.diffType == "delete" {
			testcase = &Testcase{
				Name:      inputFile.path,
				IsDeleted: true,
				Time:      0,
				XMLName:   xml.Name{},
				Skipped:   nil,
				Failure:   nil,
			}
			testcases = append(testcases, *testcase)
			continue
		}

		if rule.Language == LanguageRego {
			testcase, err = evalTestcase_Rego(rule.Path, queryString, inputFile.path)
		} else if rule.Language == LanguageJavascript {
			testcase, err = evalTestcase_Javascript(rule.Path, inputFile.path)
		}

		if err != nil {
			return nil, err
		}
		if testcase.Failure != nil {
			failuresCount++
		}

		if testcase.Skipped != nil {
			skippedCount++
		}

		totalTime += testcase.Time
		testcases = append(testcases, *testcase)
	}

	testsuite := &Testsuite{
		Name:      rule.Path,
		Tests:     len(testcases),
		Failures:  failuresCount,
		Skipped:   skippedCount,
		Time:      totalTime,
		Testcases: testcases,
	}

	return testsuite, nil
}

func ReadRulesMetadata(rulesPath string) ([]Rule, error) {
	rules := make([]Rule, 0)
	filepath.Walk(rulesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), "_test.rego") && strings.HasSuffix(info.Name(), ".rego") {
			rule, err := parseRuleMetadata_Rego(path)
			if err != nil {
				return err
			}
			rules = append(rules, *rule)
		}
		if !info.IsDir() && !strings.HasSuffix(info.Name(), "_test.js") && strings.HasSuffix(info.Name(), ".js") {
			rule, err := parseRuleMetadata_Javascript(path)
			if err != nil {
				return err
			}
			rules = append(rules, *rule)
		}
		return nil
	})
	return rules, nil
}
