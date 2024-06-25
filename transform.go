package transform

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	defaultTagName = "transform"
)

// FieldLevel ...
type FieldLevel interface {
	// GetTag returns the current validation tag name
	GetTag() string
	// FieldName returns the current field name+
	FieldName() string
	// Field returns the current field value
	Field() reflect.Value
	// Funcs is returning the list of tag functions
	Funcs() []string
}

// Func transforms the field value
type Func func(fl FieldLevel) error

var internalTransformers = map[string]Func{
	"trim": trimFunc,
}

func trimFunc(fl FieldLevel) error {
	fl.Field().SetString(strings.TrimSpace(fl.Field().String()))

	return nil
}

var _ FieldLevel = &fieldLevel{}

type fieldLevel struct {
	field   reflect.StructField
	val     reflect.Value
	json    bool
	tagName string
}

// Field ...
func (fl fieldLevel) Field() reflect.Value {
	return fl.val
}

// FieldName ...
func (fl fieldLevel) FieldName() string {
	return fl.field.Name
}

// GetTag ...
func (fl fieldLevel) GetTag() string {
	return fl.field.Tag.Get(fl.tagName)
}

// Parent ...
func (fl fieldLevel) Parent() reflect.Value {
	return fl.val
}

// Funcs ...
func (fl fieldLevel) Funcs() []string {
	tag := fl.GetTag()
	return strings.Split(tag, ",")
}

var (
	// ErrNoPointer is returned when the interface is not a pointer
	ErrNoPointer = errors.New("transformer: interface must be a pointer")
	// ErrNoAddressable is returned when the interface is not addressable
	ErrNoAddressable = errors.New("transformer: interface must be addressable (a pointer)")
)

// Transformer ...
type Transformer interface {
	transform(string, interface{}) error
}

// TransformerImpl ...
type TransformerImpl struct {
	// TagName is the name of the tag to look for
	TagName string
}

// TransformerOpt ...
type TransformerOpt func(o *TransformerImpl)

// WithTagName ...
func WithTagName(tagName string) TransformerOpt {
	return func(o *TransformerImpl) {
		o.TagName = tagName
	}
}

// Transform ...
func Transform(name string, s interface{}) error {
	t := NewTransformer()

	return t.Transform(name, s)
}

// NewTransformer ...
func NewTransformer(opts ...TransformerOpt) *TransformerImpl {
	t := new(TransformerImpl)
	t.TagName = defaultTagName

	// configure transformer
	for _, o := range opts {
		o(t)
	}

	return t
}

// Transform ...
func (t *TransformerImpl) Transform(name string, s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr {
		return ErrNoPointer
	}

	if val.IsNil() {
		return nil // bail out if nil
	}

	val = val.Elem()
	if !val.CanAddr() {
		return ErrNoAddressable
	}

	return t.transform(val)
}

// transcode is doing the heavy lifting in the background
func (t *TransformerImpl) transform(val reflect.Value, field ...FieldLevel) error {
	var err error

	valKind := getKind(reflect.Indirect(val))

	if len(field) > 0 {
		valKind = getKind(field[0].Field())
	}

	switch valKind {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Uint, reflect.Float32:
		err = t.transformType(field[0])
	case reflect.Struct:
		err = t.transformStruct(val)
	default:
		// we have to work on here for value to pointed to
		return fmt.Errorf("transformer: unsupported type %s", valKind)
	}

	return err // should be nil
}

// transcodeType
func (t *TransformerImpl) transformType(field FieldLevel) error {
	for _, f := range field.Funcs() {
		fn, ok := internalTransformers[f]
		if !ok {
			return fmt.Errorf("transformer: function %s does not exist", f)
		}

		if err := fn(field); err != nil {
			return err
		}
	}

	return nil
}

// transdecodeStruct
func (t *TransformerImpl) transformStruct(val reflect.Value) error {
	valInterface := reflect.Indirect(val)
	valType := valInterface.Type()

	// Thes slice will keep track of all struct to transform
	structs := make([]reflect.Value, 1, 5)
	structs[0] = val

	fields := []fieldLevel{}

	for len(structs) > 0 {
		structVal := structs[0]
		structs = structs[1:]

		for i := 0; i < valType.NumField(); i++ {
			fieldType := valType.Field(i)

			isJSON := false

			// detected if this field is json
			if fieldType.Tag.Get("json") != "" {
				isJSON = true
			}

			fields = append(fields, fieldLevel{fieldType, structVal.Field(i), isJSON, t.TagName})
		}
	}

	// evaluate all fields
	for _, f := range fields {
		field, val, isJSON := f.field, f.val, f.json

		tag := field.Tag.Get(t.TagName)
		tag = strings.SplitN(tag, ",", 2)[0]

		if !val.CanAddr() {
			continue
		}

		// we try to deal with json here
		if isJSON && tag == "" {
			if !val.CanAddr() {
				continue
			}

			// check if we have to omit
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}

			continue
		}

		if err := t.transform(val, f); err != nil {
			return err
		}

		return nil

	}

	return nil
}

// getKind is returning the kind of the reflected value
func getKind(val reflect.Value) reflect.Kind {
	kind := val.Kind()

	switch {
	case kind >= reflect.Int && kind <= reflect.Int64:
		return reflect.Int
	case kind >= reflect.Uint && kind <= reflect.Uint64:
		return reflect.Uint
	case kind >= reflect.Float32 && kind <= reflect.Float64:
		return reflect.Float32
	default:
		return kind
	}
}
