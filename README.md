# Go REST API

A backend REST API built with Go, PostgreSQL, and the standard `net/http` stack.

The project focuses on authentication, authorization, database persistence, secure session management, automated testing, and containerized development.

## Features

* User registration and authentication
* Access tokens using JWT
* Refresh-token based sessions
* Refresh-token rotation and reuse detection
* Session-family revocation
* Argon2id password hashing
* Role-based authorization
* Self-resource and privileged-resource access control
* PostgreSQL persistence with SQL migrations
* Structured application errors
* Request ID, logging, and panic-recovery middleware
* Graceful HTTP server shutdown
* Docker and Docker Compose support
* Unit, integration, and race-detector tests
* GitHub Actions CI

## Tech Stack

* **Go 1.27**
* **PostgreSQL**
* **pgx/v5**
* **golang-migrate**
* **JWT**
* **Argon2id**
* **Docker / Docker Compose**
* **GitHub Actions**

## Architecture

The application follows a layered structure that keeps HTTP handling, business logic, persistence, authentication, and domain concerns separated.

```text
cmd/
└── api/
    └── main.go

internal/
├── app/              # Application wiring and route registration
├── auth/             # Password hashing and JWT handling
├── authz/            # Roles and permissions
├── config/           # Environment-based configuration
├── database/         # PostgreSQL connection setup
├── domain/           # Domain types and domain errors
├── http/
│   ├── handler/      # HTTP handlers
│   ├── middleware/   # Authentication, logging, recovery, request IDs
│   └── response/     # Cookies and structured HTTP errors
├── repository/
│   └── postgres/     # PostgreSQL repositories
├── service/          # Application/business logic
└── validation/       # Request validation

migrations/            # SQL database migrations
```

Application wiring is centralized in `internal/app`, while services depend on repository interfaces rather than concrete PostgreSQL implementations.

## Authentication

The authentication system uses a short-lived access token together with a longer-lived refresh-token session.

### Registration

`POST /auth/register`

A successful registration:

1. Normalizes the email address.
2. Hashes the password with Argon2id.
3. Creates a user with the default `user` role.
4. Generates a cryptographically random refresh token.
5. Stores only the refresh-token hash in PostgreSQL.
6. Creates an authentication session.
7. Issues an access token.
8. Returns the authentication state through secure cookies.

### Login

`POST /auth/login`

Login verifies the supplied password against the stored Argon2id hash and creates a new refresh-token session.

For an unknown user, the service also performs password verification against a dummy hash before returning the same authentication error. This helps avoid making account existence distinguishable through password-verification timing.

### Refresh

`POST /auth/refresh`

Refresh tokens are rotated rather than reused indefinitely.

Each refresh operation:

* hashes the presented refresh token
* identifies the existing session
* creates a replacement refresh token
* invalidates the previous token
* preserves the session family
* issues a new access token

A reused refresh token causes the corresponding session family to be revoked.

### Logout

`POST /auth/logout`

The refresh token is invalidated and both authentication cookies are cleared.

## Token Security

Access tokens are JWTs signed with `HS256` and include standard claims such as:

* subject
* issuer
* audience
* expiration
* issued-at time
* token ID

The JWT verifier explicitly checks the expected signing algorithm, token type, issuer, audience, expiration, subject, token ID, and issued-at claim.

Refresh tokens are generated using cryptographically secure randomness. Their hashes, rather than the raw tokens, are persisted in the database.

Authentication cookies are configured as:

* `HttpOnly`
* `Secure`
* `SameSite=Strict`

The access-token cookie uses the `__Host-` prefix and the refresh-token cookie uses the `__Secure-` prefix.

## Authorization

Authorization is evaluated separately from authentication.

The application currently defines two roles:

```text
user
admin
```

and two permissions:

```text
user:read:self
user:read:any
```

The `user` role can access its own user resource, while the `admin` role can access both its own resource and other users' resources.

Roles are resolved from the server-side user record rather than being trusted as authorization data inside the JWT.

## API

### Health

```text
GET /health/live
GET /health/ready
```

### Users

```text
POST /user
GET  /user/{id}
GET  /user
```

`GET /user/{id}` requires authentication.

`GET /user` requires the `user:read:any` permission.

### Authentication

