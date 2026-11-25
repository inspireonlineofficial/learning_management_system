// Command bookshopload load-tests the bookshop API on a LOCAL or staging
// stack. Never point it at production: it creates users, books and orders.
//
//	go run ./cmd/bookshopload -api http://127.0.0.1:8080 -dsn "$DATABASE_DSN"
//
// Scenarios (each prints throughput, latency percentiles and status codes,
// then checks the data for consistency):
//
//	browse     anonymous catalogue list + detail (each request from its own
//	           simulated client IP, like many visitors)
//	student    signed-in mix: library, orders, cart, ownership, book detail
//	stock      N students race for K printed copies
//	approvals  M purchase requests approved concurrently
//	refunds    M refund requests filed and approved concurrently
//	cart       M students fill a cart and check out at the same time
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq"
)

var (
	api      = flag.String("api", "http://127.0.0.1:8080", "API base URL")
	dsn      = flag.String("dsn", os.Getenv("DATABASE_DSN"), "Postgres DSN (setup and checks)")
	duration = flag.Duration("duration", 30*time.Second, "duration of the browse/student scenarios")
	vus      = flag.Int("vus", 50, "concurrent virtual users for browse/student")
	students = flag.Int("students", 120, "students for the write scenarios")
	only     = flag.String("only", "", "comma-separated scenarios to run (default all)")
	password = "Password123!"
	client   = &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 512, MaxIdleConns: 512}}
	db       *sql.DB
	tag      = fmt.Sprintf("%06d", rand.Intn(1000000))
	failures []string
)

type stats struct {
	mu        sync.Mutex
	latencies []time.Duration
	codes     map[int]int
}

func newStats() *stats { return &stats{codes: map[int]int{}} }

func (s *stats) add(d time.Duration, code int) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.codes[code]++
	s.mu.Unlock()
}

func (s *stats) report(name string, elapsed time.Duration) {
	sort.Slice(s.latencies, func(i, j int) bool { return s.latencies[i] < s.latencies[j] })
	n := len(s.latencies)
	pct := func(p float64) time.Duration {
		if n == 0 {
			return 0
		}
		return s.latencies[int(float64(n-1)*p)]
	}
	codes := []string{}
	for c, k := range s.codes {
		codes = append(codes, fmt.Sprintf("%d×%d", c, k))
	}
	sort.Strings(codes)
	fmt.Printf("%-10s %6d req  %7.1f req/s  p50 %6s  p95 %6s  p99 %6s  max %6s  [%s]\n", name, n,
		float64(n)/elapsed.Seconds(), round(pct(.5)), round(pct(.95)), round(pct(.99)), round(pct(1)), strings.Join(codes, " "))
	for c, k := range s.codes {
		if c >= 500 || c == 0 {
			fail("%s: %d responses with status %d", name, k, c)
		}
	}
}

func round(d time.Duration) string { return d.Round(100 * time.Microsecond).String() }

func fail(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	failures = append(failures, msg)
	fmt.Println("  FAIL:", msg)
}

func ok(format string, args ...interface{}) { fmt.Println("  ok:", fmt.Sprintf(format, args...)) }

func randomIP() string {
	return fmt.Sprintf("10.%d.%d.%d", rand.Intn(250)+1, rand.Intn(250)+1, rand.Intn(250)+1)
}

// call does one request; st may be nil. It returns status and body.
func call(st *stats, method, path, token string, body interface{}, headers ...string) (int, []byte) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, *api+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.Header.Set("X-Real-IP", randomIP())
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		if st != nil {
			st.add(time.Since(start), 0)
		}
		return 0, nil
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if st != nil {
		st.add(time.Since(start), resp.StatusCode)
	}
	return resp.StatusCode, data
}

