package fields

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type User struct {
	Username string
	FullName string
	Email    string
	Alias    string    `db:"id"`
	Theme    Theme     `db:"theme,json"`
	LastSeen time.Time `db:"-"`
}

type Theme struct {
	PrimaryColor       string
	SecondaryColor     string
	TextColor          string
	TextUppercase      bool
	FontFamilyHeadings string
	FontFamilyBody     string
	FontFamilyDefault  string
}

func ExampleWildcard() {
	sql := "SELECT " + Wildcard(User{}) + " WHERE id = $1"
	fmt.Println(sql)
	// Output:
	// SELECT "username","full_name","email","id","theme" WHERE id = $1
}

func ExampleFields() {
	fmt.Println(strings.Join(Fields(User{}), "\n"))
	// Output:
	// username
	// full_name
	// email
	// id
	// theme
}

type mock struct {
	Automatic string
	Tagged    string `db:"tagged"`
	OneTwo    string // OneTwo should be one_two in the database.
	CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
	Ignored   string `db:"-"`
}

type mockEmbed struct {
	Before int
	mock
	After string
}

type numericMock struct {
	Number int
}

type mockSimpleMultiEmbed struct {
	mock
	numericMock
}

type mockMultiEmbed struct {
	A string
	mock
	B string
	numericMock
	C string
}

type emptyEmbed struct{}

type nameMock struct {
	emptyEmbed //lint:ignore U1000 Mock to test empty embeds.
	Name       string
}

type jsonMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      Theme `db:"theme,json"`
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type HasNestedMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      Theme
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type HasPointerNestedMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      *Theme
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type themeImplicit struct {
	ID  string
	XYZ Theme `db:",json"`
}

