package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var columnsToRemove = map[string]bool{
	"DR_NO":        true,
	"Rpt Dist No":  true,
	"Mocodes":      true,
	"Crm Cd 2":     true,
	"Crm Cd 3":     true,
	"Crm Cd 4":     true,
	"Cross Street": true,
}

var multiSpaceRegex = regexp.MustCompile(`\s+`)

func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

type Coord struct {
	Lat float64
	Lon float64
}

// parseDate attempts to parse a date from several common formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		"1/2/2006 15:04",
		"01/02/2006 03:04:05 PM",
		"01/02/2006",
		"1/2/2006",
		"2006-01-02 15:04:05",
	}
	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format: %s", s)
}

func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sort.Float64s(data)
	n := len(data)
	if n%2 == 0 {
		return (data[n/2-1] + data[n/2]) / 2.0
	}
	return data[n/2]
}

func main() {
	inputFile := "data/raw/Crime_Data_from_2020_to_Present.csv"
	outputFile := "data/processed/Crime_Data_Clean.csv"

	if len(os.Args) > 1 {
		inputFile = os.Args[1]
	}
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	log.Printf("Starting data cleaning process...")
	log.Printf("Input file: %s", inputFile)
	log.Printf("Output file: %s", outputFile)

	outDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// ---------------------------------------------------------
	// PASS 1: Compute LAT/LON medians by AREA NAME
	// ---------------------------------------------------------
	log.Println("Pass 1: Computing LAT/LON medians by AREA NAME...")
	inFile, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	reader.ReuseRecord = true

	headers, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read header: %v", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[h] = i
	}

	areaIdx, okArea := headerMap["AREA NAME"]
	latIdx, okLat := headerMap["LAT"]
	lonIdx, okLon := headerMap["LON"]

	if !okArea || !okLat || !okLon {
		log.Fatalf("Required columns for geospatial imputation not found in header")
	}

	latData := make(map[string][]float64)
	lonData := make(map[string][]float64)

	var rowCount int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: error reading row %d during Pass 1: %v", rowCount, err)
			continue
		}
		rowCount++

		area := record[areaIdx]
		lat, _ := strconv.ParseFloat(record[latIdx], 64)
		lon, _ := strconv.ParseFloat(record[lonIdx], 64)

		if lat != 0 || lon != 0 {
			latData[area] = append(latData[area], lat)
			lonData[area] = append(lonData[area], lon)
		}
	}

	medians := make(map[string]Coord)
	for area := range latData {
		medians[area] = Coord{
			Lat: median(latData[area]),
			Lon: median(lonData[area]),
		}
	}

	// ---------------------------------------------------------
	// PASS 2: Cleaning Data and Feature Engineering
	// ---------------------------------------------------------
	log.Println("Pass 2: Cleaning data, generating features, and writing output...")
	
	// Rewind the file back to the start
	_, err = inFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalf("Failed to seek input file: %v", err)
	}
	
	reader = csv.NewReader(inFile)
	reader.ReuseRecord = true

	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	headers, err = reader.Read() // Read original header again
	if err != nil {
		log.Fatalf("Failed to read header on pass 2: %v", err)
	}

	var outputHeaders []string
	var colIndices []int
	outIdxMap := make(map[string]int)

	// Add the preserved columns to the output and register their original indices
	for i, h := range headers {
		if !columnsToRemove[h] {
			outIdxMap[h] = len(colIndices)
			colIndices = append(colIndices, i)
			outputHeaders = append(outputHeaders, toSnakeCase(h))
		}
	}

	// Add the new engineered columns to the end
	outputHeaders = append(outputHeaders, "year", "month", "day_of_week", "days_to_report", "hour", "victim_identified")

	if err := writer.Write(outputHeaders); err != nil {
		log.Fatalf("Failed to write header: %v", err)
	}

	rowCount = 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: error reading row %d during Pass 2: %v", rowCount, err)
			continue
		}
		rowCount++

		// Initialize row output slice with preserved columns
		outRecord := make([]string, len(colIndices))
		for outI, origI := range colIndices {
			outRecord[outI] = record[origI]
		}

		// Helper func to get original column values easily
		origVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		dateRptdStr := origVal("Date Rptd")
		dateOccStr := origVal("DATE OCC")
		timeOccStr := origVal("TIME OCC")
		victAgeStr := origVal("Vict Age")
		victSexStr := origVal("Vict Sex")
		victDescStr := origVal("Vict Descent")
		weaponDescStr := origVal("Weapon Desc")
		locationStr := origVal("LOCATION")
		areaStr := origVal("AREA NAME")
		latStr := origVal("LAT")
		lonStr := origVal("LON")

		// 2. Date conversion and Feature Engineering
		var year, month, dayOfWeek, daysToReport, hour string

		dateOcc, errOcc := parseDate(dateOccStr)
		if errOcc == nil {
			year = fmt.Sprintf("%d", dateOcc.Year())
			month = fmt.Sprintf("%d", int(dateOcc.Month()))
			dayOfWeek = dateOcc.Weekday().String()
			
			// Standardize Date format to ISO implicitly
			if idx, ok := outIdxMap["DATE OCC"]; ok {
				outRecord[idx] = dateOcc.Format("2006-01-02 15:04:05")
			}
		}

		dateRptd, errRptd := parseDate(dateRptdStr)
		if errOcc == nil && errRptd == nil {
			daysRpt := dateRptd.Truncate(24 * time.Hour)
			daysOcc := dateOcc.Truncate(24 * time.Hour)
			diff := daysRpt.Sub(daysOcc).Hours() / 24.0
			daysToReport = fmt.Sprintf("%.0f", diff)
			
			if idx, ok := outIdxMap["Date Rptd"]; ok {
				outRecord[idx] = dateRptd.Format("2006-01-02 15:04:05")
			}
		}

		if timeOccStr != "" {
			timeVal, err := strconv.Atoi(timeOccStr)
			if err == nil {
				h := timeVal / 100
				hour = fmt.Sprintf("%d", h)
			}
		}

		// 3. Sentinel / Null Values Treatment
		origAgeVal := 0
		ageVal, err := strconv.Atoi(victAgeStr)
		if err == nil {
			origAgeVal = ageVal
			if ageVal <= 0 || ageVal > 99 {
				if idx, ok := outIdxMap["Vict Age"]; ok {
					outRecord[idx] = ""
				}
			}
		} else {
			if idx, ok := outIdxMap["Vict Age"]; ok {
				outRecord[idx] = ""
			}
		}

		vs := strings.ToUpper(strings.TrimSpace(victSexStr))
		if vs != "M" && vs != "F" {
			if idx, ok := outIdxMap["Vict Sex"]; ok {
				outRecord[idx] = "X"
			}
		}

		if victDescStr == "X" {
			if idx, ok := outIdxMap["Vict Descent"]; ok {
				outRecord[idx] = ""
			}
		}

		if weaponDescStr == "" {
			if idx, ok := outIdxMap["Weapon Desc"]; ok {
				outRecord[idx] = "NO WEAPON"
			}
		}

		// 4. String Cleaning
		if locationStr != "" {
			cleanLoc := strings.TrimSpace(locationStr)
			cleanLoc = multiSpaceRegex.ReplaceAllString(cleanLoc, " ")
			if idx, ok := outIdxMap["LOCATION"]; ok {
				outRecord[idx] = cleanLoc
			}
		}

		// 5. Geospatial Imputation
		lat, _ := strconv.ParseFloat(latStr, 64)
		lon, _ := strconv.ParseFloat(lonStr, 64)
		if lat == 0 && lon == 0 {
			if medianCoords, ok := medians[areaStr]; ok {
				if idx, ok := outIdxMap["LAT"]; ok {
					outRecord[idx] = strconv.FormatFloat(medianCoords.Lat, 'f', -1, 64)
				}
				if idx, ok := outIdxMap["LON"]; ok {
					outRecord[idx] = strconv.FormatFloat(medianCoords.Lon, 'f', -1, 64)
				}
			}
		}

		// 6. Additional Feature Engineering
		victimIdentified := "true"
		sexEmptyOrX := (victSexStr == "" || victSexStr == "X" || victSexStr == "-")
		descEmptyOrX := (victDescStr == "" || victDescStr == "X")

		if origAgeVal <= 0 && sexEmptyOrX && descEmptyOrX {
			victimIdentified = "false"
		}

		// Append engineered columns
		outRecord = append(outRecord, year, month, dayOfWeek, daysToReport, hour, victimIdentified)

		if err := writer.Write(outRecord); err != nil {
			log.Fatalf("Failed to write record: %v", err)
		}

		if rowCount%100000 == 0 {
			log.Printf("Processed %d rows...", rowCount)
		}
	}

	log.Printf("Successfully completed processing %d rows. Output saved at %s", rowCount, outputFile)
}
