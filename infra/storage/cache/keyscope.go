package cache

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/storage/cache/encoding"
)

type KeyScope[V any] interface {
	Get(ctx context.Context) (V, bool)
	Set(ctx context.Context, value V)
	Delete(ctx context.Context)

	GetMany(ctx context.Context) ([]*V, bool)
	SetMany(ctx context.Context, value []*V)
}

// Scope represents a set of cache keys,
// each containing an unordered set of values of type V.
type Scope[V any] struct {
	log     *wlog.Logger
	manager Manager

	scopeName string
}

type keyScope[V any] struct {
	manager Manager
	key     string

	serializer encoding.Serializer
}

func NewScope[V any](cache Manager, scope string) *Scope[V] {
	return &Scope[V]{
		manager:   cache,
		scopeName: scope,
	}
}

func (ks *Scope[V]) Key(domain, id int64, args ...any) KeyScope[V] {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("domain-%d-scope-%s-id-%d", domain, ks.scopeName, id))
	for _, arg := range args {
		sb.WriteString("-")

		sb.WriteString(ks.structuredKey(arg))
	}

	return &keyScope[V]{
		manager:    ks.manager,
		key:        sb.String(),
		serializer: encoding.DefaultSerializer,
	}
}

// structuredKey takes a prefix and a struct where the fields are concatenated
// in order to create a unique Cache key. Passing anything but a struct for
// "permutationStruct" will result in a panic. The Cache will only use the
// EXPORTED fields of the struct to construct the key. The permutation struct
// should be FLAT, with no nested structs. The fields can be any of the basic
// types, as well as slices and time.Time values.
//   - permutationStruct - A struct whose fields are concatenated to form a unique Cache key.
//     Only exported fields are used.
func (ks *Scope[V]) structuredKey(permutationStruct interface{}) string {
	var sb strings.Builder

	// Get the value of the interface
	v := reflect.ValueOf(permutationStruct)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		panic("permutationStruct must be a struct")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// Check if the field is exported, and if so skip it.
		if !field.CanInterface() {
			ks.log.Debug(fmt.Sprintf("permutationStruct contains unexported field: %s which won't be part of the cache key", v.Type().Field(i).Name))

			continue
		}

		if i > 0 {
			sb.WriteString("-")
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				sb.WriteString("nil")

				continue
			}

			// If it's not nil we'll dereference the pointer to handle its value.
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Slice:
			if field.IsNil() {
				sb.WriteString("nil")
			} else {
				sliceString := handleSlice(field)
				sb.WriteString(sliceString)
			}
		case reflect.Struct:
			if field.Type() == reflect.TypeOf(time.Time{}) {
				sb.WriteString(handleTime(field))

				continue
			}

			sb.WriteString(fmt.Sprintf("%v", field.Interface()))
		default:
			sb.WriteString(fmt.Sprintf("%v", field.Interface()))
		}
	}

	return sb.String()
}

func (s *keyScope[V]) Get(ctx context.Context) (V, bool) {
	var data V
	v := s.manager.Get(ctx, []byte(s.key))
	if err := s.serializer.Deserialize(v, &data); err != nil {
		return data, false
	}

	return data, true
}

func (s *keyScope[V]) Set(ctx context.Context, value V) {
	data, err := s.serializer.Serialize(value)
	if err != nil {
		return
	}

	s.manager.Set(ctx, []byte(s.key), data)
}

func (s *keyScope[V]) Delete(ctx context.Context) {}

func (s *keyScope[V]) GetMany(ctx context.Context) ([]*V, bool) {
	var data []*V
	v := s.manager.Get(ctx, []byte(s.key))
	if err := s.serializer.Deserialize(v, &data); err != nil {
		return nil, false
	}

	return data, true
}

func (s *keyScope[V]) SetMany(ctx context.Context, value []*V) {
	data, err := s.serializer.Serialize(value)
	if err != nil {
		return
	}

	s.manager.Set(ctx, []byte(s.key), data)
}

func handleSlice(v reflect.Value) string {
	// If the value is a pointer to a slice, get the actual slice
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Len() < 1 {
		return "empty"
	}

	var sliceStrings []string
	for i := 0; i < v.Len(); i++ {
		sliceStrings = append(sliceStrings, fmt.Sprintf("%v", v.Index(i).Interface()))
	}

	slices.Sort(sliceStrings)

	return strings.Join(sliceStrings, ",")
}

// handleTime turns the time.Time into an epoch string.
func handleTime(v reflect.Value) string {
	if timestamp, ok := v.Interface().(time.Time); ok {
		if !timestamp.IsZero() {
			return strconv.FormatInt(timestamp.Unix(), 10)
		}
	}

	return "empty-time"
}

//nolint:unused
func extractPermutation(cacheKey string) string {
	idIndex := strings.LastIndex(cacheKey, "id-")

	// "ID-" not found, return the original cache key.
	if idIndex == -1 {
		return cacheKey
	}

	// Find the last "-" before "ID-" to ensure we include "ID-" in the result
	lastDashIndex := strings.LastIndex(cacheKey[:idIndex], "-")

	// "-" not found before "ID-", return original string
	if lastDashIndex == -1 {
		return cacheKey
	}

	return cacheKey[:lastDashIndex+1]
}
