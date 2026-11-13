package model

import "testing"

// A page left without a size is a page of the default size, not the first page again.
func TestSearchItemPage(t *testing.T) {
	for _, tc := range []struct {
		item          SearchItem
		limit, offset int32
	}{
		{item: SearchItem{}, limit: 11, offset: 0},
		{item: SearchItem{Page: 3}, limit: 11, offset: 20},
		{item: SearchItem{Page: 2, Size: 25}, limit: 26, offset: 25},
	} {
		if l, o := tc.item.Limit(), tc.item.Offset(); l != tc.limit || o != tc.offset {
			t.Errorf("%+v: limit %d offset %d, want %d and %d", tc.item, l, o, tc.limit, tc.offset)
		}
	}
}
