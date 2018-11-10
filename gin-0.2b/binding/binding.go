package binding

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type (
	Binding interface {
		Bind(*http.Request, interface{}) error
	}

	// JSON binding
	jsonBinding struct{}

	// XML binding
	xmlBinding struct{}

	// // form binding
	formBinding struct{}
)

var (
	JSON = jsonBinding{}
	XML  = xmlBinding{}
	Form = formBinding{} // todo
)

func (_ jsonBinding) Bind(req *http.Request, obj interface{}) error {
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(obj); err == nil {
		return Validate(obj)
	} else {
		return err
	}
}

func (_ xmlBinding) Bind(req *http.Request, obj interface{}) error {
	decoder := xml.NewDecoder(req.Body)
	if err := decoder.Decode(obj); err == nil {
		return Validate(obj)
	} else {
		return err
	}
}

func (_ formBinding) Bind(req *http.Request, obj interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}
	return Validate(obj)
}

func mapForm(ptr interface{}, form map[string][]string) error {
	typ := reflect.TypeOf(ptr).Elem()
	formStruct := reflect.ValueOf(ptr).Elem()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		if inputFieldName := typeField.Tag.Get("form"); inputFieldName != "" {
			structField := formStruct.Field(i)
			if !structField.CanSet() {
				continue
			}

			inputValue, exists := form[inputFieldName]
			if !exists {
				continue
			}
			numElems := len(inputValue)
			if structField.Kind() == reflect.Slice && numElems > 0 {
				sliceOf := structField.Type().Elem().Kind()
				slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
				for i := 0; i < numElems; i++ {
					if err := setWithProperType(sliceOf, inputValue[i], slice.Index(i)); err != nil {
						return err
					}
				}
				formStruct.Elem().Field(i).Set(slice)
			} else {
				if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	switch valueKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val == "" {
			val = "0"
		}
		intVal, err := strconv.Atoi(val)
		if err != nil {
			return err
		} else {
			structField.SetInt(int64(intVal))
		}
	case reflect.Bool:
		if val == "" {
			val = "false"
		}
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return err
		} else {
			structField.SetBool(boolVal)
		}
	case reflect.Float32:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		} else {
			structField.SetFloat(floatVal)
		}
	case reflect.Float64:
		if val == "" {
			val = "0.0"
		}
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		} else {
			structField.SetFloat(floatVal)
		}
	case reflect.String:
		structField.SetString(val)
	}
	return nil
}

// Don't pass in pointers to bind to. Can lead to bugs. See:
// https://github.com/codegangsta/martini-contrib/issues/40
// https://github.com/codegangsta/martini-contrib/pull/34#issuecomment-29683659
func ensureNotPointer(obj interface{}) {
	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding models")
	}
}

// 校验同0.1
func Validate(obj interface{}) error {

	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i).Interface()
		zero := reflect.Zero(field.Type).Interface()

		// Validate nested and embedded structs (if pointer, only do so if not nil)
		if field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Ptr && !reflect.DeepEqual(zero, fieldValue)) {
			if err := Validate(fieldValue); err != nil {
				return err
			}
		}

		if strings.Index(field.Tag.Get("binding"), "required") > -1 {
			if reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				return errors.New("Required " + name)
			}
		}
	}
	return nil
}
