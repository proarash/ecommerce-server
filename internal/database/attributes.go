package database

import (
	"strings"

	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func hasLegacyAttributes(db *gorm.DB) bool {
	m := db.Migrator()
	return m.HasTable("attributes") && m.HasColumn("attributes", "title")
}

func migrateAttributeValues(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		renames := []string{
			"ALTER TABLE attributes RENAME TO attributes_legacy",
			"ALTER TABLE attributes_legacy RENAME CONSTRAINT attributes_pkey TO attributes_legacy_pkey",
			"ALTER INDEX IF EXISTS idx_attributes_key RENAME TO idx_attributes_legacy_key",
			"ALTER INDEX IF EXISTS idx_attributes_deleted_at RENAME TO idx_attributes_legacy_deleted_at",
			"ALTER SEQUENCE IF EXISTS attributes_id_seq RENAME TO attributes_legacy_id_seq",
		}
		for _, q := range renames {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		if err := tx.Migrator().CreateTable(&product.Attribute{}, &product.ProductAttributeValue{}); err != nil {
			return err
		}

		var titles []string
		if err := tx.Raw("SELECT DISTINCT trim(title) FROM attributes_legacy WHERE deleted_at IS NULL AND trim(coalesce(title, '')) <> ''").Scan(&titles).Error; err != nil {
			return err
		}
		ids := map[string]uint{}
		for _, title := range titles {
			key := product.Slugify(title)
			if key == "" {
				continue
			}
			if id, ok := ids[key]; ok {
				ids[title] = id
				continue
			}
			a := product.Attribute{Key: key, Name: title, DataType: product.DataTypeText}
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
			ids[key] = a.ID
			ids[title] = a.ID
		}

		var legacy []struct {
			ProductID uint
			Title     string
			Name      string
		}
		if err := tx.Raw(`SELECT pa.product_id, trim(l.title) AS title, trim(l.name) AS name
			FROM product_attributes pa
			JOIN attributes_legacy l ON l.id = pa.attribute_id
			WHERE l.deleted_at IS NULL`).Scan(&legacy).Error; err != nil {
			return err
		}
		rows := make([]product.ProductAttributeValue, 0, len(legacy))
		for _, item := range legacy {
			id, ok := ids[strings.TrimSpace(item.Title)]
			if !ok || item.Name == "" {
				continue
			}
			rows = append(rows, product.ProductAttributeValue{ProductID: item.ProductID, AttributeID: id, Value: item.Name})
		}
		if len(rows) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&rows, 500).Error; err != nil {
				return err
			}
		}

		return tx.Exec("DROP TABLE product_attributes, attributes_legacy").Error
	})
}

func RollbackAttributeValues(db *gorm.DB) error {
	if hasLegacyAttributes(db) || !db.Migrator().HasTable(&product.ProductAttributeValue{}) {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		steps := []string{
			`CREATE TABLE attributes_legacy (
				id bigserial PRIMARY KEY,
				created_at timestamptz,
				updated_at timestamptz,
				deleted_at timestamptz,
				key text NOT NULL UNIQUE,
				title text,
				name text
			)`,
			`CREATE TABLE product_attributes_legacy (
				product_id bigint NOT NULL,
				attribute_id bigint NOT NULL,
				PRIMARY KEY (product_id, attribute_id)
			)`,
			`INSERT INTO attributes_legacy (created_at, updated_at, key, title, name)
				SELECT now(), now(), a.key || '-' || md5(v.value), a.name, v.value
				FROM (SELECT DISTINCT attribute_id, value FROM product_attribute_values) v
				JOIN attributes a ON a.id = v.attribute_id`,
			`INSERT INTO attributes_legacy (created_at, updated_at, key, title, name)
				SELECT now(), now(), a.key, a.name, ''
				FROM attributes a
				WHERE NOT EXISTS (SELECT 1 FROM product_attribute_values v WHERE v.attribute_id = a.id)`,
			`INSERT INTO product_attributes_legacy (product_id, attribute_id)
				SELECT v.product_id, l.id
				FROM product_attribute_values v
				JOIN attributes a ON a.id = v.attribute_id
				JOIN attributes_legacy l ON l.key = a.key || '-' || md5(v.value)
				ON CONFLICT DO NOTHING`,
			"DROP TABLE product_attribute_values, attributes",
			"ALTER TABLE attributes_legacy RENAME TO attributes",
			"ALTER TABLE product_attributes_legacy RENAME TO product_attributes",
		}
		for _, q := range steps {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
