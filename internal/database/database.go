package database

import (
	"fmt"

	"github.com/proarash/ecommerce-server/internal/auth"
	"github.com/proarash/ecommerce-server/internal/cart"
	"github.com/proarash/ecommerce-server/internal/cms"
	"github.com/proarash/ecommerce-server/internal/discount"
	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/notification"
	"github.com/proarash/ecommerce-server/internal/payment"
	"github.com/proarash/ecommerce-server/internal/product"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, DisableAutomaticPing: true})
}

func Migrate(db *gorm.DB) error {
	if err := migrateLegacy(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&media.Media{},
		&wallet.Wallet{},
		&wallet.WalletTransaction{},
		&user.User{},
		&product.Category{},
		&product.Product{},
		&product.Attribute{},
		&product.ProductAttributeValue{},
		&cart.Cart{},
		&cart.CartItem{},
		&finance.Order{},
		&finance.OrderItem{},
		&finance.PreInvoice{},
		&payment.PaymentTransaction{},
		&inventory.InventoryStock{},
		&inventory.InventoryLog{},
		&cms.BlogPost{},
		&cms.Banner{},
		&cms.SiteContent{},
		&notification.Notification{},
		&discount.Discount{},
		&auth.RefreshToken{},
	); err != nil {
		return err
	}
	if err := purgeLegacyCatalog(db); err != nil {
		return err
	}
	if err := backfillCatalogSeo(db); err != nil {
		return err
	}
	return backfillMediaText(db)
}

func backfillMediaText(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&media.Media{}).Where("title = '' OR title IS NULL").
			Update("title", gorm.Expr("'تصویر ' || id")).Error; err != nil {
			return err
		}
		return tx.Model(&media.Media{}).Where("alt = '' OR alt IS NULL").
			Update("alt", gorm.Expr("title")).Error
	})
}

