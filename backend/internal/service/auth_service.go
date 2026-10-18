package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	cfg    *config.Config
	repos  *domain.Repositories
	logger *slog.Logger
}

func NewAuthService(cfg *config.Config, repos *domain.Repositories, logger *slog.Logger) *AuthService {
	return &AuthService{
		cfg:    cfg,
		repos:  repos,
		logger: logger,
	}
}

type AuthTokenClaims struct {
	UserID    string          `json:"sub"`
	Email     string          `json:"email"`
	Name      string          `json:"name"`
	Role      domain.UserRole `json:"role"`
	IssuedAt  int64           `json:"iat"`
	ExpiresAt int64           `json:"exp"`
}

type LoginResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.repos.Users.GetByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, domain.NewAppError("INVALID_CREDENTIALS", "Invalid email or password", 401, nil)
	}

	if !user.Active {
		return nil, domain.NewAppError("ACCOUNT_DISABLED", "Your account is disabled", 403, nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.NewAppError("INVALID_CREDENTIALS", "Invalid email or password", 401, nil)
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "user.login",
		ActorType:    "user",
		ActorID:      user.ID,
		ResourceID:   user.ID,
		ResourceType: "user",
	})

	return &LoginResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (*LoginResult, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanEmail == "" || len(password) < 8 {
		return nil, domain.NewAppError("VALIDATION_ERROR", "Valid email and minimum 8-character password required", 400, nil)
	}

	existing, _ := s.repos.Users.GetByEmail(ctx, cleanEmail)
	if existing != nil {
		return nil, domain.NewAppError("EMAIL_EXISTS", "A user with this email already exists", 409, nil)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	count, _ := s.repos.Users.Count(ctx)
	if count > 0 && !s.cfg.AllowRegistration {
		return nil, domain.NewAppError("REGISTRATION_DISABLED", "Public user registration is disabled on this instance. Please contact the administrator.", 403, nil)
	}

	role := domain.RoleDeveloper
	if count == 0 {
		role = domain.RoleAdmin
		s.persistLocalEnv(cleanEmail)
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        cleanEmail,
		PasswordHash: string(hashed),
		Name:         strings.TrimSpace(name),
		Role:         role,
		Active:       true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) GenerateToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := AuthTokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour).Unix(),
	}

	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(signingInput))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sigB64, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (*AuthTokenClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed jwt token")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	if !hmac.Equal(actualSig, expectedSig) {
		return nil, errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	var claims AuthTokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("invalid claims format")
	}

	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return nil, errors.New("token has expired")
	}

	return &claims, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return s.repos.Users.GetByID(ctx, userID)
}

type SetupStatus struct {
	NeedsSetup        bool  `json:"needsSetup"`
	AllowRegistration bool  `json:"allowRegistration"`
	UserCount         int64 `json:"userCount"`
}

func (s *AuthService) GetSetupStatus(ctx context.Context) (*SetupStatus, error) {
	count, err := s.repos.Users.Count(ctx)
	if err != nil {
		return nil, err
	}
	return &SetupStatus{
		NeedsSetup:        count == 0,
		AllowRegistration: s.cfg.AllowRegistration,
		UserCount:         count,
	}, nil
}

func (s *AuthService) persistLocalEnv(adminEmail string) {
	// Look for existing .env in current directory or parent directory
	envCandidates := []string{".env", "../.env"}
	targetEnv := ".env"
	for _, p := range envCandidates {
		if _, err := os.Stat(p); err == nil {
			targetEnv = p
			break
		}
	}

	content := ""
	if data, err := os.ReadFile(targetEnv); err == nil {
		content = string(data)
	}

	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines)+5)
	hasAdminEmail := false
	hasDBDriver := false
	hasSQLitePath := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ADMIN_EMAIL=") {
			hasAdminEmail = true
			newLines = append(newLines, "ADMIN_EMAIL="+adminEmail)
			continue
		}
		if strings.HasPrefix(trimmed, "DATABASE_DRIVER=") {
			hasDBDriver = true
		}
		if strings.HasPrefix(trimmed, "SQLITE_PATH=") {
			hasSQLitePath = true
		}
		newLines = append(newLines, line)
	}

	if !hasAdminEmail {
		newLines = append(newLines, "ADMIN_EMAIL="+adminEmail)
	}
	if !hasDBDriver {
		newLines = append(newLines, "DATABASE_DRIVER=sqlite")
	}
	if !hasSQLitePath {
		newLines = append(newLines, "SQLITE_PATH=./data/authentic.db")
	}

	finalContent := strings.Join(newLines, "\n")
	if !strings.HasSuffix(finalContent, "\n") {
		finalContent += "\n"
	}

	if err := os.WriteFile(targetEnv, []byte(finalContent), 0644); err != nil {
		s.logger.Warn("could not persist admin config to local .env", "error", err)
	} else {
		s.logger.Info("persisted admin and sqlite config to local .env", "file", targetEnv, "adminEmail", adminEmail)
	}
}
