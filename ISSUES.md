# Implementation Review: Issues to Fix

Snapshot reviewed: branch `main`, HEAD `36e3815` plus the uncommitted working-tree changes present on 2026-09-26.
Method: static reading of every source file against `plan.md`. Nothing was run against PostgreSQL, MinIO or Zibal.
Line numbers refer to that snapshot. If they have shifted, find the code by the function or type names given.

---

## 0. Rules that apply to every fix

These come from `CLAUDE.md` and override convenience:

1. **No comments** in source files. The only exception is Swagger annotations on handlers and in `cmd/main.go`. `// TODO` lines and commented-out code count as comments.
2. **No new third-party Go packages.** Only the libraries in `plan.md` §1 (already in `go.mod`) plus the Go standard library.
3. **Keep the `plan.md` §2 structure and architecture.** When a fix needs an endpoint, env var, model field or file that `plan.md` does not list, update `plan.md` in the same change. Each issue says what to add. If you can ask the user, confirm new endpoints before building them.
4. **Fetch official docs with `curl` before using a package API.** Section 5 lists the relevant URLs.
5. **Full Swagger annotations on every new or changed handler:** `@Summary`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Security BearerAuth` when authenticated, and `@Router`. Document JSON responses as `types.ApiResponse{data=...}`. The exception is routes skipped by `ApiResponseMiddleware` (`/ws`, `/swagger`, `/api/payment/callback`), which return unwrapped bodies.
6. **Regenerate Swagger docs** with `make swag`, which runs `swag init -g cmd/main.go -o ./docs --parseInternal --parseDependency --parseDepth 2`. Never hand-edit `docs/`.
7. **New env vars** go into `internal/config/env.go`, all three `dev.example.*` scripts and `docker-compose.yml`. Do not edit the gitignored local copies `dev.sh`, `dev.bat` and `dev.ps1`.

Work through the checklist in order. H1–H4 share one target design, described under H1; implement them together.

---

## 1. Checklist

### P0: Blockers (the app cannot work as planned)
- [ ] **B1** Auto-migrations are disabled and the DB ping is suppressed
- [ ] **B2** Seeder is disabled, so no admin can ever exist
- [ ] **B3** Media module (MinIO upload and get) is not registered
- [ ] **B4** `cmd/bootstrap.go` violates the no-comments rule

### P1: High (payment and stock correctness)
- [ ] **H1** Payment callback trusts `success`/`status` from the URL (target design for H1–H4)
- [ ] **H2** Verify network errors mark the order failed, and there is no reconciliation path
- [ ] **H3** Settling the payment and updating the order are not atomic
- [ ] **H4** The same order can be paid twice
- [ ] **H5** Reserved stock is never released (no cancellation, no expiry)
- [ ] **H6** Order lifecycle (`processing`/`delivered`/`cancelled`) cannot be reached

### P2: Medium
- [ ] **M1** Disabled or deleted accounts keep access until their JWT expires
- [ ] **M2** Customer WebSocket stays bound to a stale chat room
- [ ] **M3** Bot-only chat rooms flood the support console and admin stats
- [ ] **M4** Media upload accepts any content type into a public bucket
- [ ] **M5** Telegram bot token and JWTs leak into logs
- [ ] **M6** Soft delete plus full unique indexes block reuse of titles, SKUs, slugs and keys
- [ ] **M7** Checkout can deadlock on stock row locks
- [ ] **M8** `PATCH /api/cms/content` wipes fields that were omitted
- [ ] **M9** `docker-compose.yml` cannot run the app
- [ ] **M10** 500 responses leak internal error details

### P3: Low
- [ ] **L1** `internal/types` is not in the plan structure (decision)
- [ ] **L2** Sync `plan.md` with justified model deviations
- [ ] **L3** `GET /api/inventory` omits products without a stock row
- [ ] **L4** Inactive products and banners: public leak and staff blind spot
- [ ] **L5** Unknown media IDs are silently dropped
- [ ] **L6** WebSocket upgrader accepts any Origin
- [ ] **L7** Pre-invoice message lacks the link and item breakdown
- [ ] **L8** `telegram_chat_id` accepts any chat, including groups
- [ ] **L9** Staff avatar fields are not mutually exclusive
- [ ] **L10** Cart item races, quantity limit, and a vague checkout error
- [ ] **L11** Duplicate open chat rooms under concurrency
- [ ] **L12** Order history loses soft-deleted products
- [ ] **L13** Payment amount truncation and stale mobile
- [ ] **L14** Telegram broadcast is not throttled
- [ ] **L15** Admin notification can target a nonexistent user
- [ ] **L16** No brute-force protection on login

---

## 2. P0: Blockers

### B1: Auto-migrations are disabled and the DB ping is suppressed

**Where:** `cmd/bootstrap.go:38-68` (`Bootstrap`), imports at lines 4, 17-18 and 23-24.

**Problem:** The `db.AutoMigrate(...)` call is commented out, and `gorm.Config` sets `DisableAutomaticPing: true`.
- On a fresh database no tables exist, so every endpoint that touches the DB returns 500.
- With the ping disabled, the server starts and looks healthy even when PostgreSQL is unreachable.

**Plan:** §2 (`bootstrap.go`: "DB init, auto-migrations, admin seeder, route registration"), §4.

**Fix:**
1. Restore `AutoMigrate` with every model, keeping `media.Media` before the models that reference it:
   `media.Media`, `staff.StaffUser`, `user.User`, `product.Category`, `product.Product`, `product.Attribute`, `cart.Cart`, `cart.CartItem`, `finance.Order`, `finance.OrderItem`, `finance.PreInvoice`, `payment.PaymentTransaction`, `inventory.InventoryStock`, `inventory.InventoryLog`, `cms.BlogPost`, `cms.Banner`, `cms.SiteContent`, `chat.ChatRoom`, `chat.ChatMessage`, `notification.Notification`, `notification.NotificationRead`.
   `NotificationRead` is required by `notification.store.ListForUser` and `MarkRead`, and the committed HEAD version of the list omitted it.
2. Remove `DisableAutomaticPing: true`. Keep `TranslateError: true`, because `types.HandleError` relies on `gorm.ErrDuplicatedKey` and `gorm.ErrForeignKeyViolated`.
3. Restore the imports and delete every TODO and commented-out line (see B4).

**Acceptance:**
- On an empty database, the first boot creates all 21 tables plus the join tables `product_media`, `product_attributes` and `blog_media`.
- With an unreachable DSN, the process exits with an error instead of serving requests.

---

### B2: Seeder is disabled, so no admin can ever exist

**Where:** `cmd/bootstrap.go:69-70`, `internal/seed/seed.go:50-65` (`seedAdmin`).

**Problem:**
- `seed.Run` is commented out, and nothing else can create an admin: `POST /api/admin/staff` rejects role `admin` and itself requires an admin. As a result every `/api/admin/*` route (staff management, notification CRUD, stats) is unreachable.
- `seedAdmin` silently skips when `ADMIN_MOBILE` or `ADMIN_PASSWORD` is empty.
- `seedAdmin` never checks the values against the login DTO (`auth.LoginDto`: mobile `numeric,len=11`, password `min=6,max=72`). An admin seeded with other values can never log in.

**Plan:** §10.

**Fix:**
1. Call `seed.Run(db, envConfig)` right after `AutoMigrate`.
2. In `seedAdmin`, handle the case where no admin exists and the env values are missing or invalid (mobile not 11 digits, password length outside 6–72 bytes):
   - in production, `log.Fatal`;
   - otherwise, log a clear warning.

**Acceptance:**
- The first boot with valid `ADMIN_*` values creates exactly one admin, who can log in via `POST /api/auth/login`.
- Restarting does not create duplicates.
- A non-production boot seeds:
  - 2 storekeepers, 3 accountants, 2 marketers and 5 support agents (`DefaultAvatarID` 1–5);
  - 6 categories (2 roots and 4 children);
  - 100 products with stock rows;
  - 30 blog posts, the site content defaults and 3 banners.
- A production boot seeds only the admin.

---

### B3: Media module is not registered

**Where:** `cmd/bootstrap.go:4, 18, 89, 107-108, 113`.

**Problem:** The media route group, the MinIO client and the handler registration are all commented out.
- `POST /api/media/upload` and `GET /api/media/:id` return 404, yet Swagger still documents them.
- No media ID can ever be created for products, categories, blogs, banners, site content or staff avatars.

**Plan:** §7, and §11 "Media Management (roles: admin, storekeeper, marketer, support)".

**Fix:** Restore the three statements:
```go
mediaGroup := api.Group("", authMW, middleware.RequireRoles(staff.RoleStorekeeper, staff.RoleMarketer, staff.RoleSupport))
mediaClient := media.NewClient(context.Background(), envConfig)
media.NewHandler(media.NewStore(db), mediaClient).RegisterRoutes(mediaGroup)
```
- Admin gets access through the bypass in `RequireRoles`.
- Keep the existing `media.NewClient` behaviour: when `MINIO_ENDPOINT` is empty, uploads return 503.
- Apply M4 to the upload handler in the same pass.

**Acceptance:**
- The routes appear in gin's route list.
- With MinIO configured, an upload returns 201 with a public URL.
- With no `MINIO_ENDPOINT`, an upload returns 503.
- A customer token gets 403.

---

### B4: `cmd/bootstrap.go` violates the no-comments rule

**Where:** `cmd/bootstrap.go` lines 4, 17-18, 23-24, 38, 43-70, 89, 107-108 and 113.

**Fix:** After B1–B3, delete every remaining comment and commented-out line.

**Acceptance:** This command prints nothing:
```sh
grep -rnE '^\s*//' --include=*.go cmd internal pkg | grep -vE '//\s*@|godoc$'
```

---

## 3. P1: High

### H1: Payment callback trusts `success`/`status` from the URL (target design for H1–H4)

**Where:** `internal/payment/handler.go:130-190` (`Callback`), `internal/payment/store.go:36-47` (`Settle`), `internal/finance/store.go:113-138` (`MarkFailed`, `MarkPaid`).

**Problem:**
- When `success != 1`, the handler never contacts Zibal. It copies `status` from the query string into the transaction, settles it, marks the order failed and sends "payment failed".
- `GET /api/payment/callback` is public, so anyone holding a trackId can call `?trackId=<id>&success=0` before the customer pays.
- When Zibal's real redirect then arrives, the `tx.Status != StatusPending` check (line 142) returns early. The payment is never verified and the order never becomes paid.
- `PaymentTransaction.Status` must only ever come from Zibal API responses, never from the URL.

**Plan:** §9, workflow steps 3–5.

**Docs first:** Fetch the Zibal IPG docs (section 5) and confirm the following. If the docs differ, adjust the constants in `internal/payment/client.go`.
- `/v1/verify` result codes: 100 success, 201 already verified, 202 not paid or unsuccessful, 203 invalid trackId, 102–104 merchant errors.
- Transaction status codes: -1 pending, 1 paid and verified, 2 paid but unverified, 3 cancelled by the user, and so on.
- The callback query parameters (`success`, `trackId`, `orderId`, `status`).
- That unverified payments are reversed to the payer automatically.

**Target design.** One code path, used by the callback, the inquiry endpoint (H2) and the expiry job (H5):

1. **Bind only `trackId`** (required). `success`, `status` and `orderId` may be logged but must not drive any decision. Either remove them from `CallbackQuery` or keep them documented in Swagger as informational only.
2. **Load the payment transaction** by trackId.
   - Not found: 404 JSON, unchanged.
   - Status is not -1: redirect immediately (idempotent).
3. **Open one DB transaction.**
   - Lock the order row with `SELECT ... FOR UPDATE` (`clause.Locking{Strength: "UPDATE"}`, as `cart.Checkout` already does).
   - Re-read the payment transaction `FOR UPDATE`.
   - If the payment transaction is no longer pending, commit and redirect.
4. **If the order status is not `pending` or `failed`**, do not call verify. This covers orders already paid by another transaction, cancelled orders and expired orders.
   - Call `/v1/inquiry` and store the gateway's `status`/`result` on the payment transaction.
   - Leave the order unchanged, commit, redirect.
   - Zibal reverses the unverified payment.
5. **Otherwise call `/v1/verify`** while holding the lock. The wait is bounded by the 15 s client timeout, and contention is limited to payments for the same order. Classify the response:

   | Verify outcome | Handling |
   | :--- | :--- |
   | Transport error, undecodable body, HTTP 5xx, or a result outside {100, 201, 202, 203} | Roll back and redirect. The payment stays pending and the order is **not** marked failed; H2/H5 reconcile it later. |
   | Result 100 or 201 and `amount == tx.Amount` | Success. |
   | Result 100 or 201 with an amount mismatch | Money was captured but does not match. Settle the payment with the verify data, do **not** mark the order paid, log an error with trackId and orderId, and alert the admin Telegram chat. Add a small notifier method that calls `TelegramClient.SendAdmin`. |
   | Result 202 or 203 | Failure. |

6. **Write both updates in the same DB transaction:**
   - Update the payment transaction with status, result, ref_number, card_number and paid_at from the verify response, keeping the `status = -1` guard.
   - Call `MarkPaidTx` or `MarkFailedTx` on the order.
   - Commit.
7. **Only after the commit**, send the `payment_success` or `payment_failed` automated message and redirect with `?inv=inv-<ORDER-ID>`.

**Suggested shape** (inside the existing files, per plan §2):
- `finance.Store`:
  - `LockOrderTx(tx *gorm.DB, id uint) (Order, error)`
  - `MarkPaidTx(tx *gorm.DB, orderID uint) error`: transitions only from `pending`/`failed` and returns an error otherwise. The pre-invoice logic stays: mark `issued` pre-invoices paid, and create a paid one only when none exists.
  - `MarkFailedTx(tx *gorm.DB, orderID uint) error`
- `payment.Store`:
  - a method that runs a callback inside one DB transaction with the payment row locked, for example `Process(ctx, trackID, fn func(tx *gorm.DB, pt *PaymentTransaction) error) error`;
  - a guarded `SaveSettledTx`.
- `payment.Handler`: one exported method, for example `Resolve(ctx, trackID) (PaymentTransaction, Outcome, error)`, that implements steps 2–6. `Callback`, `Inquiry` and the expiry job all call it.

**Acceptance:**
- A request with `?success=0` or any `status` value never settles a payment without a Zibal API call, and never overrides Zibal's status.
- An unpaid trackId: verify returns 202, the payment is settled with Zibal's status, the order becomes `failed` and a `payment_failed` message is sent.
- A paid trackId: the order becomes `paid`, the pre-invoice becomes `paid` and `payment_success` is sent. Calling the callback again only redirects.

---

### H2: Verify network errors mark the order failed, and there is no reconciliation path

**Where:** `internal/payment/handler.go:149-170`, and `Inquiry` at `:227-243`.

**Problem:**
- When `Verify` returns an error, `success` stays false and lines 168-170 copy the query `status` into the payment and settle it.
- The order is marked failed even though the customer may have paid; Zibal then reverses the money.
- A stuck payment cannot be repaired, because `POST /api/payment/inquiry/:trackId` only reads.

**Fix:**
1. Covered by H1 step 5: transport or unknown errors leave the payment pending.
2. Make `POST /api/payment/inquiry/:trackId` (accountant and admin) reconcile when the local payment is still -1. Call `/v1/inquiry`, then act on the gateway status:
   - 2 (paid, unverified): run the H1 flow (verify and settle).
   - 1 (paid, verified): settle as success after the same amount check. Running the H1 flow also works, because verify returns 201.
   - A final failure status (3 and above): settle as failed and call `MarkFailedTx`.
   - Still -1: leave it.
3. Return the updated local transaction together with the gateway response (`InquiryResponse`), and update the Swagger `@Description` to say the endpoint reconciles.

**Acceptance:**
- When verify fails (for example, the gateway is unreachable), the payment stays -1 and the order stays `pending`.
- A later inquiry for that paid trackId marks the order `paid`.

---

### H3: Settling the payment and updating the order are not atomic

**Where:** `internal/payment/handler.go:171-186`.

**Problem:**
- `Settle` commits first; `MarkPaid` and `MarkFailed` errors are only logged.
- If `MarkPaid` fails, the payment is already final (the `status = -1` guard blocks any retry), but the order stays `pending` forever.

**Fix:** Covered by H1 step 6: both writes happen in one DB transaction, and notifications are sent after the commit.

**Acceptance:** If `MarkPaidTx` is forced to fail, the payment stays -1 and retrying the callback completes normally.

---

### H4: The same order can be paid twice

**Where:** `internal/payment/handler.go:59-87` (`Checkout`), `internal/finance/store.go:119-138` (`MarkPaid`).

**Problem:**
- `Checkout` lets one order have any number of pending payment transactions.
- If the customer completes two of them, both callbacks verify, so the money is captured twice.
- `MarkPaid` has no status guard. The second run finds no `issued` pre-invoice (the update matches 0 rows), so its fallback inserts a duplicate `paid` pre-invoice.

**Fix:**
1. H1 steps 3–4: lock the order row and require status `pending`/`failed` before calling verify. The second payment is then never verified, and Zibal reverses it.
2. Add the `MarkPaidTx` status guard from H1.
3. In `Checkout`, lock the order row while checking its status and creating the payment transaction, so a checkout cannot race a callback. Allowing a new checkout while an older transaction is still pending is fine, since the customer may have abandoned the gateway page.

**Acceptance:** Two trackIds for one order, both paid in the sandbox, result in:
- the order paid once;
- exactly one paid pre-invoice;
- the second payment settled without verify.

---

### H5: Reserved stock is never released

**Where:** `internal/cart/store.go:140-144` (stock dispatched at checkout), `internal/finance/store.go:113-117` (`MarkFailed`), `internal/inventory/store.go`.

**Problem:**
- Checkout removes stock immediately through `inventory.DispatchTx`, and nothing ever puts it back.
- Failed payments, abandoned pending orders and cancellations all leak stock permanently.
- There is no cancel operation, and pending orders never expire.

**Plan:** §1 (inventory tracking), §4.5 (`cancelled` status), §8.

**Fix:**
1. **Restock helper.** In `internal/inventory/store.go`, add `RestockTx(tx *gorm.DB, productID uint, quantity int, reason string) error`, mirroring `DispatchTx`: `lockStock`, add the quantity, and write an `inbound` `InventoryLog` with `CreatedBy: 0`.
   `lockStock` checks product existence with `tx.First(&product.Product{}, id)`, which fails for soft-deleted products. The restock path must use `Unscoped()` for that check so stock can be returned for deleted products.
2. **Cancellation in finance.** In `internal/finance/store.go`, add `CancelOrderTx(tx *gorm.DB, orderID uint, allowed []string) (Order, error)`. It must:
   - lock the order;
   - require the order status to be in `allowed`;
   - set the status to `cancelled`;
   - set the order's `issued` pre-invoices to `cancelled`;
   - call `inventory.RestockTx` for every `OrderItem`, with reason `order #<id> cancelled`.

   `finance` importing `inventory` creates no import cycle. Add a `CancelOrder(ctx, ...)` wrapper that runs `CancelOrderTx` in a transaction and then notifies the customer through `SendAutomatedMessage`. Add a constant such as `MsgOrderCancelled`, mapped to chat type `text` in `chatType` and to notification type `order` in `notificationType`.
