package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

// ParseGoCoverage parses a coverage.out file and returns total percentage and file breakdown
func ParseGoCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	scanner := bufio.NewScanner(reader)
	var files []FileCoverage
	var totalStmts int64
	var coveredStmts int64

	// mode: set (usually first line)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}

		// github.com/user/repo/file.go:10.2,12.5 1 1
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		stmts, _ := strconv.ParseInt(parts[len(parts)-2], 10, 64)
		count, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)

		filePath := strings.Split(parts[0], ":")[0]

		totalStmts += stmts
		if count > 0 {
			coveredStmts += stmts
		}

		// Aggregate per file (simplistic for now, in a real one we'd sum up)
		found := false
		for _, f := range files {
			if f.FilePath == filePath {
				// This is a bit complex for a one-pass scanner without a map
				// Let's use a temporary map for better accuracy
				found = true
				break
			}
		}
		if !found {
			files = append(files, FileCoverage{FilePath: filePath, Percentage: 0})
		}
	}

	// Second pass/Real implementation using a map for accuracy
	return calculateGoTotals(reader)
}

// Improved Go parser using a map
func calculateGoTotals(reader io.Reader) (float64, []FileCoverage, error) {
	type fileStat struct {
		total   int64
		covered int64
	}
	stats := make(map[string]*fileStat)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		filePath := strings.Split(parts[0], ":")[0]
		stmts, _ := strconv.ParseInt(parts[len(parts)-2], 10, 64)
		count, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)

		if _, ok := stats[filePath]; !ok {
			stats[filePath] = &fileStat{}
		}
		stats[filePath].total += stmts
		if count > 0 {
			stats[filePath].covered += stmts
		}
	}

	var totalStmts, coveredStmts int64
	var fileBreakdown []FileCoverage
	for path, s := range stats {
		percentage := 0.0
		if s.total > 0 {
			percentage = (float64(s.covered) / float64(s.total)) * 100
		}
		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   path,
			Percentage: percentage,
		})
		totalStmts += s.total
		coveredStmts += s.covered
	}

	totalPercentage := 0.0
	if totalStmts > 0 {
		totalPercentage = (float64(coveredStmts) / float64(totalStmts)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}

// ParseLCOVCoverage parses an lcov.info file (JavaScript, C++, etc.)
func ParseLCOVCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	scanner := bufio.NewScanner(reader)
	var fileBreakdown []FileCoverage
	var currentFile string
	var totalFound, totalHit int64
	var fileFound, fileHit int64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "SF:") {
			currentFile = strings.TrimPrefix(line, "SF:")
			fileFound = 0
			fileHit = 0
		} else if strings.HasPrefix(line, "LF:") {
			fileFound, _ = strconv.ParseInt(strings.TrimPrefix(line, "LF:"), 10, 64)
			totalFound += fileFound
		} else if strings.HasPrefix(line, "LH:") {
			fileHit, _ = strconv.ParseInt(strings.TrimPrefix(line, "LH:"), 10, 64)
			totalHit += fileHit
		} else if line == "end_of_record" {
			percentage := 0.0
			if fileFound > 0 {
				percentage = (float64(fileHit) / float64(fileFound)) * 100
			}
			fileBreakdown = append(fileBreakdown, FileCoverage{
				FilePath:   currentFile,
				Percentage: percentage,
			})
		}
	}

	totalPercentage := 0.0
	if totalFound > 0 {
		totalPercentage = (float64(totalHit) / float64(totalFound)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}

// Cobertura XML structures (Java, Python, C#, PHP, etc.)
type CoberturaXML struct {
	XMLName  xml.Name           `xml:"coverage"`
	Packages []CoberturaPackage `xml:"packages>package"`
}

type CoberturaPackage struct {
	Name    string           `xml:"name,attr"`
	Classes []CoberturaClass `xml:"classes>class"`
}

type CoberturaClass struct {
	Name       string  `xml:"name,attr"`
	Filename   string  `xml:"filename,attr"`
	LineRate   float64 `xml:"line-rate,attr"`
	BranchRate float64 `xml:"branch-rate,attr"`
}

// ParseCoberturaCoverage parses Cobertura XML format
func ParseCoberturaCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov CoberturaXML
	if err := xml.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	fileStats := make(map[string]float64)
	for _, pkg := range cov.Packages {
		for _, class := range pkg.Classes {
			fileStats[class.Filename] = class.LineRate * 100
		}
	}

	var fileBreakdown []FileCoverage
	var totalRate float64
	for path, rate := range fileStats {
		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   path,
			Percentage: rate,
		})
		totalRate += rate
	}

	totalPercentage := 0.0
	if len(fileStats) > 0 {
		totalPercentage = totalRate / float64(len(fileStats))
	}

	return totalPercentage, fileBreakdown, nil
}

