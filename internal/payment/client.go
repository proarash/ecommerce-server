package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	zibalBaseURL = "https://gateway.zibal.ir"

	ResultSuccess         = 100
	ResultAlreadyVerified = 201
)

type Client struct {
	merchant    string
	callbackURL string
	http        *http.Client
}

func NewClient(merchant, callbackURL string) *Client {
	return &Client{merchant: merchant, callbackURL: callbackURL, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zibalBaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 500 {
		return fmt.Errorf("zibal: http %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) Request(ctx context.Context, amount int64, orderID, description, mobile string) (ZibalRequestResponse, error) {
	var out ZibalRequestResponse
	err := c.post(ctx, "/v1/request", ZibalRequest{Merchant: c.merchant, Amount: amount, CallbackURL: c.callbackURL, Description: description, OrderID: orderID, Mobile: mobile}, &out)
	if err == nil && out.Result != ResultSuccess {
		err = fmt.Errorf("zibal request failed: %d %s", out.Result, out.Message)
	}
	return out, err
}

func (c *Client) Verify(ctx context.Context, trackID int64) (ZibalVerifyResponse, error) {
	var out ZibalVerifyResponse
	err := c.post(ctx, "/v1/verify", ZibalTrackRequest{Merchant: c.merchant, TrackID: trackID}, &out)
	return out, err
}

func (c *Client) Inquiry(ctx context.Context, trackID int64) (ZibalInquiryResponse, error) {
	var out ZibalInquiryResponse
	err := c.post(ctx, "/v1/inquiry", ZibalTrackRequest{Merchant: c.merchant, TrackID: trackID}, &out)
	return out, err
}

func StartURL(trackID int64) string {
	return fmt.Sprintf("%s/start/%d", zibalBaseURL, trackID)
}