3. **Customer endpoint** `POST /api/user/orders/:id/cancel`:
   - customer group, own orders only;
   - allowed from `pending` or `failed`;
   - registered in `finance.Handler.RegisterRoutes`, with full Swagger;
   - added to plan.md §11 "User / Customer Area".
4. **Expiry job.** Add env `ORDER_EXPIRY_MINUTES` (default `60`). In `Bootstrap`, start a goroutine with a one-minute `time.Ticker`. On each tick, load `pending`/`failed` orders whose `updated_at` is older than the expiry (add `finance.Store.StaleOrders(ctx, before time.Time, limit int)`). For each order:
   - reconcile its pending payment transactions through the H1/H2 flow (inquiry, then verify if paid but unverified);
   - if the order became paid, skip it;
   - if a payment is still -1 at the gateway and the order is younger than twice the expiry, skip it this round;
   - otherwise call `CancelOrder` with allowed statuses `pending`/`failed`.

   A payment that arrives after cancellation is handled by H1 step 4: it is not verified, so it is reversed.
5. Add `ORDER_EXPIRY_MINUTES` to the config, the dev scripts, compose and plan.md (§8 or §9).

**Acceptance:**
- After a failed payment, stock stays reserved and the customer can retry payment.
- Cancelling returns the stock with an `inbound` log, sets the pre-invoice to `cancelled` and sends a bot message.
- A stale order is cancelled and restocked automatically.
- Paid orders are never touched by the job.

