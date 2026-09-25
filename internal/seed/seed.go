package seed

import (
	"fmt"
	"log"

	"github.com/proarash/ecommerce-server/internal/cms"
	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/product"
	"github.com/proarash/ecommerce-server/internal/staff"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const samplePassword = "password123"

func hash(p string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func count(db *gorm.DB, model any, where ...any) int64 {
	var n int64
	q := db.Model(model)
	if len(where) > 0 {
		q = q.Where(where[0], where[1:]...)
	}
	q.Count(&n)
	return n
}

func Run(db *gorm.DB, cfg *config.EnvConfig) {
	seedAdmin(db, cfg)
	if cfg.Env == "production" {
		return
	}
	seedStaff(db)
	cats := seedCategories(db)
	seedProducts(db, cats)
	seedBlogs(db)
	seedSiteContent(db)
	seedBanners(db)
}

func seedAdmin(db *gorm.DB, cfg *config.EnvConfig) {
	if count(db, &staff.StaffUser{}, "role = ?", staff.RoleAdmin) > 0 {
		return
	}
	if cfg.AdminMobile == "" || cfg.AdminPassword == "" {
		log.Println("seed: ADMIN_MOBILE/ADMIN_PASSWORD not set, admin not seeded")
		return
	}
	avatar := 1
	admin := staff.StaffUser{Name: cfg.AdminName, Mobile: cfg.AdminMobile, Password: hash(cfg.AdminPassword), Role: staff.RoleAdmin, DefaultAvatarID: &avatar, Status: true}
	if err := db.Create(&admin).Error; err != nil {
		log.Println("seed: admin:", err)
		return
	}
	log.Println("seed: admin created")
}

func seedStaff(db *gorm.DB) {
	if count(db, &staff.StaffUser{}, "role <> ?", staff.RoleAdmin) > 0 {
		return
	}
	plan := []struct {
		role  string
		count int
	}{
		{staff.RoleStorekeeper, 2},
		{staff.RoleAccountant, 3},
		{staff.RoleMarketer, 2},
		{staff.RoleSupport, 5},
	}
	pw := hash(samplePassword)
	var users []staff.StaffUser
	n := 0
	for _, p := range plan {
		for i := 1; i <= p.count; i++ {
			n++
			avatar := (n-1)%5 + 1
			users = append(users, staff.StaffUser{
				Name:            fmt.Sprintf("%s %d", p.role, i),
				Mobile:          fmt.Sprintf("0930000%04d", n),
				Password:        pw,
				Role:            p.role,
				DefaultAvatarID: &avatar,
				Status:          true,
			})
		}
	}
	if err := db.Create(&users).Error; err != nil {
		log.Println("seed: staff:", err)
		return
	}
	log.Printf("seed: %d staff users created (password %q)", len(users), samplePassword)
}

func seedCategories(db *gorm.DB) []product.Category {
	var leaves []product.Category
	if count(db, &product.Category{}) > 0 {
		db.Where("parent_id IS NOT NULL").Find(&leaves)
		return leaves
	}
	tree := []struct {
		name, slug string
		children   [][2]string
	}{
		{"Electronics", "electronics", [][2]string{{"Mobile Phones", "mobile-phones"}, {"Laptops", "laptops"}}},
		{"Fashion", "fashion", [][2]string{{"Men", "men"}, {"Women", "women"}}},
	}
	for _, root := range tree {
		r := product.Category{Name: root.name, Slug: root.slug}
		if err := db.Create(&r).Error; err != nil {
			log.Println("seed: category:", err)
			continue
		}
		for _, ch := range root.children {
			c := product.Category{Name: ch[0], Slug: ch[1], ParentID: &r.ID}
			if err := db.Create(&c).Error; err != nil {
				log.Println("seed: category:", err)
				continue
			}
			leaves = append(leaves, c)
		}
	}
	log.Println("seed: categories created")
	return leaves
}

func seedProducts(db *gorm.DB, cats []product.Category) {
	if len(cats) == 0 || count(db, &product.Product{}) > 0 {
		return
	}
	products := make([]product.Product, 0, 100)
	for i := 1; i <= 100; i++ {
		cat := cats[(i-1)%len(cats)]
		products = append(products, product.Product{
			Title:       fmt.Sprintf("%s Item %03d", cat.Name, i),
			Description: fmt.Sprintf("Sample product %d in %s", i, cat.Name),
			Price:       float64(100000 + i*25000),
			SKU:         fmt.Sprintf("SKU-%05d", i),
			IsActive:    true,
			CategoryID:  cat.ID,
		})
	}
	if err := db.CreateInBatches(&products, 50).Error; err != nil {
		log.Println("seed: products:", err)
		return
	}
	stocks := make([]inventory.InventoryStock, 0, len(products))
	for _, p := range products {
		stocks = append(stocks, inventory.InventoryStock{ProductID: p.ID, Quantity: 50})
	}
	if err := db.CreateInBatches(&stocks, 50).Error; err != nil {
		log.Println("seed: stock:", err)
	}
	log.Println("seed: 100 products created")
}

func seedBlogs(db *gorm.DB) {
	if count(db, &cms.BlogPost{}) > 0 {
		return
	}
	var author staff.StaffUser
	db.Where("role = ?", staff.RoleMarketer).First(&author)
	posts := make([]cms.BlogPost, 0, 30)
	for i := 1; i <= 30; i++ {
		posts = append(posts, cms.BlogPost{
			Title:          fmt.Sprintf("Shopping Guide %d", i),
			Slug:           fmt.Sprintf("shopping-guide-%d", i),
			Description:    fmt.Sprintf("Short description for shopping guide %d", i),
			Content:        fmt.Sprintf("Full content of shopping guide %d.", i),
			SeoTitle:       fmt.Sprintf("Shopping Guide %d | Store", i),
			SeoDescription: fmt.Sprintf("Read shopping guide %d for tips and product reviews", i),
			Keywords:       "shopping,guide,tips",
			AuthorID:       author.ID,
		})
	}
	if err := db.Create(&posts).Error; err != nil {
		log.Println("seed: blogs:", err)
		return
	}
	log.Println("seed: 30 blog posts created")
}

func seedSiteContent(db *gorm.DB) {
	if count(db, &cms.SiteContent{}) > 0 {
		return
	}
	items := []cms.SiteContent{
		{Key: "h1_title", Value: "Welcome to our store", Description: "Home page H1 title"},
		{Key: "site_description", Value: "Quality products delivered to your door", Description: "Site meta description"},
		{Key: "footer_text", Value: "All rights reserved.", Description: "Footer copyright text"},
		{Key: "footer_links", Value: `[{"title":"About","url":"/about"},{"title":"Contact","url":"/contact"}]`, Description: "Footer links JSON"},
		{Key: "support_phone", Value: "021-00000000", Description: "Support phone number"},
	}
	if err := db.Create(&items).Error; err != nil {
		log.Println("seed: site content:", err)
		return
	}
	log.Println("seed: site content created")
}

func seedBanners(db *gorm.DB) {
	if count(db, &cms.Banner{}) > 0 {
		return
	}
	for i := 1; i <= 3; i++ {
		m := media.Media{URL: fmt.Sprintf("https://placehold.co/1200x400?text=Banner+%d", i), MediaType: media.TypeImage, MimeType: "image/png", FileName: fmt.Sprintf("banner-%d.png", i)}
		if err := db.Create(&m).Error; err != nil {
			log.Println("seed: banner media:", err)
			return
		}
		b := cms.Banner{Title: fmt.Sprintf("Banner %d", i), MediaID: m.ID, AltName: fmt.Sprintf("Promotional banner %d", i), LinkUrl: "/products", DisplayOrder: i, IsActive: true}
		if err := db.Create(&b).Error; err != nil {
			log.Println("seed: banner:", err)
			return
		}
	}
	log.Println("seed: banners created")
}
