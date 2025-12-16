# GophKeeper - Secure Password Manager

GophKeeper is a client-server password manager system built in Go. It provides secure storage and synchronization of private user data including passwords, text notes, binary files, and credit card information.

## Features

- 🔐 **End-to-end encryption** - All data is encrypted client-side before transmission
- 🔄 **Multi-device synchronization** - Keep your secrets in sync across devices
- 📱 **Cross-platform CLI client** - Works on Linux, macOS, and Windows
- 🗄️ **Multiple data types** - Store passwords, text, binary files, and credit cards
- 🔑 **Zero-knowledge architecture** - Server never has access to your encryption keys
- 📦 **Offline support** - Local storage with automatic sync when online

## Architecture

```
┌─────────────┐         HTTPS/TLS        ┌─────────────────┐
│   Client    │ ◄──────────────────────► │     Server      │
│   (CLI)     │                          │   (REST API)    │
├─────────────┤                          ├─────────────────┤
│ Local SQLite│                          │   PostgreSQL    │
│   Storage   │                          │    Database     │
└─────────────┘                          └─────────────────┘
       │                                          │
       │    Encrypted Data                        │
       └──────────────────────────────────────────┘
```

## Security Model

- **Encryption**: AES-256-GCM for symmetric encryption
- **Key Derivation**: Argon2id for deriving keys from passwords
- **Authentication**: JWT tokens with short-lived access tokens and long-lived refresh tokens
- **Password Hashing**: bcrypt for server-side password verification
- **Transport Security**: TLS 1.3 for all communications

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/volchkovski/gophkeeper.git
cd gophkeeper

# Build server and client
make build

# Binaries will be in ./bin/
```

### Using Docker

```bash
# Start server and database
docker-compose up -d

# Run migrations
make migrate-up
```

## Quick Start

### Server Setup

1. Configure environment variables (copy `.env.example` to `.env`):

```bash
cp .env.example .env
# Edit .env with your settings
```

2. Start the database:

```bash
docker-compose up -d postgres
```

3. Run migrations:

```bash
make migrate-up
```

4. Start the server:

```bash
make run-server
# or
./bin/gophkeeper-server
```

### Client Usage

1. Register a new account:

```bash
./bin/gophkeeper register
# Enter username and password when prompted
```

2. Login:

```bash
./bin/gophkeeper login
```

3. Add a password:

```bash
./bin/gophkeeper add password
# Follow the prompts to enter details
```

4. List all secrets:

```bash
./bin/gophkeeper list
```

5. Get a secret:

```bash
./bin/gophkeeper get "My Gmail"
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gophkeeper register` | Register a new user |
| `gophkeeper login` | Login to GophKeeper |
| `gophkeeper logout` | Logout from GophKeeper |
| `gophkeeper add password` | Add login/password |
| `gophkeeper add text` | Add text note |
| `gophkeeper add binary <file>` | Add binary file |
| `gophkeeper add card` | Add credit card |
| `gophkeeper list` | List all secrets |
| `gophkeeper get <name>` | Get a secret by name |
| `gophkeeper edit <name>` | Edit a secret |
| `gophkeeper delete <name>` | Delete a secret |
| `gophkeeper sync` | Synchronize with server |
| `gophkeeper export <name> <file>` | Export binary secret to file |
| `gophkeeper config` | Show configuration |
| `gophkeeper version` | Show version info |

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login and get tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### Secrets

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/secrets` | List all secrets |
| POST | `/api/v1/secrets` | Create new secret |
| GET | `/api/v1/secrets/:id` | Get secret by ID |
| PUT | `/api/v1/secrets/:id` | Update secret |
| DELETE | `/api/v1/secrets/:id` | Delete secret |
| POST | `/api/v1/secrets/sync` | Sync secrets |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/version` | Server version |

## Configuration

### Server Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server host | `localhost` |
| `SERVER_PORT` | Server port | `8080` |
| `DATABASE_DSN` | PostgreSQL connection string | - |
| `JWT_SECRET` | JWT signing secret | - |
| `JWT_ACCESS_TOKEN_TTL` | Access token TTL | `1h` |
| `JWT_REFRESH_TOKEN_TTL` | Refresh token TTL | `720h` |
| `SECURITY_BCRYPT_COST` | bcrypt cost factor | `12` |
| `SECURITY_RATE_LIMIT` | Requests per second | `100` |
| `TLS_ENABLED` | Enable TLS | `false` |
| `TLS_CERT_FILE` | TLS certificate path | - |
| `TLS_KEY_FILE` | TLS key path | - |
| `LOG_LEVEL` | Log level | `info` |

### Client Configuration

Configuration file: `~/.gophkeeper/config.json`

```json
{
  "server_url": "http://localhost:8080",
  "timeout": "30s",
  "auto_sync": true,
  "sync_on_start": true
}
```

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 15+ (or Docker)
- Make

### Setup Development Environment

```bash
# Install development tools
make install-tools

# Download dependencies
make deps

# Start database
docker-compose up -d postgres

# Run migrations
make migrate-up

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
gophkeeper/
├── cmd/
│   ├── server/          # Server entry point
│   └── client/          # Client entry point
├── internal/
│   ├── server/          # Server implementation
│   │   ├── handlers/    # HTTP handlers
│   │   ├── middleware/  # HTTP middleware
│   │   ├── models/      # Data models
│   │   ├── repository/  # Database layer
│   │   ├── service/     # Business logic
│   │   └── config/      # Configuration
│   ├── client/          # Client implementation
│   │   ├── commands/    # CLI commands
│   │   ├── api/         # API client
│   │   ├── storage/     # Local storage
│   │   └── config/      # Configuration
│   └── common/          # Shared code
│       ├── crypto/      # Encryption utilities
│       ├── models/      # Shared models
│       └── errors/      # Error definitions
├── pkg/
│   ├── logger/          # Logging
│   └── validator/       # Validation
├── migrations/          # Database migrations
└── tests/               # Test suites
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run only unit tests
make test-unit
```

### Building for Multiple Platforms

```bash
make build-all
# Creates binaries for:
# - Linux AMD64
# - macOS AMD64
# - macOS ARM64
# - Windows AMD64
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [sqlx](https://github.com/jmoiron/sqlx) - Database extensions
- [zap](https://github.com/uber-go/zap) - Logging

