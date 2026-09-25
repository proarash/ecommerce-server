package finance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/proarash/ecommerce-server/internal/notification"
	"gorm.io/gorm"
)

var ErrEmptyOrder = errors.New("order has no items")

type Store interface {
	CreateOrderTx(tx *gorm.DB, userID uint, lines []OrderLine) (Order, PreInvoice, error)
	NotifyOrderPlaced(ctx context.Context, order Order, pre PreInvoice)
	FindOrder(ctx context.Context, id uint) (Order, error)
	FindUserOrder(ctx context.Context, id, userID uint) (Order, error)
	ListOrders(ctx context.Context, q ListQuery) ([]Order, int64, error)
	UpdateOrderStatus(ctx context.Context, id uint, status string) error
	MarkPaid(ctx context.Context, orderID uint) error
	CreatePreInvoice(ctx context.Context, dto CreatePreInvoiceDto, issuedBy string) (PreInvoice, error)
	ListPreInvoices(ctx context.Context, q ListQuery) ([]PreInvoice, int64, error)
	Report(ctx context.Context, q ReportQuery) (ReportResponse, error)
}

type store struct {
	db       *gorm.DB
	notifier *notification.Service
}

func NewStore(db *gorm.DB, notifier *notification.Service) Store {
	return &store{db: db, notifier: notifier}
}

func invoiceNumber(orderID uint) string {
	return fmt.Sprintf("INV-%d-%d", orderID, time.Now().UnixNano())
}

func (s *store) CreateOrderTx(tx *gorm.DB, userID uint, lines []OrderLine) (Order, PreInvoice, error) {
	if len(lines) == 0 {
		return Order{}, PreInvoice{}, ErrEmptyOrder
	}
	order := Order{UserID: userID, Status: OrderPending}
	for _, l := range lines {
		order.Items = append(order.Items, OrderItem{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		order.TotalAmount += l.UnitPrice * float64(l.Quantity)
	}
	if err := tx.Create(&order).Error; err != nil {
		return Order{}, PreInvoice{}, err
	}
	pre := PreInvoice{OrderID: order.ID, UserID: userID, InvoiceNumber: invoiceNumber(order.ID), Amount: order.TotalAmount, IssuedBy: IssuedBySystem, Status: PreInvoiceIssued}
	if err := tx.Create(&pre).Error; err != nil {
		return Order{}, PreInvoice{}, err
	}
	return order, pre, nil
}

func (s *store) notify(ctx context.Context, p notification.AutomatedMessagePayload) {
	if err := s.notifier.SendAutomatedMessage(ctx, p); err != nil {
		log.Println("finance: notify:", err)
	}
}

func (s *store) NotifyOrderPlaced(ctx context.Context, order Order, pre PreInvoice) {
	s.notify(ctx, notification.AutomatedMessagePayload{
		UserID:    order.UserID,
		Title:     "Invoice issued",
		Message:   fmt.Sprintf("Your order #%d was placed. Total: %.0f", order.ID, order.TotalAmount),
		Type:      notification.MsgInvoice,
		ExtraData: map[string]interface{}{"order_id": order.ID, "total_amount": order.TotalAmount},
	})
	s.notifyPreInvoice(ctx, pre)
}

func (s *store) notifyPreInvoice(ctx context.Context, pre PreInvoice) {
	s.notify(ctx, notification.AutomatedMessagePayload{
		UserID:    pre.UserID,
		Title:     "Pre-invoice issued",
		Message:   fmt.Sprintf("Pre-invoice %s for order #%d. Amount: %.0f", pre.InvoiceNumber, pre.OrderID, pre.Amount),
		Type:      notification.MsgPreInvoice,
		ExtraData: map[string]interface{}{"order_id": pre.OrderID, "invoice_number": pre.InvoiceNumber, "amount": pre.Amount},
	})
}

func (s *store) FindOrder(ctx context.Context, id uint) (Order, error) {
	return gorm.G[Order](s.db).Preload("Items", nil).Preload("Items.Product", nil).Where("id = ?", id).First(ctx)
}

func (s *store) FindUserOrder(ctx context.Context, id, userID uint) (Order, error) {
	return gorm.G[Order](s.db).Preload("Items", nil).Preload("Items.Product", nil).Where("id = ? AND user_id = ?", id, userID).First(ctx)
}

func (s *store) ListOrders(ctx context.Context, q ListQuery) ([]Order, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Order{})
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.UserID != 0 {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Order
	err := tx.Preload("Items").Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) UpdateOrderStatus(ctx context.Context, id uint, status string) error {
	res := s.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) MarkPaid(ctx context.Context, orderID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		if err := tx.Model(&order).Update("status", OrderPaid).Error; err != nil {
			return err
		}
		res := tx.Model(&PreInvoice{}).Where("order_id = ? AND status = ?", orderID, PreInvoiceIssued).Update("status", PreInvoicePaid)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			pre := PreInvoice{OrderID: orderID, UserID: order.UserID, InvoiceNumber: invoiceNumber(orderID), Amount: order.TotalAmount, IssuedBy: IssuedBySystem, Status: PreInvoicePaid}
			return tx.Create(&pre).Error
		}
		return nil
	})
}

func (s *store) CreatePreInvoice(ctx context.Context, dto CreatePreInvoiceDto, issuedBy string) (PreInvoice, error) {
	order, err := gorm.G[Order](s.db).Where("id = ?", dto.OrderID).First(ctx)
	if err != nil {
		return PreInvoice{}, err
	}
	amount := order.TotalAmount
	if dto.Amount != nil {
		amount = *dto.Amount
	}
	pre := PreInvoice{OrderID: order.ID, UserID: order.UserID, InvoiceNumber: invoiceNumber(order.ID), Amount: amount, IssuedBy: issuedBy, Status: PreInvoiceIssued}
	if err := gorm.G[PreInvoice](s.db).Create(ctx, &pre); err != nil {
		return PreInvoice{}, err
	}
	s.notifyPreInvoice(ctx, pre)
	return pre, nil
}

func (s *store) ListPreInvoices(ctx context.Context, q ListQuery) ([]PreInvoice, int64, error) {
	tx := s.db.WithContext(ctx).Model(&PreInvoice{})
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.UserID != 0 {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PreInvoice
	err := tx.Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

var periodTrunc = map[string]string{"daily": "day", "weekly": "week", "monthly": "month", "annually": "year"}
var periodDefaultRange = map[string]time.Duration{"daily": 30 * 24 * time.Hour, "weekly": 12 * 7 * 24 * time.Hour, "monthly": 365 * 24 * time.Hour, "annually": 5 * 365 * 24 * time.Hour}

func (s *store) Report(ctx context.Context, q ReportQuery) (ReportResponse, error) {
	to := time.Now()
	if q.To != nil {
		to = q.To.Add(24 * time.Hour)
	}
	from := to.Add(-periodDefaultRange[q.Period])
	if q.From != nil {
		from = *q.From
	}
	res := ReportResponse{Period: q.Period, Rows: []ReportRow{}}
	err := s.db.WithContext(ctx).Model(&Order{}).
		Select("date_trunc(?, created_at) AS period_start, COUNT(*) AS orders_count, COALESCE(SUM(total_amount), 0) AS total_amount", periodTrunc[q.Period]).
		Where("status IN ? AND created_at >= ? AND created_at < ?", []string{OrderPaid, OrderProcessing, OrderDelivered}, from, to).
		Group("period_start").Order("period_start").
		Scan(&res.Rows).Error
	for _, r := range res.Rows {
		res.OrdersCount += r.OrdersCount
		res.TotalAmount += r.TotalAmount
	}
	return res, err
}
