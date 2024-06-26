package transform

import (
	"errors"
	"reflect"
	"strings"
)

const (
	DefaultTagName = "transform"
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
	// Kind returns the kind of the field
	Kind() reflect.Kind
	// String returns the string value of the field
	String() string
}

// Func transforms the field value
type Func func(fl FieldLevel) error

var internalTransformers = map[string]Func{
	"trim":      trimFunc,
	"ltrim":     trimLeftFunc,
	"rtrim":     trimRightFunc,
	"lowercase": toLowerCaseFunc,
	"uppercase": toUpperCaseFunc,
}

func toUpperCaseFunc(fl FieldLevel) error {
	SetString(fl, strings.ToUpper(fl.String()))

	return nil
}

func trimLeftFunc(fl FieldLevel) error {
	SetString(fl, strings.TrimLeft(fl.String(), " "))

	return nil
}

func trimRightFunc(fl FieldLevel) error {
	SetString(fl, strings.TrimRight(fl.String(), " "))

	return nil
}

func trimFunc(fl FieldLevel) error {
	SetString(fl, strings.TrimSpace(fl.String()))

	return nil
}

func toLowerCaseFunc(fl FieldLevel) error {
	SetString(fl, strings.ToLower(fl.String()))

	return nil
}

var _ FieldLevel = (*fieldLevel)(nil)

type fieldLevel struct {
	field   reflect.StructField
	val     reflect.Value
	json    bool
	tagName string
}

// Field returns the current field value
func (fl fieldLevel) Field() reflect.Value {
	return fl.val
}

// FieldName returns the current field name
func (fl fieldLevel) FieldName() string {
	return fl.field.Name
}

// GetTag returns the current transform tag
func (fl fieldLevel) GetTag() string {
	return fl.field.Tag.Get(fl.tagName)
}

// Funcs return the list of tag functions
func (fl fieldLevel) Funcs() []string {
	tag := fl.GetTag()
	return strings.Split(tag, ",")
}

// Kind returns the kind of the field
func (fl fieldLevel) Kind() reflect.Kind {
	return fl.val.Kind()
}

// String returns the string value of the field
func (fl fieldLevel) String() string {
	if fl.Kind() == reflect.Ptr {
		return fl.Field().Elem().String()
	}

	return fl.Field().String()
}

var (
	// ErrNoPointer is returned when the interface is not a pointer
	ErrNoPointer = errors.New("transformer: interface must be a pointer")
	// ErrNoAddressable is returned when the interface is not addressable
	ErrNoAddressable = errors.New("transformer: interface must be addressable (a pointer)")
	// ErrNoStruct is returned when the interface is not a struct
	ErrNoStruct = errors.New("transformer: interface must be a struct")
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
func Transform(s interface{}) error {
	t := NewTransformer()

	return t.Transform(s)
}

// NewTransformer ...
func NewTransformer(opts ...TransformerOpt) *TransformerImpl {
	t := new(TransformerImpl)
	t.TagName = DefaultTagName

	// configure transformer
	for _, o := range opts {
		o(t)
	}

	return t
}

// Transform ...
func (t *TransformerImpl) Transform(s interface{}) error {
	ifv := reflect.ValueOf(s)

	if ifv.IsNil() {
		return nil // bail out of if this nil
	}

	if ifv.Kind() != reflect.Ptr { // we only accept pointer
		return ErrNoPointer
	}

	ifv = ifv.Elem()
	if !ifv.CanAddr() {
		return ErrNoAddressable
	}

	if ifv.Kind() != reflect.Struct {
		return ErrNoStruct // we only support struct, because of the need of tags
	}

	return t.transform(ifv)
}

// this is the heavy lifting
func (t *TransformerImpl) transform(ifv reflect.Value) error {
	vif := reflect.Indirect(ifv)
	vt := vif.Type()

	fields := []FieldLevel{}

	for i := 0; i < ifv.NumField(); i++ {
		ft := vt.Field(i)

		isJSON := false

		// detected if this field is json
		if ft.Tag.Get("json") != "" {
			isJSON = true
		}

		fields = append(fields, fieldLevel{ft, ifv.Field(i), isJSON, t.TagName})
	}

	return t.transformFields(fields...)
}

// transformField
func (t *TransformerImpl) transformFields(fields ...FieldLevel) error {
	for _, f := range fields {
		k := f.Kind()

		if k == reflect.Ptr {
			k = f.Field().Elem().Kind()
		}

		// nolint:exhaustive
		switch k {
		case reflect.String:
			if f.Field().CanSet() {
				if err := t.transformField(f); err != nil {
					return err
				}
			}
		default:
			return nil
		}
	}

	return nil
}

func (t *TransformerImpl) transformField(field FieldLevel) error {
	for _, f := range field.Funcs() {
		fn, ok := internalTransformers[f]
		if !ok {
			return nil // bail out if we don't have the function
		}

		if err := fn(field); err != nil {
			return err
		}
	}

	return nil
}

// SetString ...
func SetString(f FieldLevel, s string) {
	if f.Kind() == reflect.Ptr && f.Field().IsNil() {
		return // we don't want to set nil
	}

	if f.Kind() == reflect.Ptr {
		f.Field().Set(reflect.ValueOf(&s))
	} else {
		f.Field().SetString(s)
	}
}
