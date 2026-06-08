package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "api-gateway").Logger()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	temporalClient, err := client.Dial(client.Options{
		HostPort:      env("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace:     env("TEMPORAL_NAMESPACE", "default"),
		Logger:        temporalLogger{log: log},
		DataConverter: pipeline.NewDataConverter(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect temporal")
	}
	defer temporalClient.Close()

	authDB, err := sql.Open("postgres", env("DATABASE_DSN", "postgres://catalog:catalog@localhost:5432/smart_home?sslmode=disable"))
	if err != nil {
		log.Fatal().Err(err).Msg("open auth database")
	}
	defer authDB.Close()
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	if err := authDB.PingContext(dbCtx); err != nil {
		log.Fatal().Err(err).Msg("connect auth database")
	}

	registry := prometheus.NewRegistry()
	startedTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "api_gateway_workflows_started_total", Help: "Total workflow start requests grouped by status."},
		[]string{"status"},
	)
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}), startedTotal)

	apiServer := &http.Server{
		Addr:              env("HTTP_ADDRESS", ":8080"),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           buildAPI(log, temporalClient, startedTotal, newAuthService(authDB, env("JWT_SECRET", "dev-jwt-secret"), 24*time.Hour)),
	}
	metricsServer := &http.Server{
		Addr:              env("METRICS_ADDRESS", ":2116"),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}

	go serve(log, "api", apiServer)
	go serve(log, "metrics", metricsServer)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = apiServer.Shutdown(shutdownCtx)
	_ = metricsServer.Shutdown(shutdownCtx)
}

func buildAPI(log zerolog.Logger, temporalClient client.Client, startedTotal *prometheus.CounterVec, auth *authService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		var req credentialsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		access, refresh, err := auth.register(req.Email, req.Password)
		if err != nil {
			log.Warn().Str("event", "auth.register").Str("email", normalizeEmail(req.Email)).Err(err).Msg("registration rejected")
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, authResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    "Bearer",
			User:         &authUser{Email: normalizeEmail(req.Email), Name: req.Name},
		})
	})
	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req credentialsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		access, refresh, err := auth.login(req.Email, req.Password)
		if err != nil {
			log.Warn().Str("event", "auth.login").Str("email", normalizeEmail(req.Email)).Err(err).Msg("login rejected")
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeJSON(w, http.StatusOK, authResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    "Bearer",
			User:         &authUser{Email: normalizeEmail(req.Email)},
		})
	})
	mux.HandleFunc("POST /auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.RefreshToken == "" {
			writeError(w, http.StatusUnauthorized, "refresh_token is required")
			return
		}
		subject, access, refresh, err := auth.refresh(req.RefreshToken)
		if err != nil {
			log.Warn().Str("event", "auth.refresh").Err(err).Msg("refresh rejected")
			writeError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		writeJSON(w, http.StatusOK, authResponse{
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    "Bearer",
			User:         &authUser{Email: subject},
		})
	})
	mux.HandleFunc("POST /auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		var req forgotPasswordRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		resetToken, err := auth.createResetToken(req.Email)
		if err != nil {
			log.Error().Str("event", "auth.forgot-password").Str("email", normalizeEmail(req.Email)).Err(err).Msg("reset token creation failed")
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, forgotPasswordResponse{
			Message:    "if the user exists, a reset token was generated",
			ResetToken: resetToken,
		})
	})
	mux.HandleFunc("POST /auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
		var req resetPasswordRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if err := auth.resetPassword(req.Email, req.ResetToken, req.NewPassword); err != nil {
			log.Warn().Str("event", "auth.reset-password").Str("email", normalizeEmail(req.Email)).Err(err).Msg("password reset rejected")
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "password reset"})
	})
	mux.HandleFunc("POST /start", func(w http.ResponseWriter, r *http.Request) {
		if !auth.authorized(r) {
			startedTotal.WithLabelValues("unauthorized").Inc()
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req pipeline.PipelineRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			startedTotal.WithLabelValues("invalid").Inc()
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := validate(req); err != nil {
			startedTotal.WithLabelValues("invalid").Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.RequestID == "" {
			req.RequestID = fmt.Sprintf("manual-%s", time.Now().UTC().Format("20060102-150405"))
		}

		run, err := temporalClient.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
			ID:        "main-pipeline-" + req.RequestID,
			TaskQueue: workflows.MainPipelineTaskQueue,
		}, workflows.MainPipelineWorkflow, req)
		if err != nil {
			startedTotal.WithLabelValues("failure").Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		startedTotal.WithLabelValues("success").Inc()
		log.Info().Str("workflow_id", run.GetID()).Str("run_id", run.GetRunID()).Msg("workflow started")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"workflow_id": run.GetID(), "run_id": run.GetRunID()})
	})
	resultHandler := func(w http.ResponseWriter, r *http.Request) {
		if !auth.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		workflowID := r.PathValue("workflow_id")
		if workflowID == "" {
			workflowID = r.URL.Query().Get("workflow_id")
		}
		runID := r.URL.Query().Get("run_id")
		if workflowID == "" {
			http.Error(w, "workflow_id is required; use /result/{workflow_id} or /result?workflow_id=...", http.StatusBadRequest)
			return
		}

		description, err := temporalClient.DescribeWorkflowExecution(r.Context(), workflowID, runID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		status := description.WorkflowExecutionInfo.Status
		if status != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"workflow_id": workflowID,
				"run_id":      description.WorkflowExecutionInfo.Execution.RunId,
				"status":      status.String(),
			})
			return
		}

		var result pipeline.PipelineResult
		if err := temporalClient.GetWorkflow(r.Context(), workflowID, runID).Get(r.Context(), &result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
	mux.HandleFunc("GET /result", resultHandler)
	mux.HandleFunc("GET /result/{workflow_id}", resultHandler)
	return mux
}

