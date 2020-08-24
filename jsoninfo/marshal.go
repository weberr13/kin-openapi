package jsoninfo

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// package global marshaling context.
// This is required since the encoding uses multiple, independent ObjectEncoder instances
// and there's no way to pass this context through.
// This context contains the root component references and the current path depth
// (see EncodeStructFieldsAndExtensions and setResultField below)
//
// This context is used in MarshalRef
var marshalContext = newMarshalContext(map[interface{}]string{})

// MarshalStrictStruct function:
//   * Marshals struct fields, ignoring MarshalJSON() and fields without 'json' tag.
//   * Correctly handles StrictStruct semantics.
func MarshalStrictStruct(value StrictStruct) ([]byte, error) {
	encoder := NewObjectEncoder()
	if err := value.EncodeWith(encoder, value); err != nil {
		return nil, err
	}
	return encoder.Bytes()
}

func SetMarshalContext(rootComponentRefPaths map[interface{}]string) {
	marshalContext = newMarshalContext(rootComponentRefPaths)
}

type encodingContext struct {
	path                  []string
	rootComponentRefPaths map[interface{}]string
}

func newMarshalContext(rootComponentRefPaths map[interface{}]string) encodingContext {
	return encodingContext{
		path:                  []string{"#"},
		rootComponentRefPaths: rootComponentRefPaths,
	}
}

type ObjectEncoder struct {
	result map[string]json.RawMessage
}

func NewObjectEncoder() *ObjectEncoder {
	return &ObjectEncoder{
		result: make(map[string]json.RawMessage, 8),
	}
}

// Bytes returns the result of encoding.
func (encoder *ObjectEncoder) Bytes() ([]byte, error) {
	return json.Marshal(encoder.result)
}

// EncodeExtension adds a key/value to the current JSON object.
func (encoder *ObjectEncoder) EncodeExtension(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	encoder.result[key] = data
	return nil
}

// EncodeExtensionMap adds all properties to the result.
func (encoder *ObjectEncoder) EncodeExtensionMap(value map[string]json.RawMessage) error {
	if value != nil {
		result := encoder.result
		for k, v := range value {
			result[k] = v
		}
	}
	return nil
}

func (encoder *ObjectEncoder) EncodeStructFieldsAndExtensions(value interface{}) error {
	reflection := reflect.ValueOf(value)

	// Follow "encoding/json" semantics
	if reflection.Kind() != reflect.Ptr {
		// Panic because this is a clear programming error
		panic(fmt.Errorf("Value %s is not a pointer", reflection.Type().String()))
	}
	if reflection.IsNil() {
		// Panic because this is a clear programming error
		panic(fmt.Errorf("Value %s is nil", reflection.Type().String()))
	}

	// Take the element
	reflection = reflection.Elem()

	// Obtain typeInfo
	typeInfo := GetTypeInfo(reflection.Type())

	// Declare result
	result := encoder.result

	// Supported fields
iteration:
	for _, field := range typeInfo.Fields {
		// Fields without JSON tag are ignored
		if !field.HasJSONTag {
			continue
		}

		// Marshal
		fieldValue := reflection.FieldByIndex(field.Index)
		if v, ok := fieldValue.Interface().(json.Marshaler); ok {
			if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
				if field.JSONOmitEmpty {
					continue iteration
				}
				result[field.JSONName] = []byte("null")
				continue
			}
			err2 := encoder.setResultField(v, result, field)
			if err2 != nil {
				return err2
			}
			continue
		}
		switch fieldValue.Kind() {
		case reflect.Ptr, reflect.Interface:
			if fieldValue.IsNil() {
				if field.JSONOmitEmpty {
					continue iteration
				}
				result[field.JSONName] = []byte("null")
				continue
			}
		case reflect.Struct:
		case reflect.Map:
			if field.JSONOmitEmpty && (fieldValue.IsNil() || fieldValue.Len() == 0) {
				continue iteration
			}
		case reflect.Slice:
			if field.JSONOmitEmpty && fieldValue.Len() == 0 {
				continue iteration
			}
		case reflect.Bool:
			x := fieldValue.Bool()
			if field.JSONOmitEmpty && !x {
				continue iteration
			}
			s := "false"
			if x {
				s = "true"
			}
			result[field.JSONName] = []byte(s)
			continue iteration
		case reflect.Int64, reflect.Int, reflect.Int32:
			if field.JSONOmitEmpty && fieldValue.Int() == 0 {
				continue iteration
			}
		case reflect.Uint64, reflect.Uint, reflect.Uint32:
			if field.JSONOmitEmpty && fieldValue.Uint() == 0 {
				continue iteration
			}
		case reflect.Float64:
			if field.JSONOmitEmpty && fieldValue.Float() == 0.0 {
				continue iteration
			}
		case reflect.String:
			if field.JSONOmitEmpty && len(fieldValue.String()) == 0 {
				continue iteration
			}
		default:
			panic(fmt.Errorf("Field '%s' has unsupported type %s", field.JSONName, field.Type.String()))
		}

		// No special treament is needed
		// Use plain old "encoding/json".Marshal
		err := encoder.setResultField(fieldValue.Addr().Interface(), result, field)
		if err != nil {
			return err
		}
	}

	return nil
}

func (encoder *ObjectEncoder) setResultField(v interface{}, result map[string]json.RawMessage, field FieldInfo) error {
	var fieldData []byte
	var err error

	// push new element to path
	marshalContext.path = append(marshalContext.path, field.JSONName)
	if jm, ok := v.(json.Marshaler); ok {
		fieldData, err = jm.MarshalJSON()
	} else {
		fieldData, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	result[field.JSONName] = fieldData
	// pop last element from path
	marshalContext.path = marshalContext.path[:len(marshalContext.path)-1]
	return nil
}