---

### H6: Order lifecycle (`processing`/`delivered`/`cancelled`) cannot be reached

**Where:** `internal/finance/model.go:8-21`, `internal/finance/handler.go` (`RegisterRoutes`), `internal/finance/store.go:189` (report), `internal/admin/handler.go:148-157` (stats).

**Problem:**
- The statuses `processing`, `delivered` and `cancelled` exist (plan §4.5), and reports and stats count them, but no code path ever sets them.
- Paid orders can never be fulfilled or cancelled.

**Fix:** Add `PATCH /api/admin/orders/:id/status` in the admin group.
- **Body:** `{"status": "processing" | "delivered" | "cancelled"}`, validated with `oneof`, with Swagger `enums`.
- **Allowed transitions:**
  - `paid → processing`
  - `processing → delivered`
  - `pending | failed | paid | processing → cancelled`, through `CancelOrderTx` from H5 so stock is restored.
- **Cancellation after payment:** state in the customer message and in a log line that the refund must be processed manually; Zibal IPG refunds are outside this system.
- **Other transitions:** reject with 400.
- **Notifications:** notify the customer on every change.
- **Registration:** in `finance.Handler` (pass it the admin group) or in `admin.Handler`, whichever keeps the wiring simpler.
- **Plan:** add the route to plan.md §11 "Admin Management Area". If the user wants storekeepers to handle fulfilment, add their role to the guard.

