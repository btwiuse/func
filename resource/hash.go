package resource

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"reflect"
	"sort"
)

// Hash computes a unique string based on the values set in the resource.
//
// The following values contribute to the hash:
//   Resource type
//   Input fields
//
// Outputs are not included in the hash.
//
// Panics in case there was an error but a panic always indicates a bug in
// Hash(); except for nil, no user input should be able to cause a panic.
func Hash(def Definition) string {
	h := fnv.New64()

	if _, err := h.Write([]byte(def.Type())); err != nil {
		panic(err)
	}

	v := reflect.Indirect(reflect.ValueOf(def))
	t := v.Type()

	fields := Fields(t, Input)
	for _, f := range fields {
		if err := visit(h, v.Field(f.Index)); err != nil {
			panic(fmt.Sprintf("Field %v in %s: %v", f.Index, t, err))
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

var one, zero = []byte("1"), []byte("0")

func visit(w io.Writer, v reflect.Value) error {
	v = reflect.Indirect(v)
	if !v.IsValid() {
		// Nil pointers are ignored
		return nil
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return binary.Write(w, binary.LittleEndian, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return binary.Write(w, binary.LittleEndian, v.Uint())
	case reflect.Float32, reflect.Float64:
		return binary.Write(w, binary.LittleEndian, v.Float())
	case reflect.Complex64, reflect.Complex128:
		return binary.Write(w, binary.LittleEndian, v.Complex())
	case reflect.String:
		_, err := w.Write([]byte(v.String()))
		return err
	case reflect.Bool:
		b := zero
		if v.Bool() {
			b = one
		}
		_, err := w.Write(b)
		return err
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := visit(w, v.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		keys := v.MapKeys()
		values := make([]*bytes.Buffer, len(keys))
		for i, k := range keys {
			values[i] = &bytes.Buffer{}
			buf := values[i]
			if err := visit(buf, k); err != nil {
				return err
			}
			if err := visit(buf, v.MapIndex(k)); err != nil {
				return err
			}
		}
		sort.Slice(values, func(i, j int) bool {
			a := values[i].Bytes()
			b := values[j].Bytes()
			return bytes.Compare(a, b) > 0
		})
		for _, b := range values {
			if _, err := io.Copy(w, b); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := visit(w, v.Field(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		return fmt.Errorf("not supported: %s", v.Kind())
	}

	// All types should be covered above so we should never reach this.
	return fmt.Errorf("missing case for %s", v.Kind())
}
