package transform

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"sync"
	"time"
)

const defaultTagName = "transform"

var (
	timeDurationType = reflect.TypeOf(time.Duration(0))
	timeType         = reflect.TypeOf(time.Time{})
)

// TransformErrors is an array of FieldError's
// for use in custom error messages post validation.
type TransformErrors []FieldError

func (t TransformErrors) Error() string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(t); i++ {
		buff.WriteString(t[i].Error())
		buff.WriteString("\n")
	}

	return strings.TrimSpace(buff.String())
}

// FieldError contains all functions to get error details
type FieldError interface {
	// Tag returns the transform tag that failed.
	Tag() string

	// ActualTag returns the transform tag that failed, even if an
	// alias the actual tag within the alias will be returned.
	ActualTag() string

	// Namespace returns the namespace for the field error, with the tag
	// name taking precedence over the field's actual name.
	Namespace() string

	// StructNamespace returns the namespace for the field error, with the field's
	// actual name.
	StructNamespace() string

	// Field returns the fields name with the tag name taking precedence over the
	// field's actual name.
	Field() string

	// StructField returns the field's actual name from the struct, when able to determine.
	StructField() string

	// Value returns the actual field's value in case needed for creating the error
	// message
	Value() interface{}

	// Param returns the param value, in string form for comparison; this will also
	// help with generating an error message
	Param() string

	// Kind returns the Field's reflect Kind
	Kind() reflect.Kind

	// Type returns the Field's reflect Type
	Type() reflect.Type

	// Error returns the FieldError's message
	Error() string
}

// InvalidTransformError describes an invalid argument passed to `Struct`.
type InvalidTransformError struct {
	Type reflect.Type
}

// Error returns InvalidValidationError message
func (e *InvalidTransformError) Error() string {
	if e.Type == nil {
		return "transformer: (nil)"
	}

	return "transformer: (nil " + e.Type.String() + ")"
}

// per transform construct
type transform struct {
	t        *Transform
	top      reflect.Value
	ns       []byte
	actualNs []byte
	errs     TransformErrors
}

// Option represents a single option for the transformer.
type Option func(*Transform)

// TagNameFunc allows to define a custom function to get the tag name
type TagNameFunc func(field reflect.StructField) string

// Transform ...
type Transform struct {
	tagName        string
	pool           *sync.Pool
	tagNameFunc    TagNameFunc
	hasTagNameFunc bool
}

// New returns a new instance of `transform` with some defaults.
func New(opts ...Option) *Transform {
	t := &Transform{
		tagName: defaultTagName,
	}

	t.pool = &sync.Pool{
		New: func() interface{} {
			return &transform{
				t:        t,
				ns:       make([]byte, 0, 64),
				actualNs: make([]byte, 0, 64),
			}
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// SetTagName sets the tag name to be used for the transformer.
func (t *Transform) SetTagName(tagName string) {
	t.tagName = tagName
}

// Struct validates a struct and returns a transformer for it.
func (t *Transform) Struct(s interface{}) error {
	return t.StructCtx(context.Background(), s)
}

// RegisterTagNameFunc allows to define a custom function to get the tag name
func (t *Transform) RegisterTagNameFunc(fn TagNameFunc) {
	t.tagNameFunc = fn
	t.hasTagNameFunc = true
}

// StructCtx validates a struct and returns a transformer for it.
func (t *Transform) StructCtx(ctx context.Context, s interface{}) error {
	val := reflect.ValueOf(s)
	top := val

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type().ConvertibleTo(timeType) {
		return &InvalidTransformError{Type: reflect.TypeOf(s)}
	}

	td := t.pool.Get().(*transform)
	td.top = top
	td.transformStruct(ctx, top, val, val.Type(), td.ns[0:0], td.actualNs[0:0])

	var err error
	if len(td.errs) > 0 {
		err = td.errs
		td.errs = nil
	}

	t.pool.Put(td)

	return err
}

func (t *transform) transformStruct(ctx context.Context, parent reflect.Value, current reflect.Value, typ reflect.Type, ns []byte, structNs []byte) {
}