// JaCoCo XML structures (Java)
type JaCoCoXML struct {
	XMLName  xml.Name        `xml:"report"`
	Packages []JaCoCoPackage `xml:"package"`
}

type JaCoCoPackage struct {
	Name        string             `xml:"name,attr"`
	SourceFiles []JaCoCoSourceFile `xml:"sourcefile"`
}

type JaCoCoSourceFile struct {
	Name     string          `xml:"name,attr"`
	Counters []JaCoCoCounter `xml:"counter"`
}

type JaCoCoCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

// ParseJaCoCoCoverage parses JaCoCo XML format
func ParseJaCoCoCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov JaCoCoXML
	if err := xml.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	var fileBreakdown []FileCoverage
	var totalLines, coveredLines int

	for _, pkg := range cov.Packages {
		for _, sf := range pkg.SourceFiles {
			var fileMissed, fileCovered int
			for _, counter := range sf.Counters {
				if counter.Type == "LINE" {
					fileMissed = counter.Missed
					fileCovered = counter.Covered
					totalLines += fileMissed + fileCovered
					coveredLines += fileCovered
					break
				}
			}

			percentage := 0.0
			total := fileMissed + fileCovered
			if total > 0 {
				percentage = (float64(fileCovered) / float64(total)) * 100
			}

			filePath := pkg.Name + "/" + sf.Name
			fileBreakdown = append(fileBreakdown, FileCoverage{
				FilePath:   filePath,
				Percentage: percentage,
			})
		}
	}

	totalPercentage := 0.0
	if totalLines > 0 {
		totalPercentage = (float64(coveredLines) / float64(totalLines)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}

// Istanbul/NYC JSON structures (JavaScript/TypeScript)
type IstanbulJSON map[string]IstanbulFile

type IstanbulFile struct {
	Path       string                  `json:"path"`
	Statements IstanbulCoverageMetrics `json:"s"`
	Branches   IstanbulCoverageMetrics `json:"b"`
	Functions  IstanbulCoverageMetrics `json:"f"`
	Lines      map[string]int          `json:"l"`
}

type IstanbulCoverageMetrics map[string]int

// ParseIstanbulCoverage parses Istanbul/NYC JSON format
func ParseIstanbulCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov IstanbulJSON
	if err := json.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	var fileBreakdown []FileCoverage
	var totalLines, coveredLines int

	for path, file := range cov {
		var fileCovered, fileTotal int
		for _, count := range file.Lines {
			fileTotal++
			if count > 0 {
				fileCovered++
			}
		}

		percentage := 0.0
		if fileTotal > 0 {
			percentage = (float64(fileCovered) / float64(fileTotal)) * 100
		}

		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   path,
			Percentage: percentage,
		})

		totalLines += fileTotal
		coveredLines += fileCovered
	}

	totalPercentage := 0.0
	if totalLines > 0 {
		totalPercentage = (float64(coveredLines) / float64(totalLines)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}

// SimpleCov JSON structures (Ruby)
type SimpleCovJSON struct {
	Coverage map[string]SimpleCovFile `json:"coverage"`
}

type SimpleCovFile struct {
	Lines []interface{} `json:"lines"`
}

// ParseSimpleCovCoverage parses SimpleCov JSON format
func ParseSimpleCovCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov SimpleCovJSON
	if err := json.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	var fileBreakdown []FileCoverage
	var totalLines, coveredLines int

	for path, file := range cov.Coverage {
		var fileCovered, fileTotal int
		for _, line := range file.Lines {
			if line == nil {
				continue // Ignore non-executable lines
			}
			fileTotal++
			if count, ok := line.(float64); ok && count > 0 {
				fileCovered++
			}
		}

		percentage := 0.0
		if fileTotal > 0 {
			percentage = (float64(fileCovered) / float64(fileTotal)) * 100
		}

		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   path,
			Percentage: percentage,
		})

		totalLines += fileTotal
		coveredLines += fileCovered
	}

	totalPercentage := 0.0
	if totalLines > 0 {
		totalPercentage = (float64(coveredLines) / float64(totalLines)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}

// Coverage.py JSON structures (Python)
type CoveragePyJSON struct {
	Files  map[string]CoveragePyFile `json:"files"`
	Totals CoveragePyTotals          `json:"totals"`
}

type CoveragePyFile struct {
	ExecutedLines []int            `json:"executed_lines"`
	MissingLines  []int            `json:"missing_lines"`
	Summary       CoveragePyTotals `json:"summary"`
}

type CoveragePyTotals struct {
	CoveredLines   int     `json:"covered_lines"`
	NumStatements  int     `json:"num_statements"`
	PercentCovered float64 `json:"percent_covered"`
	MissingLines   int     `json:"missing_lines"`
}

// ParseCoveragePyCoverage parses Coverage.py JSON format
func ParseCoveragePyCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov CoveragePyJSON
	if err := json.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	var fileBreakdown []FileCoverage
	for path, file := range cov.Files {
		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   path,
			Percentage: file.Summary.PercentCovered,
		})
	}

	return cov.Totals.PercentCovered, fileBreakdown, nil
}

