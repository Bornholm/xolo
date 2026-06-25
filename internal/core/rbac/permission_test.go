package rbac

import "testing"

func TestNewPermissionSetWriteImpliesRead(t *testing.T) {
	set := NewPermissionSet([]string{string(PermMembersWrite)}, nil)

	if !set.Has(PermMembersWrite) {
		t.Error("expected write permission to be granted")
	}
	if !set.Has(PermMembersRead) {
		t.Error("expected write to imply read")
	}
	if set.Has(PermProvidersRead) {
		t.Error("did not expect unrelated permission")
	}
}

func TestNewPermissionSetIgnoresUnknownCodes(t *testing.T) {
	set := NewPermissionSet([]string{"bogus:code", string(PermUsageRead)}, nil)

	if set.Has(Permission("bogus:code")) {
		t.Error("unknown code should not be granted")
	}
	if !set.Has(PermUsageRead) {
		t.Error("known code should be granted")
	}
}

func TestOwnerPermissionSetGrantsEverything(t *testing.T) {
	set := OwnerPermissionSet()

	if !set.IsOwner() {
		t.Error("expected owner set")
	}
	if !set.Has(PermSettingsWrite) {
		t.Error("owner should have every permission")
	}
	if !set.HasModelAccess("any", ModelKindLLM) {
		t.Error("owner should have access to any model")
	}
}

func TestHasModelAccess(t *testing.T) {
	set := NewPermissionSet(nil, []ModelGrant{
		{ModelID: "m1", Kind: ModelKindLLM},
	})

	if !set.HasModelAccess("m1", ModelKindLLM) {
		t.Error("expected granted model access")
	}
	if set.HasModelAccess("m1", ModelKindVirtual) {
		t.Error("kind should matter")
	}
	if set.HasModelAccess("m2", ModelKindLLM) {
		t.Error("ungranted model should not be accessible")
	}
}

func TestIsKnown(t *testing.T) {
	if !IsKnown(string(PermModelUseOrg)) {
		t.Error("declared permission should be known")
	}
	if IsKnown("nonexistent:perm") {
		t.Error("undeclared permission should not be known")
	}
}

func TestCatalogOnlyKnownCodes(t *testing.T) {
	for _, group := range Catalog() {
		for _, def := range group.Perms {
			if !IsKnown(string(def.Code)) {
				t.Errorf("catalog code %q reported as unknown", def.Code)
			}
		}
	}
}
