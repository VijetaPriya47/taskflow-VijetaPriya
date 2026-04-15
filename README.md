# TaskFlow Backend

## 1. Overview
In the three days given to me for this assignment, I spent the first day thinking about the best architecture. Since the MVP is so minimal, a basic Clean Architecture was enough, but since I had extra time to research and think, I decided an enterprise-grade Onion Architecture—incorporating Hexagonal principles with adapters (storage and transport) and ports (internal/domain)—would be a great implementation. Although I realize this is a massive overkill for the scope, I was able to comfortably implement it in 3 hours, so I did it.

This is a standalone monolithic backend for the **TaskFlow** assignment, serving as a git-versioning application (like GitHub or GitLab) where users create projects and task issues, and assignees can be added. 

**Assumptions**: Any authenticated user can make a task in any project, but deleting a task can only be performed by the project owner or the task owner.

**Tech Stack**:
- **Go**: `net/http` + `go-chi/chi`
- **DB**: PostgreSQL (via `pgxpool`)
- **Migrations**: `golang-migrate`
- **Auth**: bcrypt (cost 12) + JWT (24h expiry)
- **Logging**: `slog` (JSON in production)

## 2. Architecture Decisions
**Onion/Clean Layering & Hexagonal Architecture**:
- `internal/domain`: Core entities, interfaces (ports), and sentinel errors.
- `internal/application`: Use-cases enforcing authorization and business rules.
- `internal/transport/http`: REST handlers, middleware, and errors mapping (Adapter).
- `internal/storage/postgres`: Postgres repositories (Adapter).

***Why this structure?***
True to Clean Architecture, domain models are perfectly isolated. Outer layers depend inward. Graceful shutdown (`cmd/taskflow/main.go`) and structured request logging (`requestLogMiddleware`) do not interfere with core task-management logic.

***Design Tradeoff (Rule of Thumb)***: Code is split by concept only when it's substantial (handlers vs middleware vs helpers). I don't split small 5-30 line helpers just because I can to prevent things from becoming too verbose. 

***Architectural Tradeoff***: Fully breaking apart Clean Architecture for a small MVP adds structural boilerplate initially, but it guarantees completely flat, decoupled scaling and zero technical debt.

**Other specific decisions**:
- **Dedicated Validator Service**: Robust input validation rules have been isolated into a custom framework-agnostic package strictly at `internal/validator`. HTTP handlers depend on this utility to evaluate payload constraints, keeping the router code slim while preserving identical, predictable 400 error payload shapes.

 - ***Tradeoff***: Introduces another internal dependency to learn, but overwhelmingly benefits by keeping HTTP handlers razor-thin.

- **SQL Injection Prevention**: We prevent SQL injection by strictly using **parameterized queries** through the `pgxpool` driver. This ensures that user inputs (like emails or titles) are treated as literal data by Postgres, never as executable code.
  