func TestWildcard(t *testing.T) {
	t.Parallel()
	var uninitializedPointer *jsonMock
	testCases := []struct {
		v    any
		desc string
		want string
	}{
		{
			v:    emptyEmbed{},
			desc: "empty",
		},
		{
			v: struct {
				unexported int
			}{},
			desc: "unexported",
			want: "",
		},
		{
			v: &struct {
				unexported int
			}{},
			desc: "unexported pointer",
			want: "",
		},
		{
			v: struct {
				One int
			}{},
			desc: "single",
			want: `"one"`,
		},
		{
			v: &struct {
				One int
			}{},
			desc: "pointer single",
			want: `"one"`,
		},
		{
			v: mock{
				Automatic: "auto string",
				Tagged:    "tag string",
			},
			desc: "mock",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: mock{
				Automatic: "auto string",
				Tagged:    "tag string",
			},
			desc: "cached",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			desc: "anonymous",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: struct {
				Automatic  string
				Tagged     string `db:"tagged"`
				OneTwo     string // OneTwo should be one_two in the database.
				CamelCase  string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored    string `db:"-"`
				Copy       string
				Duplicated string `db:"copy"`
			}{},
			desc: "duplicated",
			want: `"automatic","tagged","one_two","CamelCase","copy"`,
		},
		{
			v:    mockEmbed{},
			desc: "embed",
			want: `"before","automatic","tagged","one_two","CamelCase","after"`,
		},
		{
			v:    numericMock{},
			desc: "numeric",
			want: `"number"`,
		},
		{
			v:    mockSimpleMultiEmbed{},
			desc: "multisimpleembed",
			want: `"automatic","tagged","one_two","CamelCase","number"`,
		},
		{
			v:    mockMultiEmbed{},
			desc: "multiembed",
			want: `"a","automatic","tagged","one_two","CamelCase","b","number","c"`,
		},
		{
			v:    &mock{},
			desc: "pointer",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v:    &nameMock{},
			desc: "namemock",
			want: `"name"`,
		},
		{
			v:    &jsonMock{},
			desc: "json",
			want: `"id","name","code","is_active","theme","created_at","modified_at"`,
		},
		{
			v:    uninitializedPointer,
			desc: "uninitializedPointer",
			want: `"id","name","code","is_active","theme","created_at","modified_at"`,
		},
		{
			v:    nil,
			desc: "nil",
			want: "",
		},
		{
			v:    &HasNestedMock{},
			desc: "HasNestedMock",
			want: `"id","name","code","is_active","theme.primary_color" as "theme.primary_color","theme.secondary_color" as "theme.secondary_color","theme.text_color" as "theme.text_color","theme.text_uppercase" as "theme.text_uppercase","theme.font_family_headings" as "theme.font_family_headings","theme.font_family_body" as "theme.font_family_body","theme.font_family_default" as "theme.font_family_default","theme","created_at","modified_at"`,
		},
		{
			v:    &HasPointerNestedMock{},
			desc: "HasNestedMock",
			want: `"id","name","code","is_active","theme.primary_color" as "theme.primary_color","theme.secondary_color" as "theme.secondary_color","theme.text_color" as "theme.text_color","theme.text_uppercase" as "theme.text_uppercase","theme.font_family_headings" as "theme.font_family_headings","theme.font_family_body" as "theme.font_family_body","theme.font_family_default" as "theme.font_family_default","theme","created_at","modified_at"`,
		},
		{
			// Testing an edge case:
			// Regular fields containing dots are aliased even when unnecessary, and this should be okay.
			// This is a conscious design decision to reduce complexity avoiding leaking internal details from
			// internal/structref through the fields() function.
			v: struct {
				RegularFieldWithDots string `db:"regular.field.with.dots"`
			}{},
			desc: "RegularFieldWithDots",
			want: `"regular.field.with.dots" as "regular.field.with.dots"`,
		},
		{
			v:    themeImplicit{},
			desc: "implicit",
			want: `"id","xyz"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := Wildcard(tc.v); tc.want != got {
				t.Errorf("expected expression to be %v, got %v instead", tc.want, got)
			}
		})
	}
}

func BenchmarkWildcard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Wildcard(mock{})
	}
}

func BenchmarkWildcardAsync(b *testing.B) {
	var w sync.WaitGroup
	w.Add(b.N)
	for i := 0; i < b.N; i++ {
		go func() {
			Wildcard(mock{})
			w.Done()
		}()
	}
	w.Wait()
}

func TestWildcardCache(t *testing.T) {
	old := wildcardsCache
	t.Cleanup(func() {
		wildcardsCache = old // Restore default caching.
	})

	const maxCached = 3
	wildcardsCache = &lru{
		cap: maxCached,

		m: map[reflect.Type]*list.Element{},
		l: list.New(),
	}

	mocks := []struct {
		v           any
		want        string
		cachedItems int
	}{
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			want:        `"automatic","tagged","one_two","CamelCase"`,
			cachedItems: 1,
		},
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			want:        `"automatic","tagged","one_two","CamelCase"`,
			cachedItems: 1,
		},
		{
			v: struct {
				Number int
			}{},
			want:        `"number"`,
			cachedItems: 2,
		},
		{
			v: struct {
				Name string
			}{},
			want:        `"name"`,
			cachedItems: 3,
		},
		{
			v: struct {
				A string
				B string
				C string
			}{},
			want:        `"a","b","c"`,
			cachedItems: 3,
		},
		{
			v: struct {
				Name string
			}{},
			want:        `"name"`,
			cachedItems: 3,
		},
		{
			v: struct {
				Name string
				Age  int
			}{},
			want:        `"name","age"`,
			cachedItems: 3,
		},
	}
	for _, m := range mocks {
		orig := Wildcard(m.v)
		cached := Wildcard(m.v)

		if orig != m.want {
			t.Errorf("wanted %v, got %v instead", m.want, orig)
		}
		if orig != cached {
			t.Errorf("wanted cached value %v, got %v instead", m.want, cached)
		}
		if wildcardsCache.l.Len() != len(wildcardsCache.m) {
			t.Error("cache doubly linked list and map length should match")
		}
		if len(wildcardsCache.m) > maxCached {
			t.Errorf("cache should contain %d once full, got %d instead", maxCached, len(wildcardsCache.m))
		}
		if len(wildcardsCache.m) != m.cachedItems {
			t.Errorf("wanted %d cached items, found %d", m.cachedItems, len(wildcardsCache.m))
		}
	}
}
