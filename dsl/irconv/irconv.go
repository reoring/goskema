package irconv

import (
	"context"
	"reflect"
	"unsafe"

	ir "github.com/reoring/goskema/internal/ir"
)

// ToIRFromSchemaDynamic converts a subset of DSL schemas into IR by inspecting
// their dynamic concrete types. This is intentionally minimal and will be expanded.
// Unsupported nodes return nil.
func ToIRFromSchemaDynamic(s any) ir.Schema {
	if s == nil {
		return nil
	}
	// Array
	if isTypeName(s, "ArraySchema") {
		item := ToIRFromSchemaDynamic(getPrivateField(s, "elem"))
		if item == nil {
			return nil
		}
		return &ir.Array{Item: item}
	}
	// Object
	if isTypeName(s, "objectSchema") {
		irObj := &ir.Object{
			Required:      map[string]struct{}{},
			UnknownPolicy: 0,
			UnknownTarget: "",
		}
		// unknownPolicy / unknownTarget
		if upAny := getPrivateField(s, "unknownPolicy"); upAny != nil {
			rv := reflect.ValueOf(upAny)
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				irObj.UnknownPolicy = int(rv.Int())
			}
		}
		if ut, ok := getPrivateField(s, "unknownTarget").(string); ok {
			irObj.UnknownTarget = ut
		}
		// required: collect keys via reflection
		if reqAny := getPrivateField(s, "required"); reqAny != nil {
			rv := reflect.ValueOf(reqAny)
			if rv.Kind() == reflect.Map {
				for _, k := range rv.MapKeys() {
					if k.Kind() == reflect.String {
						irObj.Required[k.String()] = struct{}{}
					}
				}
			}
		}
		// fields: map[string]AnyAdapter-like; iterate keys and recurse on adapter.orig
		if fieldsAny := getPrivateField(s, "fields"); fieldsAny != nil {
			rv := reflect.ValueOf(fieldsAny)
			if rv.Kind() == reflect.Map {
				for _, k := range rv.MapKeys() {
					if k.Kind() != reflect.String {
						continue
					}
					name := k.String()
					ad := rv.MapIndex(k)
					if !ad.IsValid() || (ad.Kind() == reflect.Pointer && ad.IsNil()) {
						return nil
					}
					orig := getPrivateField(ad.Interface(), "orig")
					sch := ToIRFromSchemaDynamic(orig)
					if sch == nil {
						return nil
					}
					f := ir.Field{Name: name, Schema: sch}
					// capture default when applyDefault is present
					if defFn := getPrivateField(ad.Interface(), "applyDefault"); defFn != nil {
						dfv := reflect.ValueOf(defFn)
						if dfv.Kind() == reflect.Func && !dfv.IsNil() && dfv.Type().NumIn() == 1 && dfv.Type().NumOut() == 2 {
							// expect func(context.Context) (any, error)
							outs := dfv.Call([]reflect.Value{reflect.ValueOf(context.Background())})
							if len(outs) == 2 {
								if erri := outs[1].Interface(); erri == nil {
									f.Default = outs[0].Interface()
								}
							}
						}
					}
					irObj.Fields = append(irObj.Fields, f)
				}
			}
		}
		return irObj
	}
	// Primitives
	if isTypeName(s, "stringSchema") {
		return &ir.Primitive{Name: "string"}
	}
	if isTypeName(s, "boolSchema") {
		return &ir.Primitive{Name: "bool"}
	}
	if isTypeName(s, "numberJSONSchema") {
		return &ir.Primitive{Name: "number"}
	}
	return nil
}

// --- tiny reflection helpers to read unexported fields/types ---

func isTypeName(v any, name string) bool {
	t := reflect.TypeOf(v)
	if t == nil {
		return false
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name() == name
}

func getPrivateField(v any, field string) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	// ensure addressable by creating a settable copy when needed
	if !rv.CanAddr() {
		tmp := reflect.New(rv.Type()).Elem()
		tmp.Set(rv)
		rv = tmp
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		return nil
	}
	// Create a new value we can read from even if unexported
	f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	return f.Interface()
}
