package main

// All tests for the Milk Quality API.
// No MySQL required — mockStore implements dbStore in memory.
//
// UNIT TESTS — pure logic, no HTTP, no DB
//   TestMilkIsValid                          isValid() returns true for a complete valid struct
//                                            isValid() returns false for empty CowID
//                                            isValid() returns false for zero Fat
//                                            isValid() returns false for negative Protein
//                                            isValid() returns false for zero PH
//                                            isValid() returns false for zero SCC
//
// HANDLER TESTS — httptest.NewRecorder, no middleware, no network
//   TestAllMilk_ReturnsSeededData            allMilk returns all records from store
//   TestAllMilk_EmptyTable_ReturnsEmptySlice allMilk returns 200 on empty store
//   TestAllMilk_DBError_Returns500           allMilk returns 500 on store error
//   TestAllMilk_Timeout_Returns504           allMilk returns 504 on context.DeadlineExceeded
//
//   TestMilkById_Found_Returns200            milkById returns 200 and correct record
//   TestMilkById_NotFound_Returns404         milkById returns 404 on ErrNotFound
//   TestMilkById_DBError_Returns500          milkById returns 500 on store error
//   TestMilkById_Timeout_Returns504          milkById returns 504 on context.DeadlineExceeded
//
//   TestSendMilk_ValidPayload_Returns201     sendMilk returns 201 and passes correct data to store
//   TestSendMilk_MalformedJSON_Returns400    sendMilk returns 400 on invalid JSON
//   TestSendMilk_InvalidFields_Returns400    sendMilk returns 400 when isValid() fails
//   TestSendMilk_DBError_Returns500          sendMilk returns 500 on store error
//   TestSendMilk_Timeout_Returns504          sendMilk returns 504 on context.DeadlineExceeded
//
// MIDDLEWARE TESTS — httptest.NewServer, full Logger+Authentication+TimeKeeper chain
//   TestAuth_POST_MissingAPIKey_Returns401   POST with no Api-key header returns 401
//   TestAuth_POST_WrongAPIKey_Returns401     POST with wrong Api-key header returns 401
//   TestAuth_POST_CorrectAPIKey_Passes       POST with correct Api-key header passes through
//   TestAuth_GET_NoKeyRequired_Passes        GET requests bypass auth and return 200

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock store — per-method function fields so each test controls exactly
// what each method returns, independently of the others.
// ---------------------------------------------------------------------------

type mockStore struct {
	MilkByIdFunc func(ctx context.Context, id string) (*milk, error)
	AllMilkFunc  func(ctx context.Context) ([]milk, error)
	SendMilkFunc func(ctx context.Context, m milk) (int64, error)
}

func (m *mockStore) MilkById(ctx context.Context, id string) (*milk, error) {
	if m.MilkByIdFunc == nil {
		return nil, nil
	}
	return m.MilkByIdFunc(ctx, id)
}

func (m *mockStore) AllMilk(ctx context.Context) ([]milk, error) {
	if m.AllMilkFunc == nil {
		return nil, nil
	}
	return m.AllMilkFunc(ctx)
}

func (m *mockStore) SendMilk(ctx context.Context, mk milk) (int64, error) {
	if m.SendMilkFunc == nil {
		return 0, nil
	}
	return m.SendMilkFunc(ctx, mk)
}

// ---------------------------------------------------------------------------
// isValid() unit tests — pure logic, no HTTP, no DB
// ---------------------------------------------------------------------------

