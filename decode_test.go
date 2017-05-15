package csvencoding_test

import (
	"encoding/csv"
	"strings"
	"time"

	"github.com/hcliff/csvencoding"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func reader(input string) *csv.Reader {
	r := csv.NewReader(strings.NewReader(input))
	// csv spec technically calls for double quoting strings.
	// 99.99% of the time this doesn't matter
	r.LazyQuotes = true
	return r
}

func decode(input string, output interface{}) error {
	decoder := csvencoding.NewDecoder(reader(input))
	return decoder.Decode(output)
}

var _ = Describe("CSV Decoding", func() {
	var err error

	Describe("Cell Values", func() {
		It("Should populate nested paths", func() {
			values := &csvencoding.CellValues{}
			values.Set("my.nested.struct", "henry")
			Ω(values).Should(Equal(
				&csvencoding.CellValues{
					"my": &csvencoding.CellValues{
						"nested": &csvencoding.CellValues{
							"struct": "henry",
						},
					},
				},
			))
		})
	})

	It("Should decode basic types", func() {
		input := "string,int,bool,float\nhenry,23,true,60.429"
		output := struct {
			String string
			Int    int
			Bool   bool
			Float  float64
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.String).Should(Equal("henry"))
		Ω(output.Int).Should(Equal(23))
		Ω(output.Bool).Should(Equal(true))
		Ω(output.Float).Should(Equal(float64(60.429)))
	})

	It("Should decode pointers", func() {
		input := "string,int,bool,float\nhenry,23,true,60.429"
		output := struct {
			String *string
			Int    *int
			Bool   *bool
			Float  *float64
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(*output.String).Should(Equal("henry"))
		Ω(*output.Int).Should(Equal(23))
		Ω(*output.Bool).Should(Equal(true))
		Ω(*output.Float).Should(Equal(float64(60.429)))
	})

	It("Should decode arrays", func() {
		input := "strings,ints,bools,floats,pstrings\n\"vin,diesel\",\"23,24\",\"true,false\",\"60.429,50.534\",vin"
		output := struct {
			Strings  []string
			Ints     []int
			Bools    []bool
			Floats   []float64
			PStrings []*string
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.Strings).Should(ConsistOf("vin", "diesel"))
		Ω(output.Ints).Should(ConsistOf(23, 24))
		Ω(output.Bools).Should(ConsistOf(true, false))
		Ω(output.Floats).Should(ConsistOf(float64(60.429), float64(50.534)))
		pString := new(string)
		*pString = "vin"
		Ω(output.PStrings).Should(ConsistOf(pString))
	})

	It("should decode time", func() {
		input := "time\n2000-10-09T08:07:06.000000005Z\n"
		output := struct {
			Time time.Time
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		expectedOutput := time.Date(2000, 10, 9, 8, 7, 6, 5, time.UTC)
		Ω(output.Time).Should(BeTemporally("==", expectedOutput))
	})

	It("Should decode custom types", func() {
		input := "json\nhello world"
		output := struct {
			// json struct defined with custom GetCSV method
			Json csvgetter
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.Json).Should(Equal(csvgetter{"set": "csv"}))
	})

	It("Should use custom empty & null types", func() {
		input := "name,age\nVIN,IMMORTAL"
		output := struct {
			Name string
			Age  *int
		}{}
		decoder := csvencoding.NewDecoder(reader(input))
		decoder.EmptyValue = "VIN"
		decoder.NilValue = "IMMORTAL"
		Ω(output.Name).Should(Equal(""))
		Ω(output.Age).Should(BeNil())
	})

	It("should use field name tags", func() {
		input := "handle\nhenry"
		output := struct {
			Name string `csv:"handle"`
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.Name).Should(Equal("henry"))
	})

	It("should skip unexported fields", func() {
		input := "name\nvin\n"
		output := struct {
			private       string
			PublicSkipped string `csv:"-"`
			Name          string
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.Name).Should(Equal("vin"))
	})

	It("should populate exported embedded structs", func() {
		input := "name\nhenry"
		type AnonymousStruct struct {
			Name string
		}
		output := struct {
			AnonymousStruct
		}{}
		err = decode(input, &output)
		Ω(err).Should(BeNil())
		Ω(output.Name).Should(Equal("henry"))
	})

	Context("Nested structs", func() {
		input := "person.name\nhenry"
		type personStruct struct {
			Name string
		}

		It("should populate child fields", func() {
			output := struct {
				Person personStruct
			}{}
			err = decode(input, &output)
			Ω(err).Should(BeNil())
			Ω(output.Person.Name).Should(Equal("henry"))
		})

		It("should support nested pointer fields", func() {
			output := struct {
				Person *personStruct
			}{}
			err = decode(input, &output)
			Ω(err).Should(BeNil())
			Ω(output.Person).ShouldNot(BeNil())
			Ω(output.Person.Name).Should(Equal("henry"))
		})
	})
})
