package csvencoding_test

import (
	"bytes"
	"encoding/csv"
	"time"

	"github.com/hcliff/csvencoding"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const HIS_NAME_IS = "robert paulson"

var _ = Describe("CSV Encoding", func() {

	var err error
	var b bytes.Buffer
	var encoder *csvencoding.Encoder

	BeforeEach(func() {
		b.Reset()
		w := csv.NewWriter(&b)
		// Write out to our test buffer
		encoder = csvencoding.NewEncoder(w)
	})

	It("should encode basic data types", func() {
		input := struct {
			String string
			Int    int
			Bool   bool
			Float  float64
		}{"henry", 23, true, float64(60.429)}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "henry,23,true,60.429\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode pointers", func() {
		stringP := new(string)
		*stringP = "henry"
		intP := new(int)
		*intP = 23
		boolP := new(bool)
		*boolP = true
		floatP := new(float64)
		*floatP = 60.429
		type embedded struct {
			First  string
			Second string
		}
		input := struct {
			String *string
			Int    *int
			Bool   *bool
			Float  *float64
		}{stringP, intP, boolP, floatP}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "henry,23,true,60.429\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode slices", func() {
		stringP := new(string)
		*stringP = "vin"
		input := struct {
			Strings  []string
			Ints     []int
			Bools    []bool
			Floats   []float64
			Pstrings []*string
		}{
			[]string{"vin", "diesel"},
			[]int{23, 24},
			[]bool{true, false},
			[]float64{60.429, 50.534},
			[]*string{stringP},
		}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "\"vin,diesel\",\"23,24\",\"true,false\",\"60.429,50.534\",vin\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode structs", func() {
		type ChildStruct struct {
			Names []string
		}

		stepChild := ChildStruct{[]string{"wut"}}

		input := struct {
			Child     ChildStruct
			StepChild *ChildStruct
		}{
			Child:     ChildStruct{[]string{"uno", "dos"}},
			StepChild: &stepChild,
		}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "\"uno,dos\",wut\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode time", func() {
		twoThousand := time.Date(2000, 10, 9, 8, 7, 6, 5, time.UTC)
		input := struct {
			Time time.Time
		}{
			twoThousand,
		}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "2000-10-09T08:07:06.000000005Z\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode nil structs", func() {
		type ChildStruct struct {
			Names []string
			Ages  []int
		}
		input := struct {
			Child *ChildStruct
		}{nil}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		input = struct {
			Child *ChildStruct
		}{nil}
		input = struct {
			Child *ChildStruct
		}{&ChildStruct{
			Names: []string{"vin"},
			// surprising I know
			Ages: []int{47},
		}}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "NULL,NULL\nvin,47\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should encode custom types", func() {
		input := struct {
			Json csvgetter
		}{csvgetter{"name": "henry"}}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "getcsv\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("Should use custom empty & null types", func() {
		input := struct {
			Name string `csv:",omitEmpty"`
			Age  *int
		}{Name: ""}
		encoder.EmptyValue = "VIN"
		encoder.NilValue = "IMMORTAL"
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "VIN,IMMORTAL\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should skip unexported fields", func() {
		input := struct {
			private       string
			PublicSkipped string `csv:"-"`
			Name          string
		}{"riddick", "dom", "vin"}
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		expectedOutput := "vin\n"
		Ω(b.String()).Should(Equal(expectedOutput))
	})

	It("should populate exported embedded annonymous structs", func() {
		type AnonymousStruct struct {
			Name string
		}
		input := struct {
			AnonymousStruct
		}{AnonymousStruct{"riddick"}}
		expectedOutput := "riddick\n"
		err = encoder.Encode(input)
		Ω(err).Should(BeNil())
		Ω(b.String()).Should(Equal(expectedOutput))
	})

})