func TestMilkIsValid(t *testing.T) {
	tests := []struct {
		name  string
		milk  milk
		valid bool
	}{
		{
			name:  "valid milk sample",
			milk:  milk{CowID: "cow-1", Fat: 3.5, Protein: 3.2, PH: 6.8, SCC: 150000},
			valid: true,
		},
		{
			name:  "empty cow ID",
			milk:  milk{CowID: "", Fat: 3.5, Protein: 3.2, PH: 6.8, SCC: 150000},
			valid: false,
		},
		{
			name:  "zero fat",
			milk:  milk{CowID: "cow-1", Fat: 0, Protein: 3.2, PH: 6.8, SCC: 150000},
			valid: false,
		},
		{
			name:  "negative protein",
			milk:  milk{CowID: "cow-1", Fat: 3.5, Protein: -1.0, PH: 6.8, SCC: 150000},
			valid: false,
		},
		{
			name:  "zero pH",
			milk:  milk{CowID: "cow-1", Fat: 3.5, Protein: 3.2, PH: 0, SCC: 150000},
			valid: false,
		},
		{
			name:  "zero SCC",
			milk:  milk{CowID: "cow-1", Fat: 3.5, Protein: 3.2, PH: 6.8, SCC: 0},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.milk.isValid()
			if got != tt.valid {
				t.Errorf("isValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// allMilk handler tests
// ---------------------------------------------------------------------------

func TestAllMilk_ReturnsSeededData(t *testing.T) {
	seeded := []milk{
		{CowID: "cow-1", Fat: 3.5, Protein: 3.2, PH: 6.8, SCC: 150000},
		{CowID: "cow-2", Fat: 4.0, Protein: 3.5, PH: 6.7, SCC: 200000},
	}
	store := &mockStore{
		AllMilkFunc: func(ctx context.Context) ([]milk, error) {
			return seeded, nil
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk", nil)
	rec := httptest.NewRecorder()
	app.allMilk(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []milk
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d records, want 2", len(got))
	}
}

func TestAllMilk_EmptyTable_ReturnsEmptySlice(t *testing.T) {
	store := &mockStore{
		AllMilkFunc: func(ctx context.Context) ([]milk, error) {
			return []milk{}, nil
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk", nil)
	rec := httptest.NewRecorder()
	app.allMilk(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAllMilk_DBError_Returns500(t *testing.T) {
	store := &mockStore{
		AllMilkFunc: func(ctx context.Context) ([]milk, error) {
			return nil, errors.New("connection refused")
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk", nil)
	rec := httptest.NewRecorder()
	app.allMilk(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestAllMilk_Timeout_Returns504(t *testing.T) {
	store := &mockStore{
		AllMilkFunc: func(ctx context.Context) ([]milk, error) {
			return nil, context.DeadlineExceeded
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk", nil)
	rec := httptest.NewRecorder()
	app.allMilk(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// milkById handler tests
// ---------------------------------------------------------------------------

func TestMilkById_Found_Returns200(t *testing.T) {
	expected := &milk{CowID: "cow-1", Fat: 3.5, Protein: 3.2, PH: 6.8, SCC: 150000}
	store := &mockStore{
		MilkByIdFunc: func(ctx context.Context, id string) (*milk, error) {
			return expected, nil
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk/cow-1", nil)
	req.SetPathValue("id", "cow-1")
	rec := httptest.NewRecorder()
	app.milkById(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var got milk
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if got.CowID != expected.CowID {
		t.Errorf("CowID = %s, want %s", got.CowID, expected.CowID)
	}
}

func TestMilkById_NotFound_Returns404(t *testing.T) {
	store := &mockStore{
		MilkByIdFunc: func(ctx context.Context, id string) (*milk, error) {
			return nil, ErrNotFound
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk/ghost", nil)
	req.SetPathValue("id", "ghost")
	rec := httptest.NewRecorder()
	app.milkById(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestMilkById_DBError_Returns500(t *testing.T) {
	store := &mockStore{
		MilkByIdFunc: func(ctx context.Context, id string) (*milk, error) {
			return nil, errors.New("disk read error")
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk/cow-1", nil)
	req.SetPathValue("id", "cow-1")
	rec := httptest.NewRecorder()
	app.milkById(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestMilkById_Timeout_Returns504(t *testing.T) {
	store := &mockStore{
		MilkByIdFunc: func(ctx context.Context, id string) (*milk, error) {
			return nil, context.DeadlineExceeded
		},
	}
	app := &App{store: store}

	req := httptest.NewRequest(http.MethodGet, "/milk/cow-1", nil)
	req.SetPathValue("id", "cow-1")
	rec := httptest.NewRecorder()
	app.milkById(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// sendMilk handler tests
// ---------------------------------------------------------------------------

func TestSendMilk_ValidPayload_Returns201(t *testing.T) {
	var received milk
	store := &mockStore{
		SendMilkFunc: func(ctx context.Context, m milk) (int64, error) {
			received = m // capture what was passed to the store
			return 1, nil
		},
	}
	app := &App{store: store}

	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	req := httptest.NewRequest(http.MethodPost, "/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.sendMilk(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if received.CowID != "cow-1" {
		t.Errorf("store received CowID = %s, want cow-1", received.CowID)
	}
}

func TestSendMilk_MalformedJSON_Returns400(t *testing.T) {
	app := &App{store: &mockStore{}}

	body := `{"cowid":"cow-1", bad json`
	req := httptest.NewRequest(http.MethodPost, "/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.sendMilk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestSendMilk_InvalidFields_Returns400(t *testing.T) {
	app := &App{store: &mockStore{}}

	// missing fat, protein, pH, scc — isValid() should return false
	body := `{"cowid":"cow-1"}`
	req := httptest.NewRequest(http.MethodPost, "/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.sendMilk(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestSendMilk_DBError_Returns500(t *testing.T) {
	store := &mockStore{
		SendMilkFunc: func(ctx context.Context, m milk) (int64, error) {
			return 0, errors.New("insert failed")
		},
	}
	app := &App{store: store}

	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	req := httptest.NewRequest(http.MethodPost, "/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.sendMilk(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestSendMilk_Timeout_Returns504(t *testing.T) {
	store := &mockStore{
		SendMilkFunc: func(ctx context.Context, m milk) (int64, error) {
			return 0, context.DeadlineExceeded
		},
	}
	app := &App{store: store}

	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	req := httptest.NewRequest(http.MethodPost, "/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.sendMilk(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// newTestServer — spins up the full middleware stack against a mock store.
// Use this when you need Logger + Authentication + TimeKeeper in the chain.
// Use httptest.NewRecorder directly when testing a handler in isolation.
// ---------------------------------------------------------------------------

func newTestServer(t *testing.T, store dbStore) *httptest.Server {
	t.Helper()
	app := &App{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /milk", app.allMilk)
	mux.HandleFunc("GET /milk/{id}", app.milkById)
	mux.HandleFunc("POST /milk", app.sendMilk)
	handler := Logger(Authentication(TimeKeeper(mux)))
	return httptest.NewServer(handler)
}

// ---------------------------------------------------------------------------
// Authentication middleware tests
// ---------------------------------------------------------------------------

func TestAuth_POST_MissingAPIKey_Returns401(t *testing.T) {
	// API_KEY must be set to a non-empty value so the middleware has something
	// to compare against — otherwise missing header == empty string == empty env == passes
	os.Setenv("API_KEY", "correct-key")
	defer os.Unsetenv("API_KEY")

	ts := newTestServer(t, &mockStore{})
	defer ts.Close()

	// Send POST with no Api-key header at all
	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	resp, err := http.Post(ts.URL+"/milk", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_POST_WrongAPIKey_Returns401(t *testing.T) {
	os.Setenv("API_KEY", "correct-key")
	defer os.Unsetenv("API_KEY")

	ts := newTestServer(t, &mockStore{})
	defer ts.Close()

	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-key", "wrong-key") // matches middleware: r.Header.Get("Api-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_POST_CorrectAPIKey_Passes(t *testing.T) {
	os.Setenv("API_KEY", "correct-key")
	defer os.Unsetenv("API_KEY")

	store := &mockStore{
		SendMilkFunc: func(ctx context.Context, m milk) (int64, error) {
			return 1, nil
		},
	}
	ts := newTestServer(t, store)
	defer ts.Close()

	body := `{"cowid":"cow-1","fat":3.5,"protein":3.2,"pH":6.8,"scc":150000}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/milk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-key", "correct-key") // matches middleware: r.Header.Get("Api-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}

func TestAuth_GET_NoKeyRequired_Passes(t *testing.T) {
	// GET requests should not require an API key
	store := &mockStore{
		AllMilkFunc: func(ctx context.Context) ([]milk, error) {
			return []milk{}, nil
		},
	}
	ts := newTestServer(t, store)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/milk")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