// Clover XML structures (PHP, JavaScript)
type CloverXML struct {
	XMLName xml.Name      `xml:"coverage"`
	Project CloverProject `xml:"project"`
}

type CloverProject struct {
	Files   []CloverFile  `xml:"file"`
	Metrics CloverMetrics `xml:"metrics"`
}

type CloverFile struct {
	Name    string        `xml:"name,attr"`
	Metrics CloverMetrics `xml:"metrics"`
}

type CloverMetrics struct {
	Elements          int `xml:"elements,attr"`
	CoveredElements   int `xml:"coveredelements,attr"`
	Statements        int `xml:"statements,attr"`
	CoveredStatements int `xml:"coveredstatements,attr"`
}

// ParseCloverCoverage parses Clover XML format
func ParseCloverCoverage(reader io.Reader) (float64, []FileCoverage, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, nil, err
	}

	var cov CloverXML
	if err := xml.Unmarshal(data, &cov); err != nil {
		return 0, nil, err
	}

	var fileBreakdown []FileCoverage
	for _, file := range cov.Project.Files {
		percentage := 0.0
		if file.Metrics.Statements > 0 {
			percentage = (float64(file.Metrics.CoveredStatements) / float64(file.Metrics.Statements)) * 100
		}

		fileBreakdown = append(fileBreakdown, FileCoverage{
			FilePath:   file.Name,
			Percentage: percentage,
		})
	}

	totalPercentage := 0.0
	if cov.Project.Metrics.Statements > 0 {
		totalPercentage = (float64(cov.Project.Metrics.CoveredStatements) / float64(cov.Project.Metrics.Statements)) * 100
	}

	return totalPercentage, fileBreakdown, nil
}
