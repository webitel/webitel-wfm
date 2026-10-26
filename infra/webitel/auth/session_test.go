package auth

import (
	"testing"
	"time"

	authmodel "buf.build/gen/go/webitel/webitel-go/protocolbuffers/go"
)

func TestNewSession(t *testing.T) {
	now := time.Now().UnixMilli()
	info := &authmodel.Userinfo{
		UserId:    7,
		Dc:        3,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Roles:     []*authmodel.ObjectId{{Id: 11}, {Id: 12}},
		License: []*authmodel.LicenseUser{
			{Prod: "WFM", Scope: []string{"pause_templates"}, ExpiresAt: now + 60_000},
			{Prod: "EXPIRED", Scope: []string{"x"}, ExpiresAt: now - 1},
			{Prod: "FUTURE", Scope: []string{"x"}, IssuedAt: now + 60_000},
			{Prod: "SCOPELESS"},
		},
		Scope: []*authmodel.Objclass{
			{Class: "pause_templates", Rbac: true, Obac: true, Access: "rwxd"},
			{Class: "shift_templates", Rbac: true, Access: "r"},
			{Class: "broken", Obac: true, Access: "r?"},
		},
		Permissions: []*authmodel.Permission{{Id: "add"}},
	}

	s := NewSession("tok", info)

	if got := s.Roles(); len(got) != 3 || got[0] != 7 || got[1] != 11 || got[2] != 12 {
		t.Errorf("roles = %v, want [7 11 12] (user id first)", got)
	}

	for product, want := range map[string]bool{"WFM": true, "EXPIRED": false, "FUTURE": false, "SCOPELESS": false, "NONE": false} {
		if got := s.HasLicense(product); got != want {
			t.Errorf("HasLicense(%s) = %v, want %v", product, got, want)
		}
	}

	if p := s.Permission("pause_templates"); p.Access != 15 || !p.CanCreate() || !p.CanDelete() {
		t.Errorf("rwxd grant = %+v, want access 15 with every action allowed", p)
	}

	if p := s.Permission("shift_templates"); p.CanCreate() || !p.CanRead() {
		t.Errorf("r grant = %+v, want read only", p)
	}

	if p := s.Permission("broken"); p.Access != 0 || p.CanRead() {
		t.Errorf("unknown access char must void the grant, got %+v", p)
	}

	// An unknown class is denied and stays under row-level rules, as wbt's
	// NotAllowPermission has it (Obac and Rbac both set).
	if p := s.Permission("unknown"); !p.Obac || !p.Rbac || p.CanRead() {
		t.Errorf("unknown class must be denied and rbac-guarded, got %+v", p)
	}

	// "add" is an administrative grant: RBAC is skipped for creates only.
	if s.UseRBAC(AccessCreate, s.Permission("pause_templates")) {
		t.Error("UseRBAC(create) = true, want false for an admin grant")
	}

	if !s.UseRBAC(AccessRead, s.Permission("pause_templates")) {
		t.Error("UseRBAC(read) = false, want true")
	}

	if !s.UseRBAC(AccessRead, s.Permission("unknown")) {
		t.Error("UseRBAC on an unknown class must be true: no grant means row-level filtering applies")
	}

	if s.Expired() {
		t.Error("session with a future expiry reported expired")
	}

	if err := s.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}

	if !NewSession("tok", &authmodel.Userinfo{UserId: 1, ExpiresAt: 1}).Expired() {
		t.Error("session expired in 1970 reported valid")
	}
}
