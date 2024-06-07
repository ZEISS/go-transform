package transform

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type tagType uint8

const defaultTagName = "transform"

const (
	namespaceSeparator = "."
	leftBracket        = "["
	rightBracket       = "]"
)

var restrictedTags = map[string]struct{}{}

var (
	timeDurationType = reflect.TypeOf(time.Duration(0))
	timeType         = reflect.TypeOf(time.Time{})
)

// FieldLevel contains all the information and helper functions
// to transform a field
type FieldLevel interface {
	// Top returns the top level struct, if any
	Top() reflect.Value

	// Parent returns the current fields parent struct, if any or
	// the comparison value if called 'VarWithValue'
	Parent() reflect.Value

	// Field returns current field for transformation
	Field() reflect.Value

	// FieldName returns the field's name with the tag
	// name taking precedence over the fields actual name.
	FieldName() string

	// StructFieldName returns the struct field's name
	StructFieldName() string

	// Param returns param for transformation against current field
	Param() string

	// GetTag returns the current transformation tag name
	GetTag() string

	// ExtractType gets the actual underlying type of field value.
	// It will dive into pointers, customTypes and return you the
	// underlying value and it's kind.
	ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)

	// GetStructFieldOK traverses the parent struct to retrieve a specific field denoted by the provided namespace
	// in the param and returns the field, field kind and whether is was successful in retrieving
	// the field at all.
	//
	// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
	// could not be retrieved because it didn't exist.
	//
	// Deprecated: Use GetStructFieldOK2() instead which also return if the value is nullable.
	GetStructFieldOK() (reflect.Value, reflect.Kind, bool)

	// GetStructFieldOKAdvanced is the same as GetStructFieldOK except that it accepts the parent struct to start looking for
	// the field and namespace allowing more extensibility for transformators.
	//
	// Deprecated: Use GetStructFieldOKAdvanced2() instead which also return if the value is nullable.
	GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool)
}

var _ FieldLevel = new(transform)

// FuncCtx is a function that receives the current context and the FieldLevel
type FuncCtx func(ctx context.Context, fl FieldLevel) bool

// TransformErrors ...
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

// Error ...
func (e *InvalidTransformError) Error() string {
	if e.Type == nil {
		return "transformer: (nil)"
	}

	return "transformer: (nil " + e.Type.String() + ")"
}

// StructLevel ...
type StructLevel interface {
	// Tansformer ...
	Tansformer() *Transform

	// Top returns the top level struct, if any
	Top() reflect.Value

	// Parent returns the current fields parent struct, if any
	Parent() reflect.Value

	// Current returns the current struct.
	Current() reflect.Value

	// ExtractType ...
	ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)

	// ReportError ...
	ReportError(field interface{}, fieldName, structFieldName string, tag, param string)

	// ReportValidationErrors ...
	ReportValidationErrors(relativeNamespace, relativeActualNamespace string, errs TransformErrors)
}

type StructLevelFuncCtx func(ctx context.Context, sl StructLevel)

type cStruct struct {
	name   string
	fields []*cField
	fn     StructLevelFuncCtx
}

type cField struct {
	idx        int
	name       string
	altName    string
	namesEqual bool
	cTags      *cTag
}

// per transform construct
type transform struct {
	t          *Transform
	top        reflect.Value
	ns         []byte
	actualNs   []byte
	errs       TransformErrors
	flField    reflect.Value // StructLevel & FieldLevel
	cf         *cField       // StructLevel & FieldLevel
	ct         *cTag         // StructLevel & FieldLevel
	slflParent reflect.Value // StructLevel & FieldLevel
}

type cTag struct {
	tag                  string
	aliasTag             string
	actualAliasTag       string
	param                string
	keys                 *cTag
	next                 *cTag
	fn                   FuncCtx
	typeof               tagType
	hasTag               bool
	hasAlias             bool
	hasParam             bool
	isBlockEnd           bool
	runValidationWhenNil bool
}

// Option represents a single option for the transformer.
type Option func(*Transform)

// TagNameFunc allows to define a custom function to get the tag name
type TagNameFunc func(field reflect.StructField) string

type internalTransformationFuncWrapper struct {
	fn FuncCtx
}

// Transform ...
type Transform struct {
	tagName        string
	pool           *sync.Pool
	tagNameFunc    TagNameFunc
	hasTagNameFunc bool
	transformtions map[string]internalTransformationFuncWrapper
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

// SetTagName ...
func (t *Transform) SetTagName(tagName string) {
	t.tagName = tagName
}

// Struct ...
func (t *Transform) Struct(s interface{}) error {
	return t.StructCtx(context.Background(), s)
}

// RegisterTagNameFunc allows to define a custom function to get the tag name
func (t *Transform) RegisterTagNameFunc(fn TagNameFunc) {
	t.tagNameFunc = fn
	t.hasTagNameFunc = true
}

// StructCtx ...
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

// ExtractType gets the actual underlying type of field value.
func (t *transform) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	return t.extractTypeInternal(field, false)
}

func (t *Transform) registerTransformer(tag string, fn FuncCtx) error {
	if len(tag) == 0 {
		return errors.New("function Key cannot be empty")
	}

	if fn == nil {
		return errors.New("function cannot be empty")
	}

	t.transformtions[tag] = internalTransformationFuncWrapper{fn: fn}

	return nil
}

// Top ...
func (t *transform) Top() reflect.Value {
	return t.top
}

// Parent ...
func (t *transform) Parent() reflect.Value {
	return t.slflParent
}

