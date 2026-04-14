package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	service "taskflow-backend/internal/application"
	postgres "taskflow-backend/internal/storage/postgres"
	httpapi "taskflow-backend/internal/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthAndProjectFlow(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	// register
	reg := map[string]any{"name": "A", "email": "a@example.com", "password": "password123"}
	res := doJSON(t, ts.URL+"/auth/register", reg, "")
	if res.StatusCode != 201 {
		t.Fatalf("register status=%d body=%s", res.StatusCode, res.Body)
	}
	var ar struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	_ = json.Unmarshal([]byte(res.Body), &ar)
	if ar.Token == "" || ar.User.ID == "" {
		t.Fatalf("missing token/user in register response: %s", res.Body)
	}

	// create project
	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "P1"}, ar.Token)
	if pr.StatusCode != 201 {
		t.Fatalf("create project status=%d body=%s", pr.StatusCode, pr.Body)
	}
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)
	if p.ID == "" {
		t.Fatalf("missing project id: %s", pr.Body)
	}

	// list projects (pagination bonus surface)
	lp := doReq(t, "GET", ts.URL+"/projects?page=1&limit=10", nil, ar.Token)
	if lp.StatusCode != 200 {
		t.Fatalf("list projects status=%d body=%s", lp.StatusCode, lp.Body)
	}
}

func TestAuthorizationOwnerOnlyProjectPatch(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	u1 := registerAndToken(t, ts.URL, "owner@example.com")
	u2 := registerAndToken(t, ts.URL, "other@example.com")

	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "P1"}, u1)
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)

	patch := doReq(t, "PATCH", ts.URL+"/projects/"+p.ID, map[string]any{"name": "X"}, u2)
	if patch.StatusCode != 403 {
		t.Fatalf("expected 403, got %d body=%s", patch.StatusCode, patch.Body)
	}
}

func TestTaskFiltersAndDeleteAuthorization(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	ownerTok := registerAndToken(t, ts.URL, "owner2@example.com")
	otherTok := registerAndToken(t, ts.URL, "other2@example.com")

	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "P2"}, ownerTok)
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)

	// create task
	tr := doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "T1", "priority": "low"}, ownerTok)
	if tr.StatusCode != 201 {
		t.Fatalf("create task status=%d body=%s", tr.StatusCode, tr.Body)
	}
	var task struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(tr.Body), &task)

	// update status to done
	up := doReq(t, "PATCH", ts.URL+"/tasks/"+task.ID, map[string]any{"status": "done"}, ownerTok)
	if up.StatusCode != 200 {
		t.Fatalf("patch task status=%d body=%s", up.StatusCode, up.Body)
	}

	// filter by status
	list := doReq(t, "GET", ts.URL+"/projects/"+p.ID+"/tasks?status=done&limit=10&page=1", nil, ownerTok)
	if list.StatusCode != 200 {
		t.Fatalf("list tasks status=%d body=%s", list.StatusCode, list.Body)
	}

	// delete by non-owner/non-creator forbidden
	del := doReq(t, "DELETE", ts.URL+"/tasks/"+task.ID, nil, otherTok)
	if del.StatusCode != 403 {
		t.Fatalf("expected 403, got %d body=%s", del.StatusCode, del.Body)
	}
}

func TestAnyUserCanCreateTaskInOthersProject(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	ownerTok := registerAndToken(t, ts.URL, "projowner@example.com")
	creatorTok := registerAndToken(t, ts.URL, "taskcreator@example.com")

	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "Shared"}, ownerTok)
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)
	if pr.StatusCode != 201 || p.ID == "" {
		t.Fatalf("create project status=%d body=%s", pr.StatusCode, pr.Body)
	}

	tr := doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "By stranger", "priority": "medium"}, creatorTok)
	if tr.StatusCode != 201 {
		t.Fatalf("non-owner create task status=%d body=%s", tr.StatusCode, tr.Body)
	}
	var task struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(tr.Body), &task)

	// creator may delete own task
	delCreator := doReq(t, "DELETE", ts.URL+"/tasks/"+task.ID, nil, creatorTok)
	if delCreator.StatusCode != 204 {
		t.Fatalf("creator delete status=%d body=%s", delCreator.StatusCode, delCreator.Body)
	}

	// new task; owner deletes task they did not create
	tr2 := doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "For owner delete", "priority": "low"}, creatorTok)
	if tr2.StatusCode != 201 {
		t.Fatalf("second create status=%d body=%s", tr2.StatusCode, tr2.Body)
	}
	var task2 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(tr2.Body), &task2)
	delOwner := doReq(t, "DELETE", ts.URL+"/tasks/"+task2.ID, nil, ownerTok)
	if delOwner.StatusCode != 204 {
		t.Fatalf("owner delete creator task status=%d body=%s", delOwner.StatusCode, delOwner.Body)
	}
}

