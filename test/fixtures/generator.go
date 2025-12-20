package fixtures

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// Generator generates test CSV files
type Generator struct {
	outputDir string
	rand      *rand.Rand
}

// NewGenerator creates a new test data generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateSimple generates a simple CSV file
func (g *Generator) GenerateSimple(filename string, rows int) (string, error) {
	path := filepath.Join(g.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"id", "name", "value"}); err != nil {
		return "", err
	}

	// Write rows
	for i := 0; i < rows; i++ {
		record := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("name_%d", i+1),
			fmt.Sprintf("%d", g.rand.Intn(1000)),
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	return path, nil
}

// GenerateWithErrors generates a CSV file with intentional errors
func (g *Generator) GenerateWithErrors(filename string, rows int, errorRate float64) (string, error) {
	path := filepath.Join(g.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"id", "name", "age", "email"}); err != nil {
		return "", err
	}

	// Write rows with occasional errors
	for i := 0; i < rows; i++ {
		var record []string

		// Inject errors based on error rate
		if g.rand.Float64() < errorRate {
			// Generate error: missing field, wrong type, etc.
			errorType := g.rand.Intn(3)
			switch errorType {
			case 0: // Missing field
				record = []string{
					fmt.Sprintf("%d", i+1),
					fmt.Sprintf("name_%d", i+1),
				}
			case 1: // Wrong type (text in age field)
				record = []string{
					fmt.Sprintf("%d", i+1),
					fmt.Sprintf("name_%d", i+1),
					"invalid_age",
					fmt.Sprintf("user%d@example.com", i+1),
				}
			case 2: // Extra field
				record = []string{
					fmt.Sprintf("%d", i+1),
					fmt.Sprintf("name_%d", i+1),
					fmt.Sprintf("%d", 20+g.rand.Intn(50)),
					fmt.Sprintf("user%d@example.com", i+1),
					"extra_field",
				}
			}
		} else {
			// Valid record
			record = []string{
				fmt.Sprintf("%d", i+1),
				fmt.Sprintf("name_%d", i+1),
				fmt.Sprintf("%d", 20+g.rand.Intn(50)),
				fmt.Sprintf("user%d@example.com", i+1),
			}
		}

		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	return path, nil
}

// GenerateLarge generates a large CSV file for performance testing
func (g *Generator) GenerateLarge(filename string, rows int) (string, error) {
	path := filepath.Join(g.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"id", "timestamp", "user_id", "product_id", "quantity", "price", "total", "status"}
	if err := writer.Write(headers); err != nil {
		return "", err
	}

	// Write rows
	for i := 0; i < rows; i++ {
		price := float64(g.rand.Intn(10000)) / 100.0
		quantity := g.rand.Intn(10) + 1
		total := price * float64(quantity)

		record := []string{
			fmt.Sprintf("%d", i+1),
			time.Now().Add(-time.Duration(g.rand.Intn(86400)) * time.Second).Format(time.RFC3339),
			fmt.Sprintf("user_%d", g.rand.Intn(1000)),
			fmt.Sprintf("prod_%d", g.rand.Intn(100)),
			fmt.Sprintf("%d", quantity),
			fmt.Sprintf("%.2f", price),
			fmt.Sprintf("%.2f", total),
			randomStatus(g.rand),
		}

		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	return path, nil
}

// GenerateMultiple generates multiple CSV files
func (g *Generator) GenerateMultiple(prefix string, count int, rowsPerFile int) ([]string, error) {
	var files []string

	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("%s_%d.csv", prefix, i+1)
		path, err := g.GenerateSimple(filename, rowsPerFile)
		if err != nil {
			return nil, err
		}
		files = append(files, path)
	}

	return files, nil
}

// GenerateWithDifferentHeaders generates files with inconsistent headers
func (g *Generator) GenerateWithDifferentHeaders(dir string) ([]string, error) {
	files := []string{}

	// File 1: Standard headers
	file1 := filepath.Join(dir, "file1.csv")
	f1, err := os.Create(file1)
	if err != nil {
		return nil, err
	}

	w1 := csv.NewWriter(f1)
	w1.Write([]string{"id", "name", "value"})
	w1.Write([]string{"1", "Alice", "100"})
	w1.Flush()
	f1.Close()
	files = append(files, file1)

	// File 2: Different headers (should fail validation)
	file2 := filepath.Join(dir, "file2.csv")
	f2, err := os.Create(file2)
	if err != nil {
		return nil, err
	}

	w2 := csv.NewWriter(f2)
	w2.Write([]string{"id", "username", "amount"}) // Different!
	w2.Write([]string{"2", "Bob", "200"})
	w2.Flush()
	f2.Close()
	files = append(files, file2)

	return files, nil
}

// GenerateEmpty generates an empty CSV file
func (g *Generator) GenerateEmpty(filename string) (string, error) {
	path := filepath.Join(g.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	file.Close()

	return path, nil
}

// GenerateHeaderOnly generates a CSV file with only a header
func (g *Generator) GenerateHeaderOnly(filename string) (string, error) {
	path := filepath.Join(g.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write only header
	if err := writer.Write([]string{"id", "name", "value"}); err != nil {
		return "", err
	}

	return path, nil
}

// randomStatus returns a random status string
func randomStatus(r *rand.Rand) string {
	statuses := []string{"pending", "completed", "cancelled", "refunded"}
	return statuses[r.Intn(len(statuses))]
}

// CleanupFiles removes generated test files
func CleanupFiles(files ...string) error {
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
