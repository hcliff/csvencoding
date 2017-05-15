package csvencoding

import (
	"encoding"
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Decoder struct {
	r      *csv.Reader
	header []string
	err    error
	// A cell value that translates to the types zero value
	EmptyValue string
	// A cell value that translates to null
	NilValue string
}

func (d Decoder) Header() []string {
	return d.header
}

func NewDecoder(r *csv.Reader) *Decoder {
	header, err := r.Read()
	return &Decoder{
		r:          r,
		header:     header,
		err:        err,
		EmptyValue: DefaultEmptyValue,
		NilValue:   DefaultNilValue,
	}
}

type Setter interface {
	SetCSV([]string) error
}

func indirectSetter(v reflect.Value) Setter {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	// If this value implements Setter return it
	if u, ok := v.Interface().(Setter); ok {
		return u
	}
	return nil
}

func indirectTextUnmarshaler(v reflect.Value) encoding.TextUnmarshaler {

	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
		return u
	}

	return nil
}

func (dec *Decoder) readStringTo(field reflect.Value, value string) (err error) {
	if value == dec.NilValue {
		return nil
	}

	// Handle pointers
	if field.Kind() == reflect.Ptr {
		// Instantiate a pointer to the correct underlying type
		elem := reflect.New(field.Type().Elem())
		// Assign said pointer to field
		field.Set(elem)
		field = elem.Elem()
	}

	reflectType := field.Type()

	// This comes after the pointer so if the value is empty but not null
	// a pointer with an empty value is instantiated
	if value == dec.EmptyValue {
		return nil
	}

	// Handle custom csv methods
	if setter := indirectSetter(field); setter != nil {
		if err := setter.SetCSV([]string{value}); err != nil {
			return err
		}
		return nil
	}

	if textUnmarshaler := indirectTextUnmarshaler(field); textUnmarshaler != nil {
		if err := textUnmarshaler.UnmarshalText([]byte(value)); err != nil {
			return err
		}
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int:
		i, err := strconv.ParseInt(value, 0, 0)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Int8:
		i, err := strconv.ParseInt(value, 0, 8)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Int16:
		i, err := strconv.ParseInt(value, 0, 16)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Int32:
		i, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Int64:
		i, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Float32:
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Slice:
		values := strings.Split(value, ",")
		sliceValue := reflect.MakeSlice(reflectType, len(values), len(values))

		for i, value := range values {
			if err := dec.readStringTo(sliceValue.Index(i), value); err != nil {
				return err
			}
		}

		// Only set once everything succesfully parsed
		field.Set(sliceValue)
	default:
		return fmt.Errorf("Can't unmarshal %s from csv", field.Type().String())
	}

	return nil
}

func (dec *Decoder) readCellValuesTo(field reflect.Value, value *CellValues) (err error) {

	// Handle pointers
	if field.Kind() == reflect.Ptr {
		// Instantiate a pointer to the correct underlying type
		elem := reflect.New(field.Type().Elem())
		// Assign said pointer to field
		field.Set(elem)
		field = elem.Elem()
	}

	// For now we only support decoding to struts, this could easily be fied
	switch field.Kind() {
	case reflect.Struct:
		if err := dec.readStructTo(field, value); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Can't unmarshal %s from csv", field.Type().String())
	}
	if dec.err != nil {
		return dec.err
	}
	return nil
}

func (dec *Decoder) readStructTo(reflectValue reflect.Value, values *CellValues) (err error) {

	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	reflectType := reflectValue.Type()

	for i := 0; i < reflectType.NumField(); i++ {
		fieldType := reflectType.Field(i)
		fieldValue := reflectValue.Field(i)

		key := strings.Split(fieldType.Tag.Get("csv"), ",")
		fieldName := key[0]
		// PkgPath == "" and !Anonymous for unexported fields
		if key[0] == "-" || (fieldType.PkgPath != "" && !fieldType.Anonymous) {
			continue
		}
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}
		if fieldType.Anonymous {
			if err := dec.readStructTo(fieldValue, values); err != nil {
				return err
			}
		} else if cell, ok := values.Get(fieldName); ok {
			switch cell := cell.(type) {
			case *CellValues:
				if err := dec.readCellValuesTo(fieldValue, cell); err != nil {
					return err
				}
			case string:
				if err := dec.readStringTo(fieldValue, cell); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unexpected type %T\n", cell)
			}

		}
	}

	return nil
}

// Recursive struct, where the value is either a string
// or another CellValues map
type CellValues map[string]interface{}

// Helper method, CellValues is only used as a pointer
// (which do not support key lookup)
// this allows code not to have to deference its CellValues pointer
func (vs CellValues) Get(key string) (interface{}, bool) {
	value, ok := vs[key]
	return value, ok
}

// Recursivly pops one part off the key and has
// its child handle the tail
func (vs CellValues) Set(key, value string) {
	parts := strings.SplitN(key, ".", 2)
	headKey := parts[0]
	// This is the terminus of the key
	if len(parts) == 1 {
		// Set the concrete string value from the csv
		vs[headKey] = value
	} else {
		tail := parts[1]
		// If this is the first visit to this child, populate
		// it with a CellValue struct
		if _, ok := vs[headKey]; !ok {
			vs[headKey] = &CellValues{}
		}
		// Cast our value to cellValue and recurse down the key
		vs[headKey].(*CellValues).Set(tail, value)
	}
}

func (dec *Decoder) Decode(i interface{}) error {
	if dec.err != nil {
		return dec.err
	}

	// fetch the next csv row
	var r []string
	if r, dec.err = dec.r.Read(); dec.err != nil {
		return dec.err
	}

	// Construct a key:value map
	// based upon the header
	m := &CellValues{}
	for i, key := range dec.header {
		m.Set(key, r[i])
	}

	reflectValue := reflect.ValueOf(i)

	// Decoder only handles root structs for now
	if err := dec.readStructTo(reflectValue, m); err != nil {
		dec.err = err
	}

	return dec.err
}