func TestUnauthorizedRequiresBearer(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	res := doReq(t, "GET", ts.URL+"/projects?page=1&limit=20", nil, "")
	if res.StatusCode != 401 {
		t.Fatalf("expected 401, got %d body=%s", res.StatusCode, res.Body)
	}
}

func TestValidationErrorShape(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	res := doJSON(t, ts.URL+"/auth/register", map[string]any{"name": "", "email": "", "password": ""}, "")
	if res.StatusCode != 400 {
		t.Fatalf("expected 400, got %d body=%s", res.StatusCode, res.Body)
	}
	var out struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	_ = json.Unmarshal([]byte(res.Body), &out)
	if out.Error != "validation failed" {
		t.Fatalf("expected validation failed, got %q body=%s", out.Error, res.Body)
	}
	if len(out.Fields) == 0 || out.Fields["email"] == "" || out.Fields["password"] == "" {
		t.Fatalf("expected field errors, got %+v body=%s", out.Fields, res.Body)
	}
}

func TestProjectDeleteCascadesTasks(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	ownerTok := registerAndToken(t, ts.URL, "cascade@example.com")

	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "Cascade"}, ownerTok)
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)

	tr := doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "T1", "priority": "low"}, ownerTok)
	if tr.StatusCode != 201 {
		t.Fatalf("create task status=%d body=%s", tr.StatusCode, tr.Body)
	}

	delProj := doReq(t, "DELETE", ts.URL+"/projects/"+p.ID, nil, ownerTok)
	if delProj.StatusCode != 204 {
		t.Fatalf("delete project status=%d body=%s", delProj.StatusCode, delProj.Body)
	}

	// project should be gone
	getProj := doReq(t, "GET", ts.URL+"/projects/"+p.ID, nil, ownerTok)
	if getProj.StatusCode != 404 {
		t.Fatalf("expected 404, got %d body=%s", getProj.StatusCode, getProj.Body)
	}
}

func TestProjectStatsEndpoint(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	ownerTok := registerAndToken(t, ts.URL, "stats@example.com")

	pr := doJSON(t, ts.URL+"/projects", map[string]any{"name": "Stats"}, ownerTok)
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(pr.Body), &p)

	_ = doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "T1", "priority": "low"}, ownerTok)
	_ = doJSON(t, ts.URL+"/projects/"+p.ID+"/tasks", map[string]any{"title": "T2", "priority": "high"}, ownerTok)

	stats := doReq(t, "GET", ts.URL+"/projects/"+p.ID+"/stats", nil, ownerTok)
	if stats.StatusCode != 200 {
		t.Fatalf("stats status=%d body=%s", stats.StatusCode, stats.Body)
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(stats.Body), &out)
	if out["by_status"] == nil || out["by_assignee"] == nil {
		t.Fatalf("missing keys in stats response: %s", stats.Body)
	}
}

func TestPaginationValidation(t *testing.T) {
	ctx := context.Background()
	ts, cleanup := newTestServer(t, ctx)
	defer cleanup()

	tok := registerAndToken(t, ts.URL, "page@example.com")
	res := doReq(t, "GET", ts.URL+"/projects?page=0&limit=-1", nil, tok)
	if res.StatusCode != 400 {
		t.Fatalf("expected 400, got %d body=%s", res.StatusCode, res.Body)
	}
}

type httpResult struct {
	StatusCode int
	Body       string
}