func validate(req pipeline.PipelineRequest) error {
	if req.FloorPlan == nil {
		return errors.New("floor_plan is required")
	}
	if req.SelectedLevels == nil || len(req.SelectedLevels) == 0 {
		return errors.New("selected_levels is required")
	}
	if req.DeviceSelection == nil {
		return errors.New("device_selection is required")
	}
	return nil
}

func serve(log zerolog.Logger, name string, server *http.Server) {
	log.Info().Str("server", name).Str("address", server.Addr).Msg("listening")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Str("server", name).Msg("server stopped")
	}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

type authUser struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type authResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	User         *authUser `json:"user,omitempty"`
}

type forgotPasswordResponse struct {
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"`
}

type authService struct {
	db         *sql.DB
	jwtSecret  []byte
	tokenTTL   time.Duration
	refreshTTL time.Duration
}

type userRecord struct {
	Email            string
	Salt             string
	PasswordHash     string
	ResetTokenHash   string
	ResetTokenExpiry *time.Time
}

func newAuthService(db *sql.DB, secret string, tokenTTL time.Duration) *authService {
	return &authService{
		db:         db,
		jwtSecret:  []byte(secret),
		tokenTTL:   tokenTTL,
		refreshTTL: 30 * 24 * time.Hour,
	}
}

func (a *authService) register(email, password string) (string, string, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	salt := randomToken(16)
	_, err := a.db.ExecContext(ctx, `
		INSERT INTO api_users (email, salt, password_hash)
		VALUES ($1, $2, $3)
	`, email, salt, hashPassword(salt, password))
	if isUniqueViolation(err) {
		return "", "", errors.New("user already exists")
	}
	if err != nil {
		return "", "", err
	}

	return a.issueTokens(email)
}

func (a *authService) login(email, password string) (string, string, error) {
	email = normalizeEmail(email)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := a.getUser(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", errors.New("invalid credentials")
	}
	if err != nil {
		return "", "", err
	}

	expected := hashPassword(user.Salt, password)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(user.PasswordHash)) != 1 {
		return "", "", errors.New("invalid credentials")
	}
	return a.issueTokens(email)
}

// refresh проверяет refresh-токен и выпускает новую пару токенов.
func (a *authService) refresh(refreshToken string) (string, string, string, error) {
	subject, err := a.validateJWT(refreshToken)
	if err != nil {
		return "", "", "", err
	}
	access, refresh, err := a.issueTokens(subject)
	if err != nil {
		return "", "", "", err
	}
	return subject, access, refresh, nil
}