**Acceptance:**
- The listed transitions work, and every other transition returns 400.
- Cancelling a paid order restores stock.
- Delivered orders appear in `GET /api/finance/reports`.

---

## 4. P2: Medium

### M1: Disabled or deleted accounts keep access until their JWT expires

**Where:** `internal/middleware/auth.go:29-47`, `internal/auth/store.go:57, 74`, `cmd/bootstrap.go:84`.

**Problem:**
- `Status` is checked only at login.
- `PATCH /api/admin/staff/:id/status` with `false` leaves the staff member fully authorized for up to 24 h (`tokenTTL`). Customers behave the same way.
- Role guards use the role stored in the token, never the current one.

**Fix:** `middleware` cannot import `staff` or `user`, because both already import `middleware`. Inject a checker instead:
```go
func Auth(secret string, check func(ctx context.Context, p *token.AuthPayload) (string, error)) gin.HandlerFunc
```
Build the closure in `Bootstrap`:
- staff tokens: `staffStore.FindByID`;
- customer tokens: `userStore.FindByID`;
- record not found: 401 `invalid token`;
- `Status == false`: 401 `account is disabled`;
- otherwise return the DB role (staff) or `token.RoleUser` (customer). The middleware overwrites `p.Role` with it before calling `c.Set`.

This covers `/ws` as well, because it uses the same middleware.

**Acceptance:**
- After a logged-in staff member is disabled, their next request gets 401.
- After they are re-enabled, requests work again.

---

### M2: Customer WebSocket stays bound to a stale chat room

**Where:** `internal/chat/handler.go:64-93` (`ServeWS`), `internal/chat/hub.go` (`Hub`, `Publish`, `UserMessage`, `BotMessage`), `internal/chat/client.go:76`.

**Problem:** The customer's socket stores `roomID` when it connects. After support closes that room:
- messages sent over the socket keep going into the closed room, because `UserMessage` never checks the room status;
- bot messages and new conversations go to a new room from `EnsureRoom`, which the socket is not subscribed to;
- the customer therefore stops receiving real-time messages until they reconnect.