func doJSON(t *testing.T, url string, body any, token string) httpResult {
	t.Helper()
	return doReq(t, "POST", url, body, token)
}

func doReq(t *testing.T, method, url string, body any, token string) httpResult {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return httpResult{StatusCode: resp.StatusCode, Body: string(raw)}
}

func registerAndToken(t *testing.T, baseURL, email string) string {
	t.Helper()
	res := doJSON(t, baseURL+"/auth/register", map[string]any{"name": "X", "email": email, "password": "password123"}, "")
	if res.StatusCode != 201 {
		t.Fatalf("register status=%d body=%s", res.StatusCode, res.Body)
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal([]byte(res.Body), &out)
	return out.Token
}

func newTestServer(t *testing.T, ctx context.Context) (*httptest.Server, func()) {
	t.Helper()
	loadDotEnvIfPresent(t, ".env")
	loadDotEnvIfPresent(t, ".env.example")
	os.Setenv("JWT_SECRET", "test-secret-change-me")

	dsn := os.Getenv("TASKFLOW_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		t.Skip("TASKFLOW_TEST_DATABASE_URL not set; skipping integration test")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("test db not reachable (TASKFLOW_TEST_DATABASE_URL=%q): %v", dsn, err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Skipf("test db not reachable (TASKFLOW_TEST_DATABASE_URL=%q): %v", dsn, err)
	}

	// Reset schema to keep tests deterministic (requires the test DB to be dedicated).
	applyMigrations(t, ctx, pool, filepath.Join("internal", "storage", "postgres", "migrations"), "down")
	applyMigrations(t, ctx, pool, filepath.Join("internal", "storage", "postgres", "migrations"), "up")

	userRepo := postgres.NewPostgresUserRepository(pool)
	projectRepo := postgres.NewPostgresProjectRepository(pool)
	taskRepo := postgres.NewPostgresTaskRepository(pool)

	authSvc := service.NewAuthService(userRepo)
	projectSvc := service.NewProjectService(projectRepo, taskRepo)
	taskSvc := service.NewTaskService(projectRepo, taskRepo)

	h := httpapi.NewRouter(httpapi.Deps{Auth: authSvc, Projects: projectSvc, Tasks: taskSvc})
	srv := httptest.NewServer(h)

	cleanup := func() {
		srv.Close()
		pool.Close()
	}
	return srv, cleanup
}

func applySQLFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rel string) {
	t.Helper()
	abs := filepath.Join(repoRoot(t), rel)
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read %s: %v", abs, err)
	}
	sql := string(b)
	parts := strings.Split(sql, ";")
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		for strings.HasPrefix(stmt, "--") {
			if i := strings.Index(stmt, "\n"); i >= 0 {
				stmt = strings.TrimSpace(stmt[i+1:])
				continue
			}
			stmt = ""
			break
		}
		if stmt == "" {
			continue
		}
		// pgcrypto seed uses crypt/gen_salt; migration creates extension first.
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("exec %s: %v\nstmt=%s", rel, err, stmt)
		}
	}
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, relDir, direction string) {
	t.Helper()
	dir := filepath.Join(repoRoot(t), relDir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}

	paths := make([]string, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "."+direction+".sql") {
			continue
		}
		paths = append(paths, filepath.Join(relDir, name))
	}
	if len(paths) == 0 {
		t.Fatalf("no %s migrations found in %s", direction, relDir)
	}
	sort.Strings(paths)
	if direction == "down" {
		for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
			paths[i], paths[j] = paths[j], paths[i]
		}
	}
	for _, p := range paths {
		applySQLFile(t, ctx, pool, p)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// test runs from backend/test/integration; walk up to backend/ root
	root := wd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		root = filepath.Dir(root)
	}
	t.Fatalf("could not locate backend repo root from %s", wd)
	return ""
}

func loadDotEnvIfPresent(t *testing.T, filename string) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("TASKFLOW_TEST_DATABASE_URL")) != "" {
		return
	}

	path := filepath.Join(repoRoot(t), filename)
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}

func init() {
	// Increase default HTTP client timeout for container cold start.
	http.DefaultClient.Timeout = 10 * time.Second
}