- **BOLA Prevention (Authorization)**: To prevent **Broken Object Level Authorization**, ownership checks are enforced at the service layer ([`internal/application/task_service.go`](file:///home/anya/VijetaPriya/backend/internal/application/task_service.go)). For example, a task can only be deleted if the requester is either the **Project Owner** or the **Task Creator**.

- **Global Error Formatting & Typed Handlers**: Handlers return standardized error types to avoid boilerplate. Global helpers like `Validation()`, `RequireUser()`, etc. live in `internal/transport/http/respond.go`. They guarantee rigid responses (e.g., 400 Validation, 401 Unauthenticated).

 - ***Tradeoff***: Requires manually mapping Database errors to Domain errors and finally to HTTP wrappers, but cleanly untethers business logic entirely from HTTP context/status codes.

- **Auditability (Activity Logs)**: Implemented using the **Decorator Pattern**. An `ActivityService` decorator wraps core use-case services so the core stays purely focused on orchestration.

  -***Tradeoff Acknowledgment***: Using the Decorator pattern here is an intentional architectural flex to prioritize domain purity, even though it adds a slight degree of structural complexity to the initial setup. This approach tightly guards Clean Architecture alignment and ensures logging completeness across entrypoints without violating Separation of Concerns.
 
- **Structured Logging**: Leveraging Go's `log/slog`, application logs are emitted as structured JSON (standard in production).
  
  - **Where are the logs?**: Logs are written to `os.Stdout` (standard output). When running via Docker, you can view them live using `docker compose logs -f taskflow`.

  - **Observability**: This standardizes the log format, enabling automated log ingestion pipelines to easily parse metrics like request methodology, duration, remote IPs, and context-bound user data without complex parsing.
   
- **Context Propagation**: Every incoming request extracts or generates a unique `X-Request-Id`. This ID, along with the authenticated user information, is injected directly into Go's `context.Context`. It is then propagated through to the service layers, echoed back in the HTTP response headers, and automatically attached to all structured `slog` entries to ensure comprehensive downstream observability.
- -***Tradeoff***: Pollutes the Go context pipeline slightly from the edges inward, but massively pays off by ensuring a flawlessly linked observability tracing system for production debugging.
  
- **Foreign key Indexes**: Included essential lookup indexes to accelerate common queries:
  - `idx_tasks_project_id`: Rapidly accesses all tasks belonging to a single project.
  - `idx_tasks_assignee_id`: Optimizes filtering and searching tasks assigned to a specific user.
  - `idx_tasks_status`: Accelerates filtering tasks by their status (e.g., matching all "done" tasks).
  - `idx_projects_owner_id`: Quickly retrieves the list of projects a particular user owns.
 
- **Chi instead of Gin**: Used `go-chi` because it is perfectly compliant with standard `net/http`, lightweight, and sidesteps unnecessary performance tradeoffs where standard library simplicity handles the job effectively.

### Directory Structure & Responsibilities
```text
backend/
├── api/                   # Postman and Bruno API collections
├── cmd/
│   └── taskflow/          # Main entrypoint (main.go), sets up wiring and graceful shutdown
├── internal/
│   ├── application/       # Use-cases (Domain Ports). Enforces auth/ownership & business logic.
│   ├── domain/            # Core business entities, repository interfaces, and sentinel errors.
│   ├── storage/
│   │   └── postgres/      # Secondary Adapters: Data access implementations & SQL migrations.
│   ├── transport/
│   │   └── http/          # Primary Adapters: REST HTTP Handlers, Routing, and Middleware.
│   └── validator/         # Dedicated framework-agnostic payload validation logic.
└── test/
    └── integration/       # E2E test suite running against a live disposable Postgres DB.
```

## 3. Running Locally
Run the following commands to get the application up and running.

```bash
git clone https://github.com/your-name/taskflow
cd taskflow/backend
cp .env.example .env
docker compose up --build
```
*App is available at http://localhost:4000*

## 4. Running Migrations
Migrations run automatically on container start via `backend/docker-entrypoint.sh` using `golang-migrate`:
```bash
migrate -path /app/migrations -database "$DATABASE_URL" up
```
Migrations live in `backend/internal/storage/postgres/migrations/`. 

*Explicit Database Evolution*: Instead of a heavy `init.sql`, migrations are separated into clear numbered stages (e.g. `001_extensions`, `006_activities`, `007_seed`) providing perfect visibility into database growth.

**Manual Migrations (Local Postgres)**:
```bash
export DATABASE_URL='postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable'
migrate -path internal/storage/postgres/migrations -database "$DATABASE_URL" up
```

## 5. Test Credentials
The database is pre-seeded with the following credentials so you can log in immediately:

Email:    test@example.com
Password: password123

## 6. API Reference
You can test the API endpoints using the provided collections.

### Bruno (Recommended)
The Bruno collection is located in `backend/api/`. Simply open Bruno and choose "Open Collection", then select the `backend/api` folder.
*   **Environment**: Select the `local` environment.
*   **Variables**: The collection uses `{{baseUrl}}` (set to `http://localhost:4000`) and `{{token}}` (copy the JWT from Login response).

### Postman
The Postman collection is located at `backend/api/postman_collection.json`.
1.  **Import**: In Postman, click **Import** and select the `.json` file.
2.  **Base URL**: Set a collection variable or environment variable named `baseUrl` to `http://localhost:4000`.
3.  **Authentication**: After logging in, copy the `token` and paste it into the `token` variable. All requests use "Bearer Token" auth pointing to that variable.

### Key Endpoints
1. **Login**
   - `POST /auth/login`
   ```bash
   curl -sS -X POST http://localhost:4000/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"test@example.com","password":"password123"}'
   ```
   **Response (200)**: `{ "token": "<jwt>", "user": {...} }`

2. **List Projects**
   - `GET /projects?page=1&limit=20`
   ```bash
   curl -sS -H "Authorization: Bearer <jwt>" "http://localhost:4000/projects?page=1&limit=20"
   ```

3. **Create a Task**
   - `POST /projects/{id}/tasks`
   ```bash
   curl -sS -X POST "http://localhost:4000/projects/<project_id>/tasks" \
     -H "Authorization: Bearer <jwt>" \
     -H 'Content-Type: application/json' \
     -d '{"title":"Design homepage","priority":"high","assignee_id":null,"due_date":"2026-04-15"}'
   ```

**Summary of Endpoints**:
- **Auth**: `POST /auth/register`, `POST /auth/login`
- **Projects**: `GET /projects`, `POST /projects`, `GET /projects/{id}`, `PATCH /projects/{id}` (Owner), `DELETE /projects/{id}` (Owner), `GET /projects/{id}/stats` (Bonus)
- **Tasks**: `GET /projects/{id}/tasks`, `POST /projects/{id}/tasks`, `PATCH /tasks/{id}`, `DELETE /tasks/{id}` (Project Owner or Creator)

## 7. What You'd Do With More Time
- **Composite Indexes**: Add composite indexes on `project_id` + `assignee_id` to further optimize query performance for task assignments within projects.
- **Exponential Backoff**: We retry DB connection with exponential backoff to handle transient startup issues like delayed DB readiness.
- **Cicuit Breaker**: To prevent cascading failures when a dependency (like a DB, API, or service) is failing.
- **Read-Heavy Lock Optimizations**: Employ optimistic lock strategies or fine-grained read-write locks when concurrently modifying identical heavily-read projects.
- **Rate limiting**: Add protections (like token buckets) at authentication endpoints to prevent DOS attacks.
- **Authorization Expansion**: Add a feature to essentially ban users from participating or viewing private projects.
- **Better collaborator model**: Explicit membership schemas for projects instead of implicitly relying on tasks.


## 8. Tests
- **Integration Tests**: These run from `backend/test/integration/api_test.go`. They work perfectly alongside Docker provided that `TASKFLOW_TEST_DATABASE_URL` is configured to a valid DB url. Two approaches:
  1) Run Postgres in Docker and run tests on host:
     ```bash
     docker run --rm -p 55432:5432 -e POSTGRES_DB=taskflow_test -e POSTGRES_USER=taskflow -e POSTGRES_PASSWORD=taskflow postgres:16-alpine
     export TASKFLOW_TEST_DATABASE_URL="postgres://taskflow:taskflow@localhost:55432/taskflow_test?sslmode=disable"
     cd backend
     go test ./test/integration -run .
     ```
  2) Run tests from inside a Docker container using Docker's internal host resolution.
- **Unit Tests**: Found in `internal/application/`. Tests such as `auth_service_test.go`, `project_service_test.go`, and `task_service_test.go` cover service-layer behaviors, focusing heavily on inputs, sentinel error paths, and authorization rules matching.
