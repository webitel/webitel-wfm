package model

type SignedInUser struct {
	RbacOptions

	Id       int64
	DomainId int64
	Token    string
	Object   string
	UseRBAC  bool
}

type RbacOptions struct {
	Groups []int
	Access uint32
}