// Field returns current field for transformation
func (t *transform) Field() reflect.Value {
	return t.flField
}

// FieldName ...
func (t *transform) FieldName() string {
	return t.cf.altName
}

// GetTag ...
func (t *transform) GetTag() string {
	return t.ct.tag
}

// StructFieldName ...
func (t *transform) StructFieldName() string {
	return t.cf.name
}

// Param ...
func (t *transform) Param() string {
	return t.ct.param
}

// GetStructFieldOK ...
func (t *transform) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	current, kind, _, found := t.getStructFieldOKInternal(t.slflParent, t.ct.param)
	return current, kind, found
}

// GetStructFieldOKAdvanced ...
func (t *transform) GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool) {
	current, kind, _, found := t.GetStructFieldOKAdvanced2(val, namespace)
	return current, kind, found
}

// GetStructFieldOKAdvanced2 ...
func (t *transform) GetStructFieldOKAdvanced2(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool) {
	return t.getStructFieldOKInternal(val, namespace)
}

func (t *transform) transformStruct(ctx context.Context, parent reflect.Value, current reflect.Value, typ reflect.Type, ns []byte, structNs []byte) {
}

// extractTypeInternal gets the actual underlying type of field value.
// It will dive into pointers, customTypes and return you the
// underlying value and it's kind.
func (t *transform) extractTypeInternal(current reflect.Value, nullable bool) (reflect.Value, reflect.Kind, bool) {
BEGIN:
	switch current.Kind() {
	case reflect.Ptr:

		nullable = true

		if current.IsNil() {
			return current, reflect.Ptr, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Interface:

		nullable = true

		if current.IsNil() {
			return current, reflect.Interface, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Invalid:
		return current, reflect.Invalid, nullable

	default:

		// if t.t.hasCustomFuncs {

		//   if fn, ok := v.v.customFuncs[current.Type()]; ok {
		//     current = reflect.ValueOf(fn(current))
		//     goto BEGIN
		//   }
		// }

		return current, current.Kind(), nullable
	}
}

func (t *transform) getStructFieldOKInternal(val reflect.Value, namespace string) (current reflect.Value, kind reflect.Kind, nullable bool, found bool) {
BEGIN:
	current, kind, nullable = t.ExtractType(val)
	if kind == reflect.Invalid {
		return
	}

	if namespace == "" {
		found = true
		return
	}

	switch kind {

	case reflect.Ptr, reflect.Interface:
		return

	case reflect.Struct:

		typ := current.Type()
		fld := namespace
		var ns string

		if !typ.ConvertibleTo(timeType) {

			idx := strings.Index(namespace, namespaceSeparator)

			if idx != -1 {
				fld = namespace[:idx]
				ns = namespace[idx+1:]
			} else {
				ns = ""
			}

			bracketIdx := strings.Index(fld, leftBracket)
			if bracketIdx != -1 {
				fld = fld[:bracketIdx]

				ns = namespace[bracketIdx:]
			}

			val = current.FieldByName(fld)
			namespace = ns
			goto BEGIN
		}

	case reflect.Array, reflect.Slice:
		idx := strings.Index(namespace, leftBracket)
		idx2 := strings.Index(namespace, rightBracket)

		arrIdx, _ := strconv.Atoi(namespace[idx+1 : idx2])

		if arrIdx >= current.Len() {
			return
		}

		startIdx := idx2 + 1

		if startIdx < len(namespace) {
			if namespace[startIdx:startIdx+1] == namespaceSeparator {
				startIdx++
			}
		}

		val = current.Index(arrIdx)
		namespace = namespace[startIdx:]
		goto BEGIN

	case reflect.Map:
		idx := strings.Index(namespace, leftBracket) + 1
		idx2 := strings.Index(namespace, rightBracket)

		endIdx := idx2

		if endIdx+1 < len(namespace) {
			if namespace[endIdx+1:endIdx+2] == namespaceSeparator {
				endIdx++
			}
		}

		key := namespace[idx:idx2]

		switch current.Type().Key().Kind() {
		case reflect.Int:
			i, _ := strconv.Atoi(key)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Int8:
			i, _ := strconv.ParseInt(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(int8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int16:
			i, _ := strconv.ParseInt(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(int16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int32:
			i, _ := strconv.ParseInt(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(int32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int64:
			i, _ := strconv.ParseInt(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Uint:
			i, _ := strconv.ParseUint(key, 10, 0)
			val = current.MapIndex(reflect.ValueOf(uint(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint8:
			i, _ := strconv.ParseUint(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(uint8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint16:
			i, _ := strconv.ParseUint(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(uint16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint32:
			i, _ := strconv.ParseUint(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(uint32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint64:
			i, _ := strconv.ParseUint(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Float32:
			f, _ := strconv.ParseFloat(key, 32)
			val = current.MapIndex(reflect.ValueOf(float32(f)))
			namespace = namespace[endIdx+1:]

		case reflect.Float64:
			f, _ := strconv.ParseFloat(key, 64)
			val = current.MapIndex(reflect.ValueOf(f))
			namespace = namespace[endIdx+1:]

		case reflect.Bool:
			b, _ := strconv.ParseBool(key)
			val = current.MapIndex(reflect.ValueOf(b))
			namespace = namespace[endIdx+1:]

		// reflect.Type = string
		default:
			val = current.MapIndex(reflect.ValueOf(key))
			namespace = namespace[endIdx+1:]
		}

		goto BEGIN
	}

	// if got here there was more namespace, cannot go any deeper
	panic("Invalid field namespace")
}
