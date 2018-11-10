package gin

import (
	"errors"
	"reflect"
	"strings"
)

// 验证
func Validate(c *Context, obj interface{}) error {

	var err error
	// 获取接收json对象的类型
	typ := reflect.TypeOf(obj)
	// 获取接收json对象的值
	val := reflect.ValueOf(obj)

	// 如果是ptr, 则获取实际的类型
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	// 迭代字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i).Interface()
		zero := reflect.Zero(field.Type).Interface()

		// Validate nested and embedded structs (if pointer, only do so if not nil)
		if field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Ptr && !reflect.DeepEqual(zero, fieldValue)) {
			err = Validate(c, fieldValue)
		}

		if strings.Index(field.Tag.Get("binding"), "required") > -1 {
			// 必选项
			if reflect.DeepEqual(zero, fieldValue) {
				name := field.Name
				if j := field.Tag.Get("json"); j != "" {
					name = j
				} else if f := field.Tag.Get("form"); f != "" {
					name = f
				}
				err = errors.New("Required " + name)
				c.Error(err, "json validation")
			}
		}
	}
	return err
}