**Fix:**
1. **Key customer connections by user ID.** Use `Hub.users map[uint]map[*Client]struct{}` and drop `Client.roomID` for customers.
2. **Publish by user.** Change the signature to `Hub.Publish(msg ChatMessage, userID uint)`. It sends to that user's connections and to all support connections. Update every `Service.Post` caller to pass the room's user ID: `UserMessage` and `BotMessage` already have it, and `SupportReply` has the loaded room.
3. **Resolve the room per message.** `Service.UserMessage(ctx, userID, text)` calls `EnsureRoom` on every message, getting the current non-closed room or creating one, and then posts. Both `readPump` and `UserSend` use it; remove the duplicated room logic from `UserSend`.
4. **Fire the support-inquiry event on the first user message.** Plan §5.1 calls this the "Support Event". Fire it when the first `user` message is posted into a room (no earlier `user` messages in that room), not when the room is created; otherwise rooms created by the bot never trigger it. Rename `OnRoomCreated` accordingly (for example `OnInquiryStarted`) and keep the welcome bot message.
5. **Simplify `ServeWS`.** It no longer needs `EnsureRoom` for customers.

**Acceptance:** With the customer connected:
1. support closes the room;
2. the customer sends a message on the same socket;
3. a new room is created and support is alerted;
4. the customer receives the support reply and later bot messages on that same socket.

---

### M3: Bot-only chat rooms flood the support console and admin stats

**Where:** `internal/chat/hub.go:129-136` (`BotMessage` → `EnsureRoom`), `internal/chat/hub.go:158-170` (`Rooms`), `internal/admin/handler.go:154` (`open_chats`).

**Problem:**
- Every order or payment bot message creates an `open` room for the customer.
- `GET /api/support/chat/rooms` and the admin `open_chats` stat then include rooms where the customer never wrote anything.

**Fix:** In `Service.Rooms` and in the `Stats` count, include only rooms that contain at least one user message:
```sql
EXISTS (SELECT 1 FROM chat_messages m WHERE m.room_id = chat_rooms.id AND m.sender_role = 'user' AND m.deleted_at IS NULL)
```
Keep creating the room itself, because plan §5.1 requires bot messages to go into the user's room.

**Acceptance:**
- An order placed without any chat does not appear in the support room list, and `open_chats` is unchanged.
- Once the customer writes, the room is listed.

---

### M4: Media upload accepts any content type into a public bucket

**Where:** `internal/media/handler.go:58-108` (`Upload`), `internal/media/client.go:46` (public-read bucket policy).

**Problem:**
- Any sniffed MIME type is accepted, including `text/html`, and SVG (which sniffs as `text/xml`).
- Files are stored with that content type in a publicly readable bucket and served from the MinIO origin. That allows stored XSS and phishing pages.
- The object extension comes from the client file name, so an HTML file renamed `a.jpg` is still served as HTML.
- The 100 MB limit is checked only after gin has read and parsed the whole multipart body.

**Plan:** §11 ("upload endpoint for image/video assets"), and the §4.1 `MediaType` enum (`image`, `video`, `document`).

**Fix:**
1. Allowlist the sniffed type and derive the object extension from it:

   | Sniffed type | Extension | MediaType |
   | :--- | :--- | :--- |
   | `image/jpeg` | `.jpg` | image |
   | `image/png` | `.png` | image |
   | `image/gif` | `.gif` | image |
   | `image/webp` | `.webp` | image |
   | `video/mp4` | `.mp4` | video |
   | `video/webm` | `.webm` | video |
   | `application/pdf` | `.pdf` | document |

   Any other type returns 415 `unsupported media type`.
2. Before `c.FormFile`, limit the body with `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(1<<20))`, and map `*http.MaxBytesError` to 413.
3. In Swagger, add `@Failure 413` and `@Failure 415`, and list the allowed types in `@Description`.

**Acceptance:**
- PNG, JPEG, MP4 and PDF uploads return 201.
- HTML, SVG, plain text and unknown binaries return 415.
- A file over 100 MB returns 413 without being fully buffered.

---

### M5: Telegram bot token and JWTs leak into logs

**Where:** `internal/notification/telegram.go:32-40`, logged at `internal/notification/service.go:110` and `:136`. Also `cmd/bootstrap.go:79` (`gin.Default()`) together with `internal/middleware/auth.go:18-22` (`?token=` on `/ws/`).

**Problem:**
- `http.Client.Do` returns `*url.Error`, whose message contains the full request URL `https://api.telegram.org/bot<TOKEN>/sendMessage`. Every network failure writes the bot token into the logs.
- gin's default logger prints the path with its raw query string, so every WebSocket connection logs the user's JWT.

**Fix:**
1. In `SendMessage`, when `Do` fails:
   ```go
   var ue *url.Error
   if errors.As(err, &ue) {
       return fmt.Errorf("telegram: %s: %w", ue.Op, ue.Err)
   }
   ```
2. Replace `gin.Default()` with `gin.New()`, `gin.Recovery()` and `gin.LoggerWithConfig(...)`, using a formatter that redacts the `token` query parameter. At minimum, skip `/ws/chat`. Read `LoggerConfig` and `LogFormatterParams` in gin's source first.

**Acceptance:**
- With an unreachable Telegram host, the log line does not contain the token.
- A WebSocket connection with `?token=` does not print the JWT.

---

### M6: Soft delete plus full unique indexes block reuse of titles, SKUs, slugs and keys

**Where:**
- `internal/product/model.go:11` (`Category.Slug`), `:22` (`Product.Title`), `:25` (`Product.SKU`), `:35` (`Attribute.Key`)
- `internal/cms/model.go:11` (`BlogPost.Slug`)

**Problem:** These are full unique indexes, but deletes are soft (`gorm.Model`). After a product is deleted, creating a new one with the same title or SKU returns 409 forever. Slugs and attribute keys behave the same way.

**Fix:**
- Make them partial unique indexes over non-deleted rows, for example `gorm:"uniqueIndex:idx_products_title,where:deleted_at IS NULL;not null"`. Confirm the `where` index option in GORM's docs first.
- Do **not** change `Cart.UserID`, `InventoryStock.ProductID` or `SiteContent.Key`. They are `ON CONFLICT` targets, and those rows are never soft-deleted.
- If a database already has the old full indexes, `AutoMigrate` will not replace an index that exists under the same name. Drop them once (`db.Migrator().DropIndex`) or use new index names.

**Acceptance:**
- Create "X", delete it, then create "X" again: 201.
- Two active products titled "X": 409.

---

### M7: Checkout can deadlock on stock row locks

**Where:** `internal/cart/store.go:126-144` (`Checkout`).