func backfillCatalogSeo(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		steps := []string{
			`UPDATE categories SET seo_title = 'خرید ' || name WHERE seo_title = '' OR seo_title IS NULL`,
			`UPDATE categories SET seo_title = left(seo_title, length(seo_title) - length(' | فروشگاه')) WHERE seo_title LIKE '% | فروشگاه'`,
			`UPDATE products SET seo_title = left(seo_title, length(seo_title) - length(' | فروشگاه')) WHERE seo_title LIKE '% | فروشگاه'`,
			`UPDATE categories SET seo_description = 'خرید انواع ' || name || ' با بهترین قیمت و ضمانت اصالت کالا' WHERE seo_description = '' OR seo_description IS NULL`,
			`UPDATE products SET seo_title = 'خرید ' || title WHERE seo_title = '' OR seo_title IS NULL`,
			`UPDATE products SET seo_description = 'خرید ' || title || ' با بهترین قیمت، ارسال سریع و ضمانت اصالت کالا' WHERE seo_description = '' OR seo_description IS NULL`,
			`UPDATE categories SET seo_keywords = jsonb_build_array(name, 'خرید ' || name, 'قیمت ' || name) WHERE seo_keywords = '[]'::jsonb`,
			`UPDATE products p SET seo_keywords = jsonb_build_array(c.name, p.title) FROM categories c WHERE c.id = p.category_id AND p.seo_keywords = '[]'::jsonb`,
		}
		for _, q := range steps {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var CatalogSlugs = []string{"back-cover", "screen-protector", "cable-adapter", "airpod", "lens-protector", "monopod", "power-bank"}

func purgeLegacyCatalog(db *gorm.DB) error {
	var current, all int64
	if err := db.Unscoped().Model(&product.Category{}).Where("slug IN ?", CatalogSlugs).Count(&current).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Model(&product.Category{}).Count(&all).Error; err != nil {
		return err
	}
	if current > 0 || all == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		products := "(SELECT id FROM products)"
		steps := []string{
			"DELETE FROM product_attribute_values WHERE product_id IN " + products,
			"DELETE FROM product_media WHERE product_id IN " + products,
			"DELETE FROM cart_items WHERE product_id IN " + products,
			"DELETE FROM order_items WHERE product_id IN " + products,
			"DELETE FROM inventory_logs WHERE product_id IN " + products,
			"DELETE FROM inventory_stocks WHERE product_id IN " + products,
			"DELETE FROM products",
			"UPDATE categories SET parent_id = NULL",
			"DELETE FROM categories",
		}
		for _, q := range steps {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func migrateLegacy(db *gorm.DB) error {
	m := db.Migrator()
	if hasLegacyAttributes(db) {
		if err := migrateAttributeValues(db); err != nil {
			return err
		}
	}
	if m.HasTable("refresh_tokens") && m.HasColumn("refresh_tokens", "token") {
		if err := m.DropTable("refresh_tokens"); err != nil {
			return err
		}
	}
	if m.HasTable("wallets") && m.HasColumn("wallets", "owner_type") {
		if err := db.Exec("DELETE FROM wallet_transactions WHERE wallet_id IN (SELECT id FROM wallets WHERE owner_type <> 'user')").Error; err != nil {
			return err
		}
		if err := db.Exec("DELETE FROM wallets WHERE owner_type <> 'user'").Error; err != nil {
			return err
		}
		if err := m.DropColumn("wallets", "owner_type"); err != nil {
			return err
		}
	}
	if m.HasTable("users") && m.HasColumn("users", "role") {
		if err := migrateRoles(db); err != nil {
			return err
		}
	}
	if m.HasTable("staff_users") {
		if err := m.DropTable("staff_users"); err != nil {
			return err
		}
	}
	if err := migrateKeywordsToJSON(db); err != nil {
		return err
	}
	return migrateProductSlugs(db)
}

func migrateProductSlugs(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("products") {
		return nil
	}
	if !m.HasColumn("products", "slug") {
		if err := db.Exec("ALTER TABLE products ADD COLUMN slug text").Error; err != nil {
			return err
		}
	}
	var rows []struct {
		ID    uint
		Title string
	}
	if err := db.Raw("SELECT id, title FROM products WHERE slug IS NULL OR slug = ''").Scan(&rows).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			slug := product.Slugify(r.Title)
			if slug == "" {
				slug = fmt.Sprintf("product-%d", r.ID)
			}
			if err := tx.Exec("UPDATE products SET slug = ? WHERE id = ?", slug, r.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func migrateKeywordsToJSON(db *gorm.DB) error {
	for _, table := range []string{"categories", "products"} {
		var dataType string
		if err := db.Raw("SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = 'seo_keywords'", table).Scan(&dataType).Error; err != nil {
			return err
		}
		if dataType == "" || dataType == "jsonb" {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			steps := []string{
				"ALTER TABLE " + table + " ALTER COLUMN seo_keywords DROP DEFAULT",
				"ALTER TABLE " + table + " ALTER COLUMN seo_keywords TYPE jsonb USING CASE WHEN seo_keywords IS NULL OR btrim(seo_keywords) = '' THEN '[]'::jsonb ELSE to_jsonb(string_to_array(seo_keywords, ',')) END",
				"ALTER TABLE " + table + " ALTER COLUMN seo_keywords SET DEFAULT '[]'::jsonb",
				"ALTER TABLE " + table + " ALTER COLUMN seo_keywords SET NOT NULL",
			}
			for _, q := range steps {
				if err := tx.Exec(q).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func migrateRoles(db *gorm.DB) error {
	all, err := user.NewRoles(user.RoleAdmin).Value()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		steps := []struct {
			sql  string
			args []any
		}{
			{`ALTER TABLE users ADD COLUMN IF NOT EXISTS roles jsonb NOT NULL DEFAULT '["user"]'`, nil},
			{`UPDATE users SET roles = '["user"]'::jsonb WHERE role = 'user'`, nil},
			{`UPDATE users SET roles = ?::jsonb WHERE role = 'admin'`, []any{all}},
			{`UPDATE users SET roles = jsonb_build_array('user', role) WHERE role NOT IN ('admin', 'user')`, nil},
			{`ALTER TABLE users DROP COLUMN role`, nil},
		}
		for _, s := range steps {
			if err := tx.Exec(s.sql, s.args...).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
