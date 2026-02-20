package health

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"voidline/internal/infrastructure/persistence/sqlite"

	healthdomain "voidline/internal/domain/health"
)

type Service struct {
	repo *sqlite.HealthRepository
}

func NewService(repo *sqlite.HealthRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	DryRun bool
	Force  bool
}

func (s *Service) ImportWithingsActivities(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/activities.csv"
	return s.importWithingsActivitiesFromPath(ctx, path, options)
}

func (s *Service) importWithingsActivitiesFromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping Withings activities: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing Withings activities...")
	start := time.Now()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read CSV header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var metrics []*healthdomain.HealthMetric
	lineNum := 1

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		lineNum++

		timestamp := s.getField(record, colMap, "from")
		if timestamp == "" {
			continue
		}

		dataStr := s.getField(record, colMap, "data")
		data := parseJSONData(dataStr)

		metricTypes := []struct {
			key       string
			unit      string
			extractor func(m map[string]interface{}) interface{}
		}{
			{key: "steps", unit: healthdomain.UnitCount, extractor: func(m map[string]interface{}) interface{} { return m["steps"] }},
			{key: "calories", unit: healthdomain.UnitKcal, extractor: func(m map[string]interface{}) interface{} { return m["calories"] }},
			{key: "distance", unit: healthdomain.UnitMeters, extractor: func(m map[string]interface{}) interface{} { return m["distance"] }},
			{key: "elevation", unit: healthdomain.UnitMeters, extractor: func(m map[string]interface{}) interface{} { v, _ := m["elevation"]; return v }},
			{key: "hr_average", unit: healthdomain.UnitBpm, extractor: func(m map[string]interface{}) interface{} { return m["hr_average"] }},
			{key: "hr_min", unit: healthdomain.UnitBpm, extractor: func(m map[string]interface{}) interface{} { return m["hr_min"] }},
			{key: "hr_max", unit: healthdomain.UnitBpm, extractor: func(m map[string]interface{}) interface{} { return m["hr_max"] }},
		}

		for _, mt := range metricTypes {
			val := mt.extractor(data)
			if val == nil {
				continue
			}

			floatVal, ok := toFloat64(val)
			if !ok {
				continue
			}

			metric := &healthdomain.HealthMetric{
				Timestamp:  timestamp,
				Platform:   healthdomain.PlatformWithings,
				MetricType: mt.key,
				Value:      floatVal,
				Unit:       mt.unit,
				SourceFile: "activities.csv",
			}

			if !options.Force {
				exists, err := s.repo.Exists(ctx, metric)
				if err != nil {
					result.Errors = append(result.Errors, healthdomain.ImportError{
						Row: lineNum,
						Err: fmt.Errorf("duplicate check failed: %w", err),
					})
					continue
				}
				if exists {
					result.Skipped++
					continue
				}
			}

			metrics = append(metrics, metric)
			result.TotalRows++
		}
	}

	if !options.DryRun && len(metrics) > 0 {
		if err := s.repo.CreateBatch(ctx, metrics); err != nil {
			return result, fmt.Errorf("failed to insert metrics: %w", err)
		}
	}

	result.Inserted = len(metrics)
	result.Duration = time.Since(start)
	fmt.Printf("Imported %d Withings activity records (skipped: %d).\n", result.Inserted, result.Skipped)

	return result, nil
}

func (s *Service) ImportSpO2(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/manual_spo2.csv"
	return s.importSpO2FromPath(ctx, path, options)
}

func (s *Service) importSpO2FromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping SpO2: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing Withings SpO2...")
	start := time.Now()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read CSV header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var metrics []*healthdomain.HealthMetric

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		timestamp := s.getField(record, colMap, "date")
		valueStr := s.getField(record, colMap, "value")

		if timestamp == "" || valueStr == "" {
			continue
		}

		floatVal, ok := toFloat64(valueStr)
		if !ok {
			continue
		}

		ts, err := time.Parse("2006-01-02 15:04:05", timestamp)
		if err != nil {
			ts, _ = time.Parse("2006-01-02", timestamp)
		}
		tsStr := ts.Format(time.RFC3339)

		metric := &healthdomain.HealthMetric{
			Timestamp:  tsStr,
			Platform:   healthdomain.PlatformWithings,
			MetricType: healthdomain.MetricTypeSpO2,
			Value:      floatVal,
			Unit:       healthdomain.UnitPercent,
			SourceFile: "manual_spo2.csv",
		}

		if !options.Force {
			exists, err := s.repo.Exists(ctx, metric)
			if err != nil {
				result.Errors = append(result.Errors, healthdomain.ImportError{Row: result.TotalRows + 1, Err: fmt.Errorf("duplicate check failed: %w", err)})
				continue
			}
			if exists {
				result.Skipped++
				continue
			}
		}

		metrics = append(metrics, metric)
		result.TotalRows++
	}

	if !options.DryRun && len(metrics) > 0 {
		if err := s.repo.CreateBatch(ctx, metrics); err != nil {
			return result, fmt.Errorf("failed to insert metrics: %w", err)
		}
	}

	result.Inserted = len(metrics)
	result.Duration = time.Since(start)
	fmt.Printf("Imported %d SpO2 records (skipped: %d).\n", result.Inserted, result.Skipped)

	return result, nil
}

func (s *Service) ImportMFPWeight(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	fmt.Println("MFP weight import requires Excel support - skipping for now.")
	return &healthdomain.ImportResult{}, nil
}

func (s *Service) getField(record []string, colMap map[string]int, keys ...string) string {
	for _, key := range keys {
		if idx, ok := colMap[key]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
	}
	return ""
}