```text
POST /auth/register
POST /auth/login
POST /auth/refresh
POST /auth/logout
```

## Example: Register

```bash
curl -i \
  -X POST http://localhost:8099/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "correct-password"
  }'
```

A successful registration responds with:

```text
HTTP/1.1 201 Created
```

and sets the authentication cookies.

## Example: Login

```bash
curl -i \
  -X POST http://localhost:8099/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "correct-password"
  }'
```

## Example: Access an authenticated resource

```bash
curl -i \
  --cookie-jar cookies.txt \
  --cookie cookies.txt \
  http://localhost:8099/user/<user-id>
```

## Error Handling

Application errors use structured JSON responses.

Example:

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "authentication required"
  }
}
```

Common application-level error codes include:

```text
INVALID_REQUEST
VALIDATION_FAILED
NOT_FOUND
CONFLICT
UNAUTHORIZED
FORBIDDEN
INVALID_PATH_PARAMETER
INVALID_QUERY_PARAMETER
INTERNAL_ERROR
```

Errors originating from repository or infrastructure failures are logged and exposed to clients as internal server errors.

## Configuration

Configuration is loaded from environment variables.

Create a local environment file:

```bash
cp .env.example .env
```

At minimum, provide:

```env
JWT_SECRET=your-secret
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rest_pro?sslmode=disable
```

Other supported settings include:

```env
HTTP_ADDR=
ENV=
JWT_ISSUER=
JWT_AUDIENCE=
ACCESS_TOKEN_TTL=
REFRESH_TOKEN_TTL=
REFRESH_IDLE_TTL=
TLS_ENABLED=
TLS_CERT_FILE=
TLS_KEY_FILE=
SHUTDOWN_TIMEOUT=
```

See `.env.example` for the complete configuration surface.

## Running with Docker Compose

The repository includes a Compose setup containing:

* PostgreSQL
* database migrations
* the API

Start the complete stack with:

```bash
docker compose up --build
```

The API is exposed on:

```text
http://localhost:8099
```

The Compose configuration waits for PostgreSQL to become healthy, runs the migrations, and then starts the API.

To stop the stack:

```bash
docker compose down
```

To remove the persistent PostgreSQL volume as well:

```bash
docker compose down -v
```

## Running Locally

Start PostgreSQL and create the required database, then configure `DATABASE_URL` in `.env`.

Run the migrations with `golang-migrate`, then start the API:

```bash
go run ./cmd/api
```

The default HTTP address is:

```text
:8091
```

Set `HTTP_ADDR=:8099` when you want the local server to use the same port as the Docker setup.

## Testing

Run all unit tests:

```bash
go test ./...
```

Run integration tests:

```bash
go test -tags=integration ./...
```

Run the race detector against the integration suite:

```bash
go test -race -tags=integration ./...
```

Check formatting:

```bash
test -z "$(gofmt -l .)"
```

Run static analysis:

```bash
go vet ./...
```

Build the Docker image:

```bash
docker build -t rest:ci .
```

## Continuous Integration

GitHub Actions runs the project's quality checks automatically.

The CI pipeline checks:

```text
gofmt
go vet
unit tests
integration tests
race detector
Docker image build
```

The integration and race-detector jobs use a PostgreSQL service container.

## Database

The application uses PostgreSQL through `pgx/v5`.

Database schema changes are tracked using versioned SQL migrations under:

```text
migrations/
```

The main persistence model consists of users and authentication sessions.

The users table stores:

```text
id
email
password_hash
role
created_at
```

Authentication sessions store:

```text
id
family_id
user_id
refresh_token_hash
created_at
expires_at
last_used_at
revoked_at
replaced_by
```

Refresh-token hashes and session-family identifiers are indexed to support session management and token rotation.

## Design Goals

The project is intended to demonstrate practical backend engineering rather than only framework usage.

The main design goals are:

* explicit separation of concerns
* dependency inversion through interfaces
* secure authentication and session handling
* server-side authorization
* context propagation across application layers
* predictable error handling
* database-backed state where revocation is required
* automated verification through unit and integration tests
* reproducible development through Docker
* CI validation before changes are merged

## Project Status

This project is under active development.

The core authentication, authorization, PostgreSQL persistence, testing, and containerization foundations are implemented, while additional API functionality and hardening can continue to be added as the project evolves.