func login(email string) string {
	for attempt := 0; attempt < 3; attempt++ {
		code, body := call(nil, "POST", "/v1/auth/login", "", map[string]string{"email": email, "password": password})
		if code == 200 {
			var t struct {
				AccessToken string `json:"access_token"`
			}
			_ = json.Unmarshal(body, &t)
			return t.AccessToken
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println("login failed for", email)
	os.Exit(1)
	return ""
}

// makeStudents creates n synthetic students (copying the seed student's
// password hash) and signs them in.
func makeStudents(n int, prefix string) []string {
	emails := make([]string, n)
	for i := range emails {
		emails[i] = fmt.Sprintf("load.%s.%s.%d@example.com", prefix, tag, i)
		_, err := db.Exec(`INSERT INTO users (id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at)
			SELECT gen_random_uuid(), $1, $2, $3, password_hash, 'student', status, true, now(), now()
			FROM users WHERE email = 'student@example.com' ON CONFLICT DO NOTHING`,
			fmt.Sprintf("Load Student %d", i), emails[i], fmt.Sprintf("load_%s_%s_%d", prefix, tag, i))
		if err != nil {
			fmt.Println("create user:", err)
			os.Exit(1)
		}
	}
	tokens := make([]string, n)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := range emails {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			tokens[i] = login(emails[i])
			<-sem
		}(i)
	}
	wg.Wait()
	return tokens
}

// makeAdmins creates n synthetic admins (a review team), so admin actions
// stay under the per-user rate limit like they would with real staff.
func makeAdmins(n int) []string {
	tokens := make([]string, n)
	for i := range tokens {
		email := fmt.Sprintf("load.admin.%s.%d@example.com", tag, i)
		if _, err := db.Exec(`INSERT INTO users (id, full_name, email, username, password_hash, role, status, profile_complete, created_at, updated_at)
			SELECT gen_random_uuid(), $1, $2, $3, password_hash, 'admin', status, true, now(), now()
			FROM users WHERE email = 'admin@example.com' ON CONFLICT DO NOTHING`,
			fmt.Sprintf("Load Admin %d", i), email, fmt.Sprintf("load_admin_%s_%d", tag, i)); err != nil {
			fmt.Println("create admin:", err)
			os.Exit(1)
		}
		tokens[i] = login(email)
	}
	return tokens
}

var minimalPDF = []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj 2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj 3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n")

func createBook(admin string, fields map[string]interface{}) string {
	body := map[string]interface{}{"title": "Load " + tag, "author": "Load Author", "subject": "Physics", "format": "digital", "price": 0, "currency": "BDT", "physical_stock": 0}
	for k, v := range fields {
		body[k] = v
	}
	code, data := call(nil, "POST", "/v1/admin/bookshop/books", admin, body)
	if code != 201 && code != 200 {
		fmt.Println("create book failed:", code, string(data))
		os.Exit(1)
	}
	var b struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(data, &b)
	if body["format"] != "physical" {
		var buf bytes.Buffer
		buf.WriteString("--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"b.pdf\"\r\nContent-Type: application/pdf\r\n\r\n")
		buf.Write(minimalPDF)
		buf.WriteString("\r\n--x--\r\n")
		req, _ := http.NewRequest("POST", *api+"/v1/admin/bookshop/books/"+b.ID+"/file", &buf)
		req.Header.Set("Authorization", "Bearer "+admin)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}
	return b.ID
}

// runFor runs fn on vus goroutines until the duration passes.
func runFor(name string, fn func(st *stats, vu int)) {
	st := newStats()
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < *vus; i++ {
		wg.Add(1)
		go func(vu int) {
			defer wg.Done()
			for ctx.Err() == nil {
				fn(st, vu)
			}
		}(i)
	}
	wg.Wait()
	st.report(name, time.Since(start))
}

// burst runs fn(i) for i in [0,n) on conc goroutines and reports.
func burst(name string, n, conc int, fn func(st *stats, i int)) time.Duration {
	st := newStats()
	jobs := make(chan int)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				fn(st, i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	elapsed := time.Since(start)
	st.report(name, elapsed)
	return elapsed
}

func want(scenario string) bool {
	if *only == "" {
		return true
	}
	for _, s := range strings.Split(*only, ",") {
		if strings.TrimSpace(s) == scenario {
			return true
		}
	}
	return false
}

func queryInt(q string, args ...interface{}) int {
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		fail("query %q: %v", q, err)
	}
	return n
}

var address = map[string]string{"name": "Load Tester", "phone": "01700000000", "line1": "House 1, Road 2", "city": "Dhaka"}

func main() {
	flag.Parse()
	var err error
	if db, err = sql.Open("postgres", *dsn); err != nil || db.Ping() != nil {
		fmt.Println("database connection failed; set -dsn")
		os.Exit(1)
	}
	if strings.Contains(*api, "inspire") && !strings.Contains(*api, "staging") {
		fmt.Println("refusing to run against what looks like production:", *api)
		os.Exit(1)
	}
	admins := makeAdmins(10)
	admin := admins[0]
	adminFor := func(i int) string { return admins[i%len(admins)] }
	fmt.Printf("bookshop load test against %s (tag %s, %d VUs, %s, %d admins)\n\n", *api, tag, *vus, *duration, len(admins))

	// Catalogue of 60 books for browsing.
	var bookIDs []string
	for i := 0; i < 60; i++ {
		bookIDs = append(bookIDs, createBook(adminFor(i), map[string]interface{}{"title": fmt.Sprintf("Load %s Book %02d", tag, i), "price": float64(rand.Intn(5)) * 100}))
	}

	if want("browse") {
		sorts := []string{"newest", "price_asc", "price_desc", "title"}
		runFor("browse", func(st *stats, vu int) {
			if rand.Intn(3) == 0 {
				call(st, "GET", "/v1/bookshop/books/"+bookIDs[rand.Intn(len(bookIDs))], "", nil)
				return
			}
			q := fmt.Sprintf("/v1/bookshop/books?limit=24&page=%d&sort=%s", rand.Intn(3)+1, sorts[rand.Intn(4)])
			if rand.Intn(2) == 0 {
				q += "&search=Load"
			}
			call(st, "GET", q, "", nil)
		})
	}

	if want("student") {
		tokens := makeStudents(*vus, "mix")
		runFor("student", func(st *stats, vu int) {
			t := tokens[vu]
			switch rand.Intn(5) {
			case 0:
				call(st, "GET", "/v1/student/bookshop/library", t, nil)
			case 1:
				call(st, "GET", "/v1/student/bookshop/orders", t, nil)
			case 2:
				call(st, "GET", "/v1/bookshop/cart", t, nil)
			case 3:
				call(st, "GET", "/v1/student/bookshop/books/"+bookIDs[rand.Intn(len(bookIDs))]+"/status", t, nil)
			default:
				call(st, "GET", "/v1/bookshop/books/"+bookIDs[rand.Intn(len(bookIDs))], t, nil)
			}
			// ~4 requests/s per user keeps each account under the 300/min limit.
			time.Sleep(250 * time.Millisecond)
		})
	}

	var writers []string
	if want("stock") || want("approvals") || want("refunds") || want("cart") {
		fmt.Printf("\ncreating %d students for write scenarios...\n", *students)
		writers = makeStudents(*students, "w")
	}

	if want("stock") {
		copies := *students / 4
		book := createBook(admin, map[string]interface{}{"title": "Load Stock " + tag, "format": "physical", "physical_stock": copies})
		var placed, outOfStock atomic.Int64
		burst("stock", len(writers), 40, func(st *stats, i int) {
			code, _ := call(st, "POST", "/v1/bookshop/orders", writers[i], map[string]interface{}{"book_id": book, "format": "physical", "shipping_address": address})
			switch code {
			case 201:
				placed.Add(1)
			case 400:
				outOfStock.Add(1)
			}
		})
		stock := queryInt(`SELECT physical_stock FROM books WHERE id = $1`, book)
		orders := queryInt(`SELECT COUNT(*) FROM orders WHERE book_id = $1`, book)
		if int(placed.Load()) == copies && orders == copies && stock == 0 && int(outOfStock.Load()) == len(writers)-copies {
			ok("%d students raced for %d copies: exactly %d orders, stock 0, %d told out of stock", len(writers), copies, orders, outOfStock.Load())
		} else {
			fail("stock race: placed=%d orders=%d stock=%d out_of_stock=%d (copies %d)", placed.Load(), orders, stock, outOfStock.Load(), copies)
		}
	}

	paidBook := ""
	if want("approvals") || want("refunds") {
		paidBook = createBook(admin, map[string]interface{}{"title": "Load Paid " + tag, "price": 250})
		requestIDs := make([]string, len(writers))
		burst("requests", len(writers), 40, func(st *stats, i int) {
			code, body := call(st, "POST", "/v1/purchase-requests", writers[i], map[string]interface{}{"item_type": "book", "item_id": paidBook, "file_name": "LOAD-" + tag})
			if code == 201 || code == 200 {
				var r struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(body, &r)
				requestIDs[i] = r.ID
			}
		})
		burst("approvals", len(writers), 40, func(st *stats, i int) {
			call(st, "POST", "/v1/admin/purchase-requests/"+requestIDs[i]+"/approve", adminFor(i), map[string]string{})
		})
		orders := queryInt(`SELECT COUNT(*) FROM orders WHERE book_id = $1`, paidBook)
		approved := queryInt(`SELECT COUNT(*) FROM purchase_requests WHERE item_id = $1 AND status = 'approved'`, paidBook)
		if orders == len(writers) && approved == len(writers) {
			ok("%d purchase requests approved concurrently: %d orders, no duplicates", approved, orders)
		} else {
			fail("approvals: orders=%d approved=%d (want %d)", orders, approved, len(writers))
		}
	}

	if want("refunds") && paidBook != "" {
		orderIDs := make([]string, len(writers))
		rows, err := db.Query(`SELECT o.id, u.email FROM orders o JOIN users u ON u.id = o.student_id WHERE o.book_id = $1`, paidBook)
		byEmail := map[string]string{}
		if err == nil {
			for rows.Next() {
				var id, email string
				_ = rows.Scan(&id, &email)
				byEmail[email] = id
			}
			rows.Close()
		}
		for i := range writers {
			orderIDs[i] = byEmail[fmt.Sprintf("load.w.%s.%d@example.com", tag, i)]
		}
		reqIDs := make([]string, len(writers))
		burst("refund-req", len(writers), 40, func(st *stats, i int) {
			code, body := call(st, "POST", "/v1/student/bookshop/orders/"+orderIDs[i]+"/refund-request", writers[i], map[string]string{"reason": "load test"})
			if code == 201 {
				var r struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal(body, &r)
				reqIDs[i] = r.ID
			}
		})
		// Each request approved twice at once: exactly one refund per order.
		burst("refund-ok", len(writers)*2, 40, func(st *stats, i int) {
			call(st, "POST", "/v1/admin/bookshop/refund-requests/"+reqIDs[i/2]+"/approve", adminFor(i),
				map[string]string{"payout_method": "bkash", "payout_reference": fmt.Sprintf("LT%s%04d", tag, i/2)})
		})
		refunded := queryInt(`SELECT COUNT(*) FROM orders WHERE book_id = $1 AND status = 'refunded'`, paidBook)
		payouts := queryInt(`SELECT COUNT(*) FROM order_refunds r JOIN orders o ON o.id = r.order_id WHERE o.book_id = $1`, paidBook)
		if refunded == len(writers) && payouts == len(writers) {
			ok("%d refund requests approved (each twice, concurrently): %d refunds, one payout record each", len(writers), payouts)
		} else {
			fail("refunds: refunded=%d payouts=%d (want %d)", refunded, payouts, len(writers))
		}
	}

	if want("cart") {
		free := createBook(adminFor(1), map[string]interface{}{"title": "Load Cart Free " + tag})
		paid := createBook(adminFor(2), map[string]interface{}{"title": "Load Cart Paid " + tag, "price": 150})
		printed := createBook(adminFor(3), map[string]interface{}{"title": "Load Cart Print " + tag, "format": "physical", "price": 0, "physical_stock": len(writers)})
		burst("cart-add", len(writers)*3, 40, func(st *stats, i int) {
			items := []map[string]string{{"book_id": free, "format": "digital"}, {"book_id": paid, "format": "digital"}, {"book_id": printed, "format": "physical"}}
			call(st, "POST", "/v1/bookshop/cart/items", writers[i/3], items[i%3])
		})
		var done atomic.Int64
		burst("checkout", len(writers), 40, func(st *stats, i int) {
			body := map[string]interface{}{"shipping_address": address, "payment_reference": "LOADCART"}
			if code, _ := call(st, "POST", "/v1/bookshop/cart/checkout", writers[i], body); code == 201 {
				done.Add(1)
			}
		})
		freeOrders := queryInt(`SELECT COUNT(*) FROM orders WHERE book_id IN ($1, $2)`, free, printed)
		requests := queryInt(`SELECT COUNT(*) FROM purchase_requests WHERE item_id = $1`, paid)
		stock := queryInt(`SELECT physical_stock FROM books WHERE id = $1`, printed)
		if int(done.Load()) == len(writers) && freeOrders == 2*len(writers) && requests == len(writers) && stock == 0 {
			ok("%d carts checked out concurrently: %d free orders, %d purchase requests, stock 0", done.Load(), freeOrders, requests)
		} else {
			fail("cart: checkouts=%d free_orders=%d requests=%d stock=%d", done.Load(), freeOrders, requests, stock)
		}
	}

	fmt.Println()
	if len(failures) > 0 {
		fmt.Printf("%d problem(s) found\n", len(failures))
		os.Exit(1)
	}
	fmt.Println("all consistency checks passed")
}