**Problem:** Stock rows are locked `FOR UPDATE` in the order cart items happen to load. Two concurrent checkouts with overlapping products in different orders can deadlock; PostgreSQL aborts one of them and that customer gets a 500.

**Fix:** Sort `lines` by `ProductID` before creating the order and dispatching stock. If L10 is not done yet, also merge duplicate product lines.

**Acceptance:** Stock dispatch always runs in ascending product ID order.

---

### M8: `PATCH /api/cms/content` wipes fields that were omitted

**Where:** `internal/cms/dto.go:43-48` (`SiteContentItemDto`), `internal/cms/store.go:145-159` (`UpsertContents`).

**Problem:**
- The upsert always updates `value` and `media_id`. Sending `{"key":"logo","description":"x"}` sets `value` to `""` and removes the logo media.
- That is not PATCH behaviour, and there is no explicit way to clear media.

**Fix:**
1. Make `Value` a `*string`.
2. Add `ClearMedia bool`, with binding `excluded_with` on `MediaID` (the same pattern as `UpdateCategoryDto.MakeRoot`).
3. Build the `DoUpdates` column list from the provided fields only, always adding `updated_at`:
   - `value` when it is non-nil;
   - `description` when it is non-nil;
   - `media_id` when `MediaID` is set or `ClearMedia` is true.
4. A new key inserts with `""` when `value` is missing.
5. Update the Swagger `@Description`.

**Acceptance:**
- Updating only `description` leaves `value` and `media_id` unchanged.
- `"clear_media": true` sets `media_id` to null.

---

### M9: `docker-compose.yml` cannot run the app

**Where:** `docker-compose.yml`, `internal/config/env.go:51`.

**Problem:**
- Compose passes `DATABASE_URL`, but the config reads `DSN`, so the DSN is empty.
- `JWT_SECRET` is missing, so the app exits with `log.Fatal("JWT_SECRET is required")`.
- There are no `ADMIN_*` variables (see B2).
- There is no MinIO service, although the plan requires MinIO.
- There are no Zibal, client URL or CORS variables.

**Fix:**
1. Set `DSN` in key/value form as in `dev.example.sh`, with `host=db`.
2. Add `JWT_SECRET`, `ADMIN_NAME`, `ADMIN_MOBILE`, `ADMIN_PASSWORD`, `ZIBAL_MERCHANT`, `ZIBAL_CALLBACK_URL`, `CLIENT_PAYMENT_REDIRECT_URL`, `CORS_ORIGINS`, `TELEGRAM_*`, `MINIO_*`, and every new variable from this document.
3. Take secrets from `${VAR}` interpolation or an `env_file` (`.env` is gitignored) instead of hardcoding them.
4. Add a `minio` service:
   - image `minio/minio`, command `server /data --console-address :9001`;
   - ports 9000 and 9001, and a named volume;
   - `MINIO_ENDPOINT=minio:9000` and `MINIO_PUBLIC_URL=http://localhost:9000`, so browsers can load media URLs;
   - make `app` depend on it.

**Acceptance:** `docker compose up` starts `app`, `db` and `minio`. Migrations and seeding run, and login and upload work.

---

### M10: 500 responses leak internal error details

**Where:** `internal/types/http.go:33-34` (the `HandleError` default case), `internal/auth/http.go:41-42`, `internal/media/handler.go:82, 98, 104`, `internal/chat/handler.go:71`.

**Problem:** 500 responses return `err.Error()` to any caller, exposing SQL errors, table and column names, and connection details.

**Fix:** For 500s, `log.Println` the error and respond with `{"error":"internal server error"}`. Keep the specific messages for 4xx responses.

**Acceptance:** A forced DB error returns the generic message, and the real error appears in the server log.

---

## 5. P3: Low

### L1: `internal/types` is not in the plan structure (decision)

**Where:** `internal/types/http.go`, `internal/types/response.go`.

**Problem:** CLAUDE.md rule 3 requires the plan.md §2 structure, and §2 has no `internal/types`. Every handler and Swagger annotation uses it (`types.ApiResponse`, `types.HandleError` and so on).

**Fix:** Default choice (ask the user if possible): keep the package and add it to plan.md §2:
- `internal/types/http.go`: request helpers and error mapping;
- `internal/types/response.go`: response envelopes.

Moving it into `middleware` would touch every handler and annotation for no functional gain.

---

### L2: Sync `plan.md` with justified model deviations

**Where:** `internal/finance/model.go:11` (`OrderFailed`), `internal/notification/model.go:26-30` (`NotificationRead`).

**Problem:**
- The plan §4.5 Order enum lacks `failed`, but §9 says "marks order as failed".
- `NotificationRead`, the per-user read state for broadcast notifications, is not in plan §4.9.

**Fix:** No code change. In plan.md:
- add `failed` to the §4.5 Order status enum;
- add the `NotificationRead` table (`NotificationID`, `UserID`, unique pair) to §4.9.

---

### L3: `GET /api/inventory` omits products without a stock row

**Where:** `internal/inventory/store.go:29-38` (`List`).

**Problem:** Plan §8 says "current stock levels for all products", but only products that already have an `inventory_stocks` row appear. A new product stays invisible until its first inbound movement.

**Fix:**
1. Paginate over products (non-deleted, including inactive ones).
2. Load the stock rows for that page's product IDs.
3. Return one `InventoryStock` per product; for products without a row, use `Quantity: 0` with `Product` set.
4. Set `total` to the product count.

**Acceptance:** A new product with no inbound movement appears with quantity 0.

---

### L4: Inactive products and banners: public leak and staff blind spot

**Where:** `internal/product/store.go:178-180` (`FindProduct`), `internal/product/handler.go:226-259`, `internal/cms/store.go:114-116` (`ActiveBanners`), `internal/cms/handler.go:39-51`.

**Problem:**
- Public `GET /api/products/:id` returns inactive products, while `GET /api/products` hides them.
- Staff have no endpoint that lists inactive products or inactive banners. Once something is deactivated, a storekeeper or marketer can find it only by remembering its ID.

**Fix:**
1. Add `middleware.OptionalAuth(secret, check)`, using the same checker as M1. It parses a token when one is present and never aborts. Use it on the public product and CMS GET routes, for example by passing `api.Group("", optionalAuth)` as the public group to `product.RegisterRoutes` and `cms.RegisterRoutes`.
2. For public callers, `GET /api/products/:id` returns 404 for inactive products.
3. With `?include_inactive=true`:
   - a storekeeper or admin token makes product list and get include inactive products;
   - a marketer or admin token makes `GET /api/cms/content` include inactive banners.
