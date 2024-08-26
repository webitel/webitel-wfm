package tests

import (
	"math/rand"

	"github.com/webitel/webitel-wfm/internal/model"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}

func ValueToPTR[T any](value T) *T {
	return &value
}

func User(access uint32, rbac bool) *model.SignedInUser {
	return &model.SignedInUser{
		Id:       1,
		DomainId: 2,
		Token:    "foobar",
		UseRBAC:  rbac,
		RbacOptions: model.RbacOptions{
			Access: access,
			Groups: []int{1, 2},
		},
	}
}
