package transform

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	defaultTagName = "transform"
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

// Transform ...
func Transform(name string, s interface{}) error {
	t := NewTransformer()

	return t.Transform(name, s)
}

// NewTransformer
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
		return errors.New("kvstructure: interface must be a pointer")
	}

	val = val.Elem()
	if !val.CanAddr() {
		return errors.New("kvstructure: interface must be addressable (a pointer)")
	}

	return t.transform(name, reflect.ValueOf(s).Elem())
}

// transcode is doing the heavy lifting in the background
func (t *TransformerImpl) transform(name string, val reflect.Value) error {
	var err error
	valKind := getKind(reflect.Indirect(val))
	switch valKind {
	// case reflect.String:
	// 	err = t.transcodeString(name, val)
	// case reflect.Bool:
	// 	err = t.transcodeBool(name, val)
	// case reflect.Int:
	// 	err = t.transcodeInt(name, val)
	// case reflect.Uint:
	// 	err = t.transcodeUint(name, val)
	// case reflect.Float32:
	// 	err = t.transcodeFloat(name, val)
	// case reflect.Struct:
	// 	err = t.transcodeStruct(name, val)
	// case reflect.Slice:
	// 	// silent do nothing
	// 	err = t.transcodeSlice(name, val)
	default:
		// we have to work on here for value to pointed to
		return fmt.Errorf("kvstructure: unsupported type %s", valKind)
	}

	// should be nil
	return err
}

// transdecodeString
func (t *TransformerImpl) transformString(name string, val reflect.Value) error {
	return nil
}

func (t *TransformerImpl) transformBool(name string, val reflect.Value) error {
	return nil
}

func (t *TransformerImpl) transformInt(name string, val reflect.Value) error {
	return nil
}

func (t *TransformerImpl) transformUint(name string, val reflect.Value) error {
	return nil
}

func (t *TransformerImpl) transformFloat(name string, val reflect.Value) error {
	return nil
}

func (t *TransformerImpl) transformSlice(name string, val reflect.Value) error {
	return nil
}

// transdecodeStruct
func (t *TransformerImpl) transformStruct(name string, val reflect.Value) error {
	valInterface := reflect.Indirect(val)
	valType := valInterface.Type()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create an errgroup to trace the latest error and return
	g, _ := errgroup.WithContext(ctx)

	// The slice will keep track of all structs we'll be transcoding.
	// There can be more structs, if we have embedded structs that are squashed.
	structs := make([]reflect.Value, 1, 5)
	structs[0] = val

	type field struct {
		field reflect.StructField
		val   reflect.Value
		json  bool
	}
	fields := []field{}

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

			fields = append(fields, field{fieldType, structVal.Elem().Field(i), isJSON})
		}
	}

	// evaluate all fields
	for _, f := range fields {
		field, val, isJSON := f.field, f.val, f.json
		kv := strings.ToLower(field.Name)

		tag := field.Tag.Get(t.TagName)
		tag = strings.SplitN(tag, ",", 2)[0]
		if tag != "" {
			kv = tag
		}

		if name != "" {
			kv = strings.Join([]string{name, kv}, "/")
		}

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

		g.Go(func() error {
			if err := t.transform(kv, val); err != nil {
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
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
