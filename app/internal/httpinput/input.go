// Package httpinput bounds and validates JSON received by the HTTP APIs.
package httpinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"

	"adhdduck/internal/imageref"
)

const ReportLimit int64 = 8 << 20

// Decode accepts one object, rejects unknown fields and trailing JSON, and
// bounds allocation before decoding. Limits are per endpoint, not per binary.
func Decode(w http.ResponseWriter, r *http.Request, limit int64, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// Bound depth and collections before allocating typed inventory slices.
	preflight := json.NewDecoder(bytes.NewReader(body))
	if err := shape(preflight, "", 0, false); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("only one JSON object is allowed")
	}
	// JSON null leaves a struct empty; normal required-field checks reject it.
	return validate(reflect.ValueOf(dst), "")
}

func shape(dec *json.Decoder, field string, depth int, metadata bool) error {
	if depth > 32 {
		return errors.New("JSON nesting too deep")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); ok {
		switch delimiter {
		case '{':
			n := 0
			seen := map[string]bool{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || len(name) > 1024 {
					return errors.New("invalid object key")
				}
				if seen[name] {
					return errors.New("duplicate object key")
				}
				seen[name] = true
				n++
				if n > 4096 {
					return errors.New("too many object entries")
				}
				if err := shape(dec, name, depth+1, metadata || field == "metadata"); err != nil {
					return err
				}
			}
		case '[':
			n, max := 0, 131072
			if field == "images" || field == "repos" {
				max = 32768
			}
			if field == "sources" {
				max = 64
			}
			for dec.More() {
				n++
				if n > max {
					return errors.New("too many array entries")
				}
				if err := shape(dec, field, depth+1, metadata); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid JSON delimiter")
		}
		_, err = dec.Token()
		return err
	}
	if s, ok := token.(string); ok {
		max := 4096
		if field == "reason" || field == "error" {
			max = 65536
		}
		if metadata {
			max = 32 << 20
		}
		if len(s) > max {
			return errors.New("JSON string too long")
		}
	}
	return nil
}

func validate(v reflect.Value, field string) error {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return validate(v.Elem(), field)
	}
	switch v.Kind() {
	case reflect.Struct:
		// time.Time and similar scalar JSON types have no exported wire fields.
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.PkgPath != "" {
				continue
			}
			if err := validate(v.Field(i), f.Name); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		} // metadata is bounded by the request and finalize cap
		max := 131072
		if field == "Images" || field == "Repos" {
			max = 32768
		}
		if field == "Sources" {
			max = 64
		}
		if v.Len() > max {
			return fmt.Errorf("too many %s entries", field)
		}
		for i := 0; i < v.Len(); i++ {
			if err := validate(v.Index(i), field); err != nil {
				return err
			}
		}
	case reflect.Map:
		if v.Len() > 4096 {
			return errors.New("too many metadata entries")
		}
		for _, key := range v.MapKeys() {
			if key.Kind() == reflect.String && len(key.String()) > 1024 {
				return errors.New("metadata key too long")
			}
		}
	case reflect.String:
		s := v.String()
		max := 4096
		if field == "Reason" || field == "Error" {
			max = 65536
		}
		if len(s) > max {
			return fmt.Errorf("%s too long", field)
		}
		if field == "Image" || field == "Images" || field == "Repos" {
			if s != "" {
				if err := imageref.Validate(s); err != nil {
					return err
				}
			}
		}
		if field == "Digest" || field == "TopDig" || field == "ChainID" || strings.HasSuffix(field, "Blobs") || strings.HasSuffix(field, "Snaps") || field == "Digests" {
			if s != "" && !imageref.ValidDigest(s) {
				return fmt.Errorf("invalid %s", field)
			}
		}
	case reflect.Float32, reflect.Float64:
		n := v.Float()
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return fmt.Errorf("invalid %s", field)
		}
		if field == "Utilization" && (n < 0 || n > 1) {
			return errors.New("utilization must be between 0 and 1")
		}
		if field == "Seconds" && n < 0 {
			return errors.New("seconds must be nonnegative")
		}
	case reflect.Int, reflect.Int64:
		if (field == "Bytes" || field == "Size" || field == "TopSize") && v.Int() < 0 {
			return fmt.Errorf("%s must be nonnegative", field)
		}
	}
	return nil
}

// Error preserves 413 for oversized requests; other decoding/validation
// failures are client errors. It never echoes credentials or body contents.
func Error(w http.ResponseWriter, message string, err error) {
	var large *http.MaxBytesError
	if errors.As(err, &large) {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, message, http.StatusBadRequest)
}