4. Document the query parameter in Swagger.

**Acceptance:**
- An anonymous request for an inactive product returns 404.
- A storekeeper with `include_inactive=true` sees it.

---

### L5: Unknown media IDs are silently dropped

**Where:** `internal/product/store.go:115-124` and `internal/cms/store.go:57-64` (both `loadMedia`).

**Problem:** `media_ids` entries that do not exist are ignored, so the request succeeds with fewer media than were sent. `attribute_ids` are already validated (`loadAttributes` → `ErrAttributeNotFound` → 400).

**Fix:**
1. Deduplicate the IDs, compare counts, and return `ErrMediaNotFound`.
2. Map it to 400 in product's `fail` helper, and add an equivalent helper in the CMS handler.

**Acceptance:** An unknown media ID in product or blog create/update returns 400, and nothing is written.

---

### L6: WebSocket upgrader accepts any Origin

**Where:** `internal/chat/handler.go:16-20`.

**Problem:** `CheckOrigin` always returns true, and authentication also accepts the `access_token` cookie. The SameSite=Strict cookie currently mitigates cross-site WebSocket hijacking, but the origin check should still match the CORS allowlist.

**Fix:** Build the upgrader in `NewHandler` from the configured `CorsOrigins`. Allow an empty `Origin` (non-browser clients) or one that is in the list.

**Acceptance:** A handshake from a non-allowlisted Origin gets 403.

---

### L7: Pre-invoice message lacks the link and item breakdown

**Where:** `internal/finance/store.go:67-86` (`NotifyOrderPlaced`, `notifyPreInvoice`), `internal/cart/store.go:150`.

**Problem:** Plan §5.1 says to "send pre-invoice link and breakdown". The current message has only the invoice number and amount.

**Fix:**
1. Add env `CLIENT_PREINVOICE_URL` (default `http://localhost:3000/preinvoices`) everywhere listed in rule 7.
2. Load the order with its items and products (`FindOrder`).
3. Build the message:
   - one line per item: `<title> × <qty> — <unit price>`;
   - the total;
   - the link `<CLIENT_PREINVOICE_URL>?inv=<invoice_number>`.
4. Put the same data (an items array and the link) in `ExtraData`.
5. Apply this to both automatic and manual pre-invoices.

**Acceptance:** The chat message, the notification and the Telegram text all show the item lines, the total and the link.

---

### L8: `telegram_chat_id` accepts any chat, including groups

**Where:** `internal/user/dto.go:6`.

**Problem:** Customers can set any value, including negative IDs (groups and channels), and their order and payment notifications are sent there.

**Fix:** Use binding `omitempty,gt=0`; private chat IDs are positive (confirm in the Telegram Bot API docs). Full ownership verification (a bot `/start` deep-link flow) is outside the plan's scope; do not build it unless asked.

**Acceptance:** A negative ID returns 400.

---

### L9: Staff avatar fields are not mutually exclusive

**Where:** `internal/staff/dto.go:3-7`, `internal/staff/handler.go:55-79`, `internal/admin/dto.go:8-9`.

**Problem:** `avatar_media_id` and `default_avatar_id` can both be set. Switching from one to the other leaves the old value in place, so the frontend cannot tell which avatar is current.

**Fix:**
1. Add `excluded_with` between the two fields in both DTOs.
2. In `UpdateMe`, setting one field sets the other to `NULL` in the same update.
3. State this in the Swagger descriptions (plan §12.1).

**Acceptance:** After switching to a preset avatar, `avatar_media_id` is null, and the reverse also holds.

---

### L10: Cart item races, quantity limit, and a vague checkout error

**Where:** `internal/cart/model.go:14-20`, `internal/cart/store.go:56-83` (`AddItem`), `internal/cart/store.go:128-130`.

**Problem:**
- There is no unique `(cart_id, product_id)`, so concurrent adds can create duplicate lines.
- Repeated adds can push the quantity past the DTO maximum of 1000.
- Checkout's "product is not available" error does not say which product.

**Fix:**
1. Add a composite unique index `idx_cart_items_cart_product` on `CartID` + `ProductID`. Keep the single-column indexes. Cart items are hard-deleted, so soft delete is not a concern here.
2. In `AddItem`:
   - lock the existing line `FOR UPDATE`;
   - if `quantity + qty > 1000`, return `ErrQuantityLimit` (400);
   - if the insert hits `gorm.ErrDuplicatedKey`, retry the update path once.
3. Wrap the unavailable error with the product ID: `fmt.Errorf("%w: product %d", ErrProductUnavailable, id)`. Existing `errors.Is` checks keep working.

**Acceptance:**
- Parallel adds of the same product produce one line.
- Going over 1000 returns 400.
- The checkout error names the product.

---

### L11: Duplicate open chat rooms under concurrency

**Where:** `internal/chat/model.go:21-26`, `internal/chat/hub.go:81-94` (`EnsureRoom`).

**Problem:** Two simultaneous first messages, or a bot message arriving together with a user message, can create two open rooms for one user.

**Fix:**
1. Add a partial unique index on `ChatRoom.UserID`: `uniqueIndex:idx_chat_rooms_active_user,where:status <> 'closed' AND deleted_at IS NULL`. Keep the plain index as well.
2. In `EnsureRoom`, re-select the active room when the insert hits `gorm.ErrDuplicatedKey`.
3. Reopening a closed room while another room is active will then return 409 through `HandleError`. Document that on `PATCH /api/support/chat/rooms/:id/status`.

**Acceptance:** Concurrent `EnsureRoom` calls for one user return the same room.

---

### L12: Order history loses soft-deleted products

**Where:** `internal/finance/store.go:88-111` (`FindOrder`, `FindUserOrder`, `ListOrders`).

**Problem:** `Preload("Items.Product")` skips soft-deleted products, so past orders show `product: null` after a product is deleted.

**Fix:** Preload `Items.Product` unscoped in all three functions; `ListOrders` currently preloads only `Items`, so add the product preload there too. Check the GORM generics `Preload` builder API in the docs.

