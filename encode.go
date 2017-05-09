package csvencoding

import (
	"encoding"
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Encoder struct {
	w   *csv.Writer
	err error
	// A cell value to be used when omitempty is specified on a reflectValue
	EmptyValue string
	// A cell value to be used for nil values
	NilValue string
}

func NewEncoder(w *csv.Writer) *Encoder {
	return &Encoder{
		w:          w,
		EmptyValue: DefaultEmptyValue,
		NilValue:   DefaultNilValue,
	}
}

type Getter interface {
	GetCSV() ([]string, error)
}

func indirectGetter(v reflect.Value) Getter {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	// If this value implements Getter return it
	if u, ok := v.Interface().(Getter); ok {
		return u
	}

	return nil
}

func indirectTextMarshaler(v reflect.Value) encoding.TextMarshaler {

	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}

	if u, ok := v.Interface().(encoding.TextMarshaler); ok {
		return u
	}

	return nil
}

func (enc Encoder) marshal(reflectValue reflect.Value, omitEmpty bool) (s []string, err error) {

	if getter := indirectGetter(reflectValue); getter != nil {
		return getter.GetCSV()
	}

	if textMarshaler := indirectTextMarshaler(reflectValue); textMarshaler != nil {
		b, err := textMarshaler.MarshalText()
		if err != nil {
			return nil, err
		}
		return []string{string(b)}, nil
	}

	if reflectValue.Kind() == reflect.Ptr {
		if reflectValue.IsNil() {
			switch e := reflectValue.Type().Elem(); e.Kind() {
			case reflect.Struct:
				output := make([]string, 0, e.NumField())
				for i := 0; i < e.NumField(); i++ {
					if tmp := e.Field(i); tmp.PkgPath == "" {
						output = append(output, enc.NilValue)
					}
				}
				return output, nil
			default:
				return []string{enc.NilValue}, nil
			}
		}
		reflectValue = reflectValue.Elem()
	}

	reflectType := reflectValue.Type()

	valueInterface := reflectValue.Interface()
	zero := reflect.Zero(reflectType).Interface()
	if reflect.DeepEqual(valueInterface, zero) && omitEmpty {
		return []string{enc.EmptyValue}, nil
	}

	switch reflectValue.Kind() {
	case reflect.Bool:
		return []string{strconv.FormatBool(reflectValue.Bool())}, nil

	case reflect.String:
		return []string{reflectValue.String()}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{strconv.FormatInt(reflectValue.Int(), 10)}, nil

	case reflect.Float32, reflect.Float64:
		return []string{strconv.FormatFloat(reflectValue.Float(), 'f', -1, 64)}, nil

	case reflect.Slice:
		output := make([]string, reflectValue.Len())
		for i := 0; i < reflectValue.Len(); i++ {
			// Index retrieves an element at a specific index (returns a reflect.Value)
			elementValue := reflectValue.Index(i)
			// Recurse down, this would let us handle multi dimensional arrays etc...
			elementOutput, err := enc.marshal(elementValue, false)
			if err != nil {
				// Interface() returns the concrete value as an interface
				// (the original value we put in)
				err = fmt.Errorf("slice element `%v`: %s", elementValue.Interface(), err.Error())
				return nil, err
			}
			output[i] = strings.Join(elementOutput, ",")
		}
		// Slices can be variable length
		// this presents a problem as two csv rows may have slices of different lengths
		// if we encoded each element as a column this would give our two csv rows
		// different lengths. this isn't typically desireable
		return []string{strings.Join(output, ",")}, nil

	case reflect.Map:
		output := make([]string, 0, reflectValue.Len())
		for _, keyValue := range reflectValue.MapKeys() {
			keyOutput, err := enc.marshal(keyValue, false)
			if err != nil {
				// Interface() returns the concrete value as an interface
				// (the original value we put in)
				err = fmt.Errorf("map key `%v`: %s", keyValue.Interface(), err.Error())
				return nil, err
			}
			// map keys can be anything comparable (including structs)
			// so it is possible we have multiple values
			keyStr := strings.Join(keyOutput, ",")

			// Index retrieves an element at a specific index (returns a reflect.Value)
			elementValue := reflectValue.MapIndex(keyValue)
			// Recurse down, this would let us handle multi dimensional arrays etc...
			valueOutput, err := enc.marshal(elementValue, false)
			if err != nil {
				// Interface() returns the concrete value as an interface
				// (the original value we put in)
				err = fmt.Errorf("map value `%v`: %s", elementValue.Interface(), err.Error())
				return nil, err
			}
			valueStr := strings.Join(valueOutput, ",")

			output = append(output, keyStr+":"+valueStr)
		}
		// See slice reasoning
		return []string{strings.Join(output, ",")}, nil

	case reflect.Struct:
		output := []string{}
		// NumField includes unexported fields
		for i := 0; i < reflectType.NumField(); i++ {
			field := reflectType.Field(i)

			key := strings.Split(field.Tag.Get("csv"), ",")
			fieldName := key[0]
			// csv:",omitEmpty"
			omitEmpty := len(key) > 1 && key[1] == "omitEmpty"

			// PkgPath == "" and !Anonymous for unexported reflectValues
			if fieldName == "-" || (field.PkgPath != "" && !field.Anonymous) {
				continue
			}

			fieldValue := reflectValue.Field(i)
			fieldOutput, err := enc.marshal(fieldValue, omitEmpty)
			if err != nil {
				err = fmt.Errorf("struct field `%s`: `%v`: %s", field.Name, fieldValue.Interface(), err.Error())
				return nil, err
			}
			output = append(output, fieldOutput...)
		}
		return output, nil

	default:
		return nil, fmt.Errorf("Can't enc.marshal %+v from csv %T", valueInterface, valueInterface)
	}
}

func (enc Encoder) Encode(i interface{}) error {
	if enc.err != nil {
		return enc.err
	}

	reflectValue := reflect.ValueOf(i)

	output, err := enc.marshal(reflectValue, false)
	if err != nil {
		enc.err = err
		return enc.err
	}

	enc.err = enc.w.Write(output)
	enc.w.Flush()

	return enc.err
}
