package model

type SignedInUser struct {
	RbacOptions

	ID       int64
	DomainID int64
	Token    string
	Object   string
	UseRBAC  bool
}

type RbacOptions struct {
	Groups []int
	Access uint32
}
