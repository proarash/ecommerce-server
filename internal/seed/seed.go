package seed

import (
	"fmt"
	"log"
	"strings"

	"github.com/proarash/ecommerce-server/internal/cms"
	"github.com/proarash/ecommerce-server/internal/config"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/product"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	samplePassword      = "password123"
	productsPerCategory = 60
	adminBalance        = 10000000000
	userBalance         = 100000000
)

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
		seedWallets(db)
		return
	}
	seedStaff(db)
	seedWallets(db)
	cats := seedCategories(db)
	seedProducts(db, cats)
	seedBlogs(db)
	seedSiteContent(db)
	seedBanners(db)
}

func seedAdmin(db *gorm.DB, cfg *config.EnvConfig) {
	if count(db, &user.User{}, "roles @> ?::jsonb", user.RoleFilter(user.RoleAdmin)) > 0 {
		return
	}
	if cfg.AdminMobile == "" || cfg.AdminPassword == "" {
		log.Println("seed: ADMIN_MOBILE/ADMIN_PASSWORD not set, admin not seeded")
		return
	}
	avatar := 1
	admin := user.User{Name: &cfg.AdminName, Mobile: cfg.AdminMobile, Password: hash(cfg.AdminPassword), Roles: user.NewRoles(user.RoleAdmin), DefaultAvatarID: &avatar, Status: true}
	if err := db.Create(&admin).Error; err != nil {
		log.Println("seed: admin:", err)
		return
	}
	log.Println("seed: admin created")
}

func seedStaff(db *gorm.DB) {
	if count(db, &user.User{}, "jsonb_array_length(roles) > 1 AND NOT roles @> ?::jsonb", user.RoleFilter(user.RoleAdmin)) > 0 {
		return
	}
	plan := []struct {
		role  string
		count int
	}{
		{user.RoleStorekeeper, 2},
		{user.RoleAccountant, 3},
		{user.RoleMarketer, 2},
		{user.RoleSupport, 5},
	}
	pw := hash(samplePassword)
	var users []user.User
	n := 0
	for _, p := range plan {
		for i := 1; i <= p.count; i++ {
			n++
			avatar := (n-1)%5 + 1
			name := fmt.Sprintf("%s %d", p.role, i)
			users = append(users, user.User{
				Name:            &name,
				Mobile:          fmt.Sprintf("0930000%04d", n),
				Password:        pw,
				Roles:           user.NewRoles(p.role),
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

func seedWallets(db *gorm.DB) {
	admins := db.Model(&user.User{}).Select("id").Where("roles @> ?::jsonb", user.RoleFilter(user.RoleAdmin))
	if err := db.Model(&wallet.Wallet{}).Where("amount = 0 AND owner_id IN (?)", admins).Update("amount", adminBalance).Error; err != nil {
		log.Println("seed: admin wallet:", err)
	}
	if err := db.Model(&wallet.Wallet{}).Where("amount = 0 AND owner_id NOT IN (?)", admins).Update("amount", userBalance).Error; err != nil {
		log.Println("seed: user wallets:", err)
	}
}

func seedCategories(db *gorm.DB) map[string]product.Category {
	cats := make(map[string]product.Category, len(catalog))
	for _, def := range catalog {
		c := product.Category{Name: def.name, Slug: def.slug, SeoTitle: fmt.Sprintf("خرید %s | فروشگاه", def.name), SeoDescription: fmt.Sprintf("خرید انواع %s با بهترین قیمت و ضمانت اصالت کالا", def.name), SeoKeywords: product.Keywords{def.name, "خرید " + def.name, "قیمت " + def.name}}
		if err := db.Where(product.Category{Slug: def.slug}).FirstOrCreate(&c).Error; err != nil {
			log.Println("seed: category:", err)
			continue
		}
		cats[def.slug] = c
	}
	return cats
}

func seedAttribute(db *gorm.DB, cache map[string]uint, name string) (uint, error) {
	key := product.Slugify(name)
	if id, ok := cache[key]; ok {
		return id, nil
	}
	a := product.Attribute{Key: key}
	if err := db.Where(product.Attribute{Key: key}).Attrs(product.Attribute{Name: name, DataType: product.DataTypeText}).FirstOrCreate(&a).Error; err != nil {
		return 0, err
	}
	cache[key] = a.ID
	return a.ID, nil
}

func seedProducts(db *gorm.DB, cats map[string]product.Category) {
	total := 0
	attrIDs := map[string]uint{}
	for _, def := range catalog {
		cat, ok := cats[def.slug]
		if !ok || count(db, &product.Product{}, "category_id = ?", cat.ID) > 0 {
			continue
		}
		products := make([]product.Product, 0, productsPerCategory)
		for i := 0; i < productsPerCategory; i++ {
			item := i % len(def.items)
			variant := (i / len(def.items)) % len(def.variants)
			specs := []struct{ name, value string }{
				{def.itemAttr, def.items[item]},
				{def.variantAttr, def.variants[variant]},
			}
			for j, ex := range def.extras {
				specs = append(specs, struct{ name, value string }{ex.title, ex.values[(i+j)%len(ex.values)]})
			}
			values := make([]product.ProductAttributeValue, 0, len(specs))
			for _, spec := range specs {
				id, err := seedAttribute(db, attrIDs, spec.name)
				if err != nil {
					log.Println("seed: attribute:", err)
					continue
				}
				values = append(values, product.ProductAttributeValue{AttributeID: id, Value: spec.value})
			}
			title := strings.TrimSpace(fmt.Sprintf("%s %s %s", def.prefix, def.items[item], def.variants[variant]))
			products = append(products, product.Product{
				Title:           title,
				Slug:            product.Slugify(title),
				Description:     fmt.Sprintf("%s، از دسته %s، با کیفیت ساخت بالا، ضمانت اصالت و سلامت فیزیکی کالا.", title, def.name),
				SeoTitle:        fmt.Sprintf("خرید %s | فروشگاه", title),
				SeoDescription:  fmt.Sprintf("خرید %s با بهترین قیمت، ارسال سریع و ضمانت اصالت کالا", title),
				SeoKeywords:     product.Keywords{def.name, def.items[item], def.variants[variant], title},
				Price:           float64(def.basePrice + item*def.step + variant*def.step*3),
				SKU:             fmt.Sprintf("%s-%03d", strings.ToUpper(def.slug), i+1),
				IsActive:        true,
				CategoryID:      cat.ID,
				AttributeValues: values,
			})
		}
		if err := db.CreateInBatches(&products, productsPerCategory).Error; err != nil {
			log.Println("seed: products:", err)
			continue
		}
		stocks := make([]inventory.InventoryStock, 0, len(products))
		for _, p := range products {
			stocks = append(stocks, inventory.InventoryStock{ProductID: p.ID, Quantity: 50})
		}
		if err := db.CreateInBatches(&stocks, productsPerCategory).Error; err != nil {
			log.Println("seed: stock:", err)
		}
		total += len(products)
	}
	log.Printf("seed: %d products created", total)
}

func seedBlogs(db *gorm.DB) {
	if count(db, &cms.BlogPost{}) > 0 {
		return
	}
	var author user.User
	db.Where("roles @> ?::jsonb", user.RoleFilter(user.RoleMarketer)).First(&author)
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
		b := cms.Banner{Title: fmt.Sprintf("Banner %d", i), MediaID: &m.ID, AltName: fmt.Sprintf("Promotional banner %d", i), LinkUrl: "/products", DisplayOrder: i, IsActive: true}
		if err := db.Create(&b).Error; err != nil {
			log.Println("seed: banner:", err)
			return
		}
	}
	log.Println("seed: banners created")
}
