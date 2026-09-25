package payment

type ZibalRequest struct {
	Merchant    string `json:"merchant"`
	Amount      int64  `json:"amount"`
	CallbackURL string `json:"callbackUrl"`
	Description string `json:"description,omitempty"`
	OrderID     string `json:"orderId,omitempty"`
	Mobile      string `json:"mobile,omitempty"`
}

type ZibalRequestResponse struct {
	TrackID int64  `json:"trackId"`
	Result  int    `json:"result"`
	Message string `json:"message"`
}

type ZibalTrackRequest struct {
	Merchant string `json:"merchant"`
	TrackID  int64  `json:"trackId"`
}

type ZibalVerifyResponse struct {
	PaidAt      string `json:"paidAt"`
	CardNumber  string `json:"cardNumber"`
	Status      int    `json:"status"`
	Amount      int64  `json:"amount"`
	RefNumber   *int64 `json:"refNumber"`
	Description string `json:"description"`
	OrderID     string `json:"orderId"`
	Result      int    `json:"result"`
	Message     string `json:"message"`
}

type ZibalInquiryResponse struct {
	CreatedAt   string `json:"createdAt"`
	PaidAt      string `json:"paidAt"`
	Verified    bool   `json:"verified"`
	Status      int    `json:"status"`
	Amount      int64  `json:"amount"`
	RefNumber   *int64 `json:"refNumber"`
	Description string `json:"description"`
	CardNumber  string `json:"cardNumber"`
	OrderID     string `json:"orderId"`
	Wage        int64  `json:"wage"`
	Result      int    `json:"result"`
	Message     string `json:"message"`
}

type CheckoutResponse struct {
	TrackID    int64  `json:"track_id"`
	PaymentURL string `json:"payment_url"`
}

type InquiryResponse struct {
	Transaction PaymentTransaction   `json:"transaction"`
	Gateway     ZibalInquiryResponse `json:"gateway"`
}

type CallbackQuery struct {
	Success int    `form:"success"`
	Status  int    `form:"status"`
	TrackID int64  `form:"trackId" binding:"required"`
	OrderID string `form:"orderId"`
}
