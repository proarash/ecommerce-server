package user

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"slices"
)

var AllRoles = []string{RoleUser, RoleAdmin, RoleStorekeeper, RoleAccountant, RoleMarketer, RoleSupport}

type Roles []string

func NewRoles(roles ...string) Roles {
	r := Roles{RoleUser}
	for _, role := range roles {
		if role == RoleAdmin {
			return slices.Clone(AllRoles)
		}
		if slices.Contains(AllRoles, role) && !slices.Contains(r, role) {
			r = append(r, role)
		}
	}
	return r
}

func (r Roles) Has(role string) bool {
	return slices.Contains(r, role)
}

func (r Roles) Value() (driver.Value, error) {
	if r == nil {
		r = Roles{}
	}
	b, err := json.Marshal([]string(r))
	return string(b), err
}

func (r *Roles) Scan(src any) error {
	var b []byte
	switch v := src.(type) {
	case nil:
		*r = Roles{}
		return nil
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("roles: unsupported type")
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*r = out
	return nil
}

func RoleFilter(role string) string {
	b, _ := json.Marshal([]string{role})
	return string(b)
}