func parseJSONData(dataStr string) map[string]interface{} {
	if dataStr == "" || dataStr == "{}" {
		return make(map[string]interface{})
	}

	dataStr = strings.Trim(dataStr, "{}")
	parts := strings.Split(dataStr, ",")
	result := make(map[string]interface{})

	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			key := strings.Trim(strings.TrimSpace(kv[0]), `"`)
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				result[key] = f
			} else {
				result[key] = val
			}
		}
	}

	return result
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func (s *Service) ImportWeight(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/weight.csv"
	return s.importWeightFromPath(ctx, path, options)
}

func (s *Service) importWeightFromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping weight: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing weight data...")
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	count := 0
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		weight := &healthdomain.WeightRecord{
			Timestamp:    s.getField(record, colMap, "date"),
			WeightLb:     parseFloat(s.getField(record, colMap, "weight (lb)")),
			FatMassLb:    parseFloat(s.getField(record, colMap, "fat mass (lb)")),
			BoneMassLb:   parseFloat(s.getField(record, colMap, "bone mass (lb)")),
			MuscleMassLb: parseFloat(s.getField(record, colMap, "muscle mass (lb)")),
			HydrationLb:  parseFloat(s.getField(record, colMap, "hydration (lb)")),
			Comments:     s.getField(record, colMap, "comments"),
			Source:       "Withings",
		}

		if !options.DryRun {
			if err := s.repo.InsertWeight(ctx, weight); err != nil {
				continue
			}
		}
		count++
	}

	result.Inserted = count
	fmt.Printf("Imported %d weight records.\n", count)
	return result, nil
}

func (s *Service) ImportSleep(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/sleep.csv"
	return s.importSleepFromPath(ctx, path, options)
}

func (s *Service) importSleepFromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping sleep: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing sleep data...")
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	count := 0
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		sleep := &healthdomain.SleepRecord{
			StartTime:              s.getField(record, colMap, "from"),
			EndTime:                s.getField(record, colMap, "to"),
			LightSleepSeconds:      parseInt(s.getField(record, colMap, "light (s)")),
			DeepSleepSeconds:       parseInt(s.getField(record, colMap, "deep (s)")),
			RemSleepSeconds:        parseInt(s.getField(record, colMap, "rem (s)")),
			AwakeSeconds:           parseInt(s.getField(record, colMap, "awake (s)")),
			WakeUpCount:            parseInt(s.getField(record, colMap, "wake up")),
			DurationToSleepSeconds: parseInt(s.getField(record, colMap, "duration to sleep (s)")),
			DurationToWakeSeconds:  parseInt(s.getField(record, colMap, "duration to wake up (s)")),
			SnoringSeconds:         parseInt(s.getField(record, colMap, "snoring (s)")),
			SnoringEpisodes:        parseInt(s.getField(record, colMap, "snoring episodes")),
			AvgHeartRate:           parseInt(s.getField(record, colMap, "average heart rate")),
			MinHeartRate:           parseInt(s.getField(record, colMap, "heart rate (min)")),
			MaxHeartRate:           parseInt(s.getField(record, colMap, "heart rate (max)")),
			Source:                 "Withings",
		}

		if !options.DryRun {
			if err := s.repo.InsertSleep(ctx, sleep); err != nil {
				continue
			}
		}
		count++
	}

	result.Inserted = count
	fmt.Printf("Imported %d sleep records.\n", count)
	return result, nil
}

func (s *Service) ImportBloodPressure(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/bp.csv"
	return s.importBloodPressureFromPath(ctx, path, options)
}

func (s *Service) importBloodPressureFromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping blood pressure: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing blood pressure data...")
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	count := 0
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		bp := &healthdomain.BloodPressureRecord{
			Timestamp: s.getField(record, colMap, "date"),
			HeartRate: parseInt(s.getField(record, colMap, "heart rate")),
			Systolic:  parseInt(s.getField(record, colMap, "systolic")),
			Diastolic: parseInt(s.getField(record, colMap, "diastolic")),
			Comments:  s.getField(record, colMap, "comments"),
			Source:    "Withings",
		}

		if !options.DryRun {
			if err := s.repo.InsertBloodPressure(ctx, bp); err != nil {
				continue
			}
		}
		count++
	}

	result.Inserted = count
	fmt.Printf("Imported %d blood pressure records.\n", count)
	return result, nil
}

func (s *Service) ImportHeartRate(ctx context.Context, sourceDir string, options ImportOptions) (*healthdomain.ImportResult, error) {
	path := sourceDir + "/withings/raw_hr_hr.csv"
	return s.importHeartRateFromPath(ctx, path, options)
}

func (s *Service) importHeartRateFromPath(ctx context.Context, path string, options ImportOptions) (*healthdomain.ImportResult, error) {
	result := &healthdomain.ImportResult{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Skipping heart rate: %s not found.\n", path)
			return result, nil
		}
		return result, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fmt.Println("Importing heart rate data...")
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to read header: %w", err)
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	count := 0
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		hr := &healthdomain.HeartRateRecord{
			Timestamp:       s.getField(record, colMap, "start"),
			DurationSeconds: s.getField(record, colMap, "duration"),
			BpmValue:        s.getField(record, colMap, "value"),
			Source:          "Withings",
		}

		if !options.DryRun {
			if err := s.repo.InsertHeartRate(ctx, hr); err != nil {
				continue
			}
		}
		count++
	}

	result.Inserted = count
	fmt.Printf("Imported %d heart rate records.\n", count)
	return result, nil
}

func parseFloat(val string) float64 {
	if val == "" {
		return 0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(val string) int {
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}
