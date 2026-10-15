package finance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/proarash/ecommerce-server/internal/notification"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrEmptyOrder      = errors.New("order has no items")
	ErrOrderNotPayable = errors.New("order is not payable in its current status")
	ErrOrderNotReviewable = errors.New("only paid orders can be confirmed or rejected")
)

type Store interface {
	CreateOrderTx(tx *gorm.DB, userID uint, lines []OrderLine, discount *OrderDiscount) (Order, PreInvoice, error)
	NotifyOrderPlaced(ctx context.Context, order Order, pre PreInvoice)
	FindOrder(ctx context.Context, id uint) (Order, error)
	FindUserOrder(ctx context.Context, id, userID uint) (Order, error)
	ListOrders(ctx context.Context, q ListQuery) ([]Order, int64, error)
	MarkFailed(ctx context.Context, id uint) error
	ReviewOrder(ctx context.Context, id, reviewerID uint, accept bool, message string, refund func(tx *gorm.DB, order Order) error) (Order, error)
	NotifyOrderReviewed(ctx context.Context, order Order)
	MarkPaid(ctx context.Context, orderID uint) error
	MarkPaidTx(tx *gorm.DB, orderID uint) error
	LockPayableOrderTx(tx *gorm.DB, orderID, userID uint) (Order, error)
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

func (s *store) CreateOrderTx(tx *gorm.DB, userID uint, lines []OrderLine, discount *OrderDiscount) (Order, PreInvoice, error) {
	if len(lines) == 0 {
		return Order{}, PreInvoice{}, ErrEmptyOrder
	}
	order := Order{UserID: userID, Status: OrderPending}
	for _, l := range lines {
		order.Items = append(order.Items, OrderItem{ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		order.TotalAmount += l.UnitPrice * float64(l.Quantity)
	}
	if discount != nil {
		order.DiscountID = &discount.ID
		order.DiscountAmount = min(discount.Amount, order.TotalAmount)
		order.TotalAmount -= order.DiscountAmount
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
	tx = s.filterUsers(tx, q)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Order
	err := tx.Preload("Items.Product").Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint, len(items))
	for i, o := range items {
		ids[i] = o.UserID
	}
	users := s.usersByID(ctx, ids)
	for i := range items {
		items[i].User = users[items[i].UserID]
	}
	return items, total, nil
}

func (s *store) filterUsers(tx *gorm.DB, q ListQuery) *gorm.DB {
	if q.UserID != 0 {
		tx = tx.Where("user_id = ?", q.UserID)
	}
	if q.Search != "" {
		like := "%" + q.Search + "%"
		tx = tx.Where("user_id IN (?)", s.db.Table("users").Select("id").Where("name ILIKE ? OR mobile ILIKE ?", like, like))
	}
	return tx
}

func (s *store) usersByID(ctx context.Context, ids []uint) map[uint]*UserBrief {
	result := map[uint]*UserBrief{}
	if len(ids) == 0 {
		return result
	}
	var users []UserBrief
	if err := s.db.WithContext(ctx).Table("users").Select("id, name, mobile").Where("id IN ?", ids).Scan(&users).Error; err != nil {
		return result
	}
	for i := range users {
		result[users[i].ID] = &users[i]
	}
	return result
}

func (s *store) ReviewOrder(ctx context.Context, id, reviewerID uint, accept bool, message string, refund func(tx *gorm.DB, order Order) error) (Order, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != OrderPaid {
			return ErrOrderNotReviewable
		}
		fields := map[string]any{"reviewed_by_id": reviewerID, "reviewed_at": time.Now()}
		if accept {
			fields["status"] = OrderConfirmed
		} else {
			if err := refund(tx, order); err != nil {
				return err
			}
			fields["status"] = OrderRejected
			fields["reject_reason"] = message
		}
		return tx.Model(&order).Updates(fields).Error
	})
	if err != nil {
		return Order{}, err
	}
	return s.FindOrder(ctx, id)
}

func (s *store) NotifyOrderReviewed(ctx context.Context, order Order) {
	title := "Order confirmed"
	message := fmt.Sprintf("Your order #%d was confirmed.", order.ID)
	if order.Status == OrderRejected {
		title = "Order rejected"
		message = fmt.Sprintf("Your order #%d was rejected: %s. %.0f was refunded to your wallet.", order.ID, order.RejectReason, order.TotalAmount)
	}
	s.notify(ctx, notification.AutomatedMessagePayload{
		UserID:    order.UserID,
		Title:     title,
		Message:   message,
		Type:      notification.MsgOrderReviewed,
		ExtraData: map[string]interface{}{"order_id": order.ID, "status": order.Status},
	})
}

func (s *store) MarkFailed(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&Order{}).
		Where("id = ? AND status IN ?", id, []string{OrderPending, OrderFailed}).
		Update("status", OrderFailed).Error
}

func (s *store) MarkPaid(ctx context.Context, orderID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.MarkPaidTx(tx, orderID)
	})
}

func (s *store) MarkPaidTx(tx *gorm.DB, orderID uint) error {
	var order Order
	if err := tx.First(&order, orderID).Error; err != nil {
		return err
	}
	if err := tx.Model(&order).Update("status", OrderPaid).Error; err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM cart_items WHERE cart_id IN (SELECT id FROM carts WHERE user_id = ?) AND product_id IN (SELECT product_id FROM order_items WHERE order_id = ?)", order.UserID, orderID).Error; err != nil {
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
}

func (s *store) LockPayableOrderTx(tx *gorm.DB, orderID, userID uint) (Order, error) {
	var order Order
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return Order{}, err
	}
	if order.Status != OrderPending && order.Status != OrderFailed {
		return Order{}, ErrOrderNotPayable
	}
	return order, nil
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
	tx = s.filterUsers(tx, q)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PreInvoice
	err := tx.Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint, len(items))
	for i, p := range items {
		ids[i] = p.UserID
	}
	users := s.usersByID(ctx, ids)
	for i := range items {
		items[i].User = users[items[i].UserID]
	}
	return items, total, nil
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
	res := ReportResponse{Period: q.Period, Rows: []ReportRow{}, ProductRows: []ProductSalesRow{}}
	statuses := []string{OrderPaid, OrderConfirmed, OrderProcessing, OrderDelivered}
	err := s.db.WithContext(ctx).Model(&Order{}).
		Select("date_trunc(?, created_at) AS period_start, COUNT(*) AS orders_count, COALESCE(SUM(total_amount), 0) AS total_amount", periodTrunc[q.Period]).
		Where("status IN ? AND created_at >= ? AND created_at < ?", statuses, from, to).
		Group("period_start").Order("period_start").
		Scan(&res.Rows).Error
	if err != nil {
		return res, err
	}
	for _, r := range res.Rows {
		res.OrdersCount += r.OrdersCount
		res.TotalAmount += r.TotalAmount
	}
	err = s.db.WithContext(ctx).Raw(`SELECT date_trunc(?, o.created_at) AS period_start, oi.product_id AS product_id, p.title AS product_title, p.sku AS product_sku,
			SUM(oi.quantity) AS quantity, COUNT(DISTINCT oi.order_id) AS orders_count, COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS total_amount
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN products p ON p.id = oi.product_id
		WHERE o.status IN ? AND o.created_at >= ? AND o.created_at < ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL
		GROUP BY period_start, oi.product_id, p.title, p.sku
		ORDER BY period_start, total_amount DESC`, periodTrunc[q.Period], statuses, from, to).Scan(&res.ProductRows).Error
	return res, err
}
