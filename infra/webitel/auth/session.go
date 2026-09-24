package auth

import (
	"slices"
	"time"

	authmodel "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

// Access is the action a method performs on its object class.
type Access uint8

const (
	AccessCreate Access = iota
	AccessRead
	AccessUpdate
	AccessDelete
)

// Value returns the bit the action occupies in a scope grant.
func (a Access) Value() uint32 {
	return [...]uint32{8, 4, 2, 1}[a]
}

func (a Access) Name() string {
	return [...]string{"create", "read", "update", "delete"}[a]
}

// Permission is what a session may do with one object class.
type Permission struct {
	Class  string
	Obac   bool
	Rbac   bool
	Access uint32
}

func (p Permission) CanCreate() bool { return p.can(AccessCreate) }
func (p Permission) CanRead() bool   { return p.can(AccessRead) }
func (p Permission) CanUpdate() bool { return p.can(AccessUpdate) }
func (p Permission) CanDelete() bool { return p.can(AccessDelete) }

func (p Permission) can(a Access) bool {
	if p.Obac || p.Rbac {
		return p.Access&a.Value() == a.Value()
	}

	return true
}

type Session struct {
	Token    string
	UserID   int64
	DomainID int64

	expiresAt   int64 // unix seconds; zero never expires
	roles       []int
	licenses    []string
	permissions []Permission
	admin       []Access
}

// NewSession maps the auth service response the way the engine's auth manager does.
func NewSession(token string, info *authmodel.Userinfo) *Session {
	s := &Session{
		Token:     token,
		UserID:    info.GetUserId(),
		DomainID:  info.GetDc(),
		expiresAt: info.GetExpiresAt(),
		licenses:  activeLicenses(info.GetLicense()),
	}

	// Access control lists match the user id alongside the role ids.
	s.roles = append(s.roles, int(info.GetUserId()))
	for _, role := range info.GetRoles() {
		s.roles = append(s.roles, int(role.GetId()))
	}

	for _, scope := range info.GetScope() {
		s.permissions = append(s.permissions, Permission{
			Class:  scope.GetClass(),
			Obac:   scope.GetObac(),
			Rbac:   scope.GetRbac(),
			Access: parseAccess(scope.GetAccess()),
		})
	}

	for _, p := range info.GetPermissions() {
		switch p.GetId() {
		case "add":
			s.admin = append(s.admin, AccessCreate)
		case "read":
			s.admin = append(s.admin, AccessRead)
		case "write":
			s.admin = append(s.admin, AccessUpdate)
		case "delete":
			s.admin = append(s.admin, AccessDelete)
		}
	}

	return s
}

func (s *Session) Validate() error {
	if s.Token == "" {
		return errors.New("session has no token")
	}

	if s.UserID < 1 {
		return errors.New("session has no user")
	}

	return nil
}

func (s *Session) Expired() bool {
	return s.expiresAt > 0 && s.expiresAt*1000 < time.Now().UnixMilli()
}

func (s *Session) HasLicense(product string) bool {
	for _, l := range s.licenses {
		if l == product {
			return true
		}
	}

	return false
}

// Roles returns a copy: the session behind it is shared by every request
// holding the token.
func (s *Session) Roles() []int {
	return slices.Clone(s.roles)
}

// Permission returns the grant for the class. An unknown class is denied and
// stays subject to row-level rules, as the engine's auth manager has it.
func (s *Session) Permission(class string) Permission {
	for _, p := range s.permissions {
		if p.Class == class {
			return p
		}
	}

	return Permission{Class: class, Obac: true, Rbac: true}
}

// UseRBAC reports whether row-level rules apply: the class is under RBAC and
// the user holds no administrative grant for the action.
func (s *Session) UseRBAC(a Access, p Permission) bool {
	if !p.Rbac {
		return false
	}

	for _, admin := range s.admin {
		if admin == a {
			return false
		}
	}

	return true
}

// activeLicenses returns the products whose license is issued and not expired.
func activeLicenses(licenses []*authmodel.LicenseUser) []string {
	now := time.Now().UnixMilli()
	out := make([]string, 0, len(licenses))

	for _, l := range licenses {
		if len(l.GetScope()) == 0 {
			continue
		}

		if exp := l.GetExpiresAt(); exp > 0 && exp <= now {
			continue
		}

		if issued := l.GetIssuedAt(); issued > 0 && now < issued {
			continue
		}

		out = append(out, l.GetProd())
	}

	return out
}

// parseAccess turns a grant such as "rwxd" into the engine's bit mask; any
// unknown character voids the grant.
func parseAccess(grant string) uint32 {
	var mask uint32

	for _, c := range grant {
		switch c {
		case 'x':
			mask |= 8
		case 'r':
			mask |= 4
		case 'w':
			mask |= 2
		case 'd':
			mask |= 1
		default:
			return 0
		}
	}

	return mask
}
