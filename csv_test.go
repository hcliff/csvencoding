package csvencoding_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCSV(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSV Suite")
}

// type with custom csv getters/setters
type csvgetter map[string]string

func (l *csvgetter) SetCSV(b []string) error {
	*l = map[string]string{"set": "csv"}
	return nil
}

func (l csvgetter) GetCSV() ([]string, error) {
	return []string{"getcsv"}, nil
}
