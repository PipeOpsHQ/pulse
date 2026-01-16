package main

import (
	"bufio"
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

// ParseLCOVCoverage parses an lcov.info file
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
