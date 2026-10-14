package product

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Keywords []string

func (k Keywords) Value() (driver.Value, error) {
	if k == nil {
		k = Keywords{}
	}
	b, err := json.Marshal([]string(k))
	return string(b), err
}

func (k *Keywords) Scan(src any) error {
	var b []byte
	switch v := src.(type) {
	case nil:
		*k = Keywords{}
		return nil
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("keywords: unsupported type")
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*k = out
	return nil
}