**Acceptance:** After deleting a product that was ordered, the order still shows the product's title and SKU.

---

### L13: Payment amount truncation and stale mobile

**Where:** `internal/payment/handler.go:74-76`.

**Problem:**
- `int64(order.TotalAmount)` truncates instead of rounding.
- The mobile sent to Zibal comes from the JWT, which is stale after `PATCH /api/user/profile` changes the mobile.

**Fix:**
1. Use `int64(math.Round(order.TotalAmount))`.
2. Inject `user.Store` into the payment handler and load the current mobile from it.

---

### L14: Telegram broadcast is not throttled

**Where:** `internal/notification/service.go:117-141` (`Dispatch`).

**Problem:** A broadcast sends to every user as fast as possible. Telegram limits bots to about 30 messages per second overall (confirm in the Bot API FAQ), so large broadcasts hit HTTP 429 and those messages are lost.

**Fix:** Standard library only:
1. Send through a `time.Ticker` at no more than 25 messages per second.
2. On HTTP 429, read `parameters.retry_after` from the response, sleep for that long, and retry once.

---

### L15: Admin notification can target a nonexistent user

**Where:** `internal/notification/handler.go:43-55`, `internal/notification/service.go:117-120`.

**Problem:** `user_id` is not validated, and there is no foreign key, so a notification can target a user who does not exist.

**Fix:** When `user_id` is set, check that the user exists and return 404 `user not found` if not. Add `@Failure 404` to Swagger.

---

### L16: No brute-force protection on login

**Where:** `internal/auth/http.go:62-70`.

**Problem:** Password attempts are unlimited per mobile and per IP.

**Fix:** Add an in-memory limiter using only the standard library (`sync.Mutex`, a map and `time`):
- key it by client IP plus mobile;
- allow at most 5 failed attempts per 15 minutes, then return 429;
- reset the count on a successful login;
- clean up old entries periodically.

Put it inside the existing auth module or in `middleware/auth.go`; add no new file unless plan.md §2 is updated. Add `@Failure 429` to Swagger.

---

## 6. Docs to fetch before coding (rule 4)

Fetch with `curl`, read, and only then write code.

| Topic | Source |
| :--- | :--- |
| GORM partial indexes (`where` option) | https://raw.githubusercontent.com/go-gorm/gorm.io/master/pages/docs/indexes.md |
| GORM row locking (`FOR UPDATE`) | https://raw.githubusercontent.com/go-gorm/gorm.io/master/pages/docs/advanced_query.md |
| GORM preload (including unscoped) | https://raw.githubusercontent.com/go-gorm/gorm.io/master/pages/docs/preload.md |
| GORM transactions | https://raw.githubusercontent.com/go-gorm/gorm.io/master/pages/docs/transactions.md |
| GORM soft delete | https://raw.githubusercontent.com/go-gorm/gorm.io/master/pages/docs/delete.md |
| gin logger configuration | https://raw.githubusercontent.com/gin-gonic/gin/master/logger.go |
| gorilla/websocket `Upgrader.CheckOrigin` | https://raw.githubusercontent.com/gorilla/websocket/main/server.go |
| minio-go client usage | https://raw.githubusercontent.com/minio/minio-go/master/README.md |
| Zibal IPG: request, verify, inquiry, callback, result and status codes, reversal of unverified payments | https://help.zibal.ir/IPG/API/ (and the examples at https://github.com/zibalco) |
| Telegram Bot API: `sendMessage`, `chat_id` semantics | https://core.telegram.org/bots/api |
| Telegram rate limits | https://core.telegram.org/bots/faq |
| `http.MaxBytesReader`, `http.DetectContentType` | https://pkg.go.dev/net/http |

---

## 7. Definition of done

- [ ] Every checklist item in section 1 is ticked, and every acceptance criterion is met.
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `make swag` has been run. Swagger UI at `http://localhost:4000/swagger/index.html` lists every route, including the new ones, and no longer lists routes that don't exist.
- [ ] The comment check from B4 prints nothing.
- [ ] `plan.md` is updated for:
  - new routes: `POST /api/user/orders/:id/cancel`, `PATCH /api/admin/orders/:id/status`;
  - new env vars: `ORDER_EXPIRY_MINUTES`, `CLIENT_PREINVOICE_URL`;
  - the `include_inactive` query parameter (L4);
  - the `failed` order status and the `NotificationRead` model (L2);
  - `internal/types` (L1).
- [ ] The new env vars appear in `internal/config/env.go`, `dev.example.sh`, `dev.example.bat`, `dev.example.ps1` and `docker-compose.yml`.
- [ ] `todo.md` reflects reality. Its bootstrap item was ticked while migrations and the seeder were disabled, and the E2E item stays open until the scenarios below pass.

### Manual end-to-end scenarios

These need PostgreSQL, MinIO and the Zibal sandbox (merchant `zibal`), most easily via the fixed `docker compose` from M9.

1. **Fresh boot:** tables are created, the admin is seeded and can log in, and sample data exists outside production (B1, B2).
2. **Media upload:** as storekeeper, upload a PNG (201) and an HTML file (415), then attach the PNG to a product (B3, M4, L5).
3. **Happy-path payment:** customer adds to cart, checks out (stock decreases, pre-invoice issued, bot messages sent), pays in the sandbox, and is redirected to `?inv=inv-<id>`. The order is `paid` and the pre-invoice is `paid` (H1).
4. **Forged callback:** call `GET /api/payment/callback?trackId=<id>&success=0` before paying, then pay. The order still ends up `paid` (H1).
5. **Double payment:** create two trackIds for one order and pay both. The order is paid once, there is one paid pre-invoice, and the second payment is not verified (H4).
6. **Failed payment, then cancel or expiry:** stock stays reserved after the failure; a customer cancel or the expiry job restores it with an inbound log (H5).
7. **Order lifecycle:** admin moves the order `paid → processing → delivered`, then cancels another paid order and its stock is restored (H6).
8. **Disabled account:** disable a logged-in staff member; their next request gets 401 (M1).
9. **Chat room lifecycle:** support closes a room, the customer writes on the same socket, a new room opens, and support is alerted. Bot-only rooms are not listed (M2, M3).
10. **Name reuse:** delete a product and create one with the same title and SKU: 201 (M6).