func (a *authService) createResetToken(email string) (string, error) {
	email = normalizeEmail(email)
	resetToken := randomToken(32)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := a.getUser(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return resetToken, nil
	}
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	if _, err := a.db.ExecContext(ctx, `
		UPDATE api_users
		SET reset_token_hash = $2,
			reset_token_expires_at = $3,
			updated_at = NOW()
		WHERE email = $1
	`, email, hashPassword(user.Salt, resetToken), expiresAt); err != nil {
		return "", err
	}

	return resetToken, nil
}

func (a *authService) resetPassword(email, resetToken, newPassword string) error {
	email = normalizeEmail(email)
	if err := validateCredentials(email, newPassword); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := a.getUser(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("invalid or expired reset token")
	}
	if err != nil {
		return err
	}
	if user.ResetTokenHash == "" || user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		return errors.New("invalid or expired reset token")
	}
	expected := hashPassword(user.Salt, resetToken)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(user.ResetTokenHash)) != 1 {
		return errors.New("invalid or expired reset token")
	}
	salt := randomToken(16)
	_, err = a.db.ExecContext(ctx, `
		UPDATE api_users
		SET salt = $2,
			password_hash = $3,
			reset_token_hash = NULL,
			reset_token_expires_at = NULL,
			updated_at = NOW()
		WHERE email = $1
	`, email, salt, hashPassword(salt, newPassword))
	return err
}

func (a *authService) getUser(ctx context.Context, email string) (*userRecord, error) {
	var user userRecord
	var resetTokenHash sql.NullString
	var resetTokenExpiry sql.NullTime
	err := a.db.QueryRowContext(ctx, `
		SELECT email, salt, password_hash, reset_token_hash, reset_token_expires_at
		FROM api_users
		WHERE email = $1
	`, email).Scan(
		&user.Email,
		&user.Salt,
		&user.PasswordHash,
		&resetTokenHash,
		&resetTokenExpiry,
	)
	if err != nil {
		return nil, err
	}
	if resetTokenHash.Valid {
		user.ResetTokenHash = resetTokenHash.String
	}
	if resetTokenExpiry.Valid {
		expiry := resetTokenExpiry.Time
		user.ResetTokenExpiry = &expiry
	}
	return &user, nil
}

func (a *authService) authorized(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		return false
	}
	_, err := a.validateJWT(token)
	return err == nil
}

func (a *authService) issueTokens(subject string) (string, string, error) {
	access, err := a.issueJWT(subject, a.tokenTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err := a.issueJWT(subject, a.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (a *authService) issueJWT(subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := map[string]interface{}{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := signHS256(unsigned, a.jwtSecret)
	return unsigned + "." + signature, nil
}

func (a *authService) validateJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := signHS256(unsigned, a.jwtSecret)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return "", errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		Subject string `json:"sub"`
		Expires int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}
	if claims.Subject == "" || time.Now().Unix() >= claims.Expires {
		return "", errors.New("expired token")
	}
	return claims.Subject, nil
}

func signHS256(unsigned string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func bearerToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[len("Bearer "):])
}

func validateCredentials(email, password string) error {
	if email == "" || !strings.Contains(email, "@") {
		return errors.New("valid email is required")
	}
	if len(password) < 8 {
		return errors.New("password must contain at least 8 characters")
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashPassword(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func randomToken(size int) string {
	data := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError отдаёт ошибку JSON-ом {"message": ...}, как ждёт фронтенд
// (ApiErrorResponse), вместо text/plain от http.Error.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type temporalLogger struct{ log zerolog.Logger }

func (l temporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.log.Debug().Fields(fields(keyvals...)).Msg(msg)
}
func (l temporalLogger) Info(msg string, keyvals ...interface{}) {
	l.log.Info().Fields(fields(keyvals...)).Msg(msg)
}
func (l temporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.log.Warn().Fields(fields(keyvals...)).Msg(msg)
}
func (l temporalLogger) Error(msg string, keyvals ...interface{}) {
	l.log.Error().Fields(fields(keyvals...)).Msg(msg)
}

func fields(keyvals ...interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i+1 < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			out[key] = keyvals[i+1]
		}
	}
	return out
}
