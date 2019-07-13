package tagparser

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

var tagRegex = regexp.MustCompile("^(\\w+),ns:(\\w+)$")

type annotatedField struct {
	sField     reflect.StructField
	fieldValue reflect.Value
	namespace  string
	name       string
	regex      *regexp.Regexp
}

func (f annotatedField) getRegex() *regexp.Regexp {
	if f.regex == nil {
		ns := regexp.QuoteMeta(f.namespace)
		name := regexp.QuoteMeta(f.name)

		f.regex = regexp.MustCompile(fmt.Sprintf("^nx:%s:%s\\[(.*)]$", ns, name))
	}

	return f.regex
}

func ParseTags(data interface{}, tags, parentTags []string) {
	t := reflect.TypeOf(data)

	if t.Kind() != reflect.Ptr {
		panic("data must be a pointer")
	}

	v := reflect.ValueOf(data).Elem()
	t = v.Type()

	fields := make([]annotatedField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		value := v.Field(i)

		tag := structField.Tag.Get("nx")
		if len(tag) == 0 {
			continue
		}

		matches := tagRegex.FindStringSubmatch(tag)
		if matches == nil {
			continue
		}

		fields = append(fields, annotatedField{
			sField:     structField,
			fieldValue: value,
			namespace:  matches[2],
			name:       matches[1],
		})
	}

	for _, field := range fields {
		value, err := findValueForField(field, tags)
		if err != nil {
			// no value found - search in parent
			value, err = findValueForField(field, parentTags)

			if err != nil {
				//fmt.Printf("warn: No values for field <%s> found\n", field.sField.Name)
				continue
			}
		}

		if !field.fieldValue.CanSet() {
			panic(fmt.Sprintf("error: Cannot set field <%s>", field.sField.Name))
		}
		field.fieldValue.Set(reflect.ValueOf(value))
	}
}

func findValueForField(field annotatedField, tags []string) (interface{}, error) {
	rex := field.getRegex()

	strValues := make([]string, 0)
	for _, tag := range tags {
		match := rex.FindStringSubmatch(tag)
		if match == nil {
			continue
		}

		strValues = append(strValues, match[1])
	}

	// convert to target type
	fType := field.sField.Type
	fKind := fType.Kind()

	if fKind == reflect.Slice {
		sKind := fType.Elem().Kind()

		if sKind == reflect.String {
			return strValues, nil
		} else if sKind == reflect.Int {
			intValues := make([]int, 0, len(strValues))

			for _, val := range strValues {
				i, err := strconv.Atoi(val)
				if err != nil {
					fmt.Printf("warn: could not parse <%v> as int for field <%v>\n", val, field.sField.Name)
					continue
				}
				intValues = append(intValues, i)
			}

			return intValues, nil
		}
	} else if fKind == reflect.String {
		if len(strValues) == 0 {
			//fmt.Printf("warn: No values available for string field <%s>. Returning empty string.\n", field.sField.Name)
			return "", fmt.Errorf("No value available.")
		}

		return strValues[0], nil
	} else if fKind == reflect.Int {
		if len(strValues) == 0 {
			//fmt.Printf("warn: No values available for int field <%s>. Returning 0.\n", field.sField.Name)
			return 0, fmt.Errorf("No value available.")
		}

		for _, val := range strValues {
			i, err := strconv.Atoi(val)
			if err != nil {
				fmt.Printf("warn: could not parse <%v> as int for field <%v>. Trying next value (if any).\n", val, field.sField.Name)
				continue
			}

			return i, nil
		}

		//fmt.Printf("warn: No values available for int field <%s>. Returning 0.\n", field.sField.Name)
		return 0, fmt.Errorf("No value available.")
	} else if fKind == reflect.Bool {
		if len(strValues) == 0 {
			//fmt.Printf("warn: No values available for bool field <%s>. Returning false.\n", field.sField.Name)
			return false, fmt.Errorf("No value available.")
		}

		for _, val := range strValues {
			b, err := strconv.ParseBool(val)
			if err != nil {
				fmt.Printf("warn: could not parse <%v> as bool for field <%v>. Trying next value (if any).\n", val, field.sField.Name)
				continue
			}

			return b, nil
		}

		//fmt.Printf("warn: No values available for bool field <%s>. Returning false.\n", field.sField.Name)
		return false, fmt.Errorf("No value available.")
	}

	panic(fmt.Sprintf("Unsupported field type <%v>", fKind))
}
