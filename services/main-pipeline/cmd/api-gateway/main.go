package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/workflows"
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
		HostPort:  env("TEMPORAL_ADDRESS", "localhost:7233"),
		Namespace: env("TEMPORAL_NAMESPACE", "default"),
		Logger:    temporalLogger{log: log},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect temporal")
	}
	defer temporalClient.Close()

	registry := prometheus.NewRegistry()
	startedTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "api_gateway_workflows_started_total", Help: "Total workflow start requests grouped by status."},
		[]string{"status"},
	)
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}), startedTotal)

	apiServer := &http.Server{
		Addr:              env("HTTP_ADDRESS", ":8080"),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           buildAPI(log, temporalClient, startedTotal, newAuthService(env("JWT_SECRET", "dev-jwt-secret"), 24*time.Hour)),
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
		token, err := auth.register(req.Email, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, tokenResponse{AccessToken: token, TokenType: "Bearer"})
	})
	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req credentialsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		token, err := auth.login(req.Email, req.Password)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, tokenResponse{AccessToken: token, TokenType: "Bearer"})
	})
	mux.HandleFunc("POST /auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		var req forgotPasswordRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		resetToken, _ := auth.createResetToken(req.Email)
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
			http.Error(w, err.Error(), http.StatusBadRequest)
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
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type forgotPasswordResponse struct {
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"`
}

type authService struct {
	mu        sync.RWMutex
	users     map[string]*userRecord
	jwtSecret []byte
	tokenTTL  time.Duration
}

type userRecord struct {
	Email            string
	Salt             string
	PasswordHash     string
	ResetTokenHash   string
	ResetTokenExpiry time.Time
}

func newAuthService(secret string, tokenTTL time.Duration) *authService {
	return &authService{
		users:     make(map[string]*userRecord),
		jwtSecret: []byte(secret),
		tokenTTL:  tokenTTL,
	}
}

func (a *authService) register(email, password string) (string, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.users[email]; exists {
		return "", errors.New("user already exists")
	}
	salt := randomToken(16)
	a.users[email] = &userRecord{
		Email:        email,
		Salt:         salt,
		PasswordHash: hashPassword(salt, password),
	}
	return a.issueJWT(email)
}

func (a *authService) login(email, password string) (string, error) {
	email = normalizeEmail(email)
	a.mu.RLock()
	user, exists := a.users[email]
	a.mu.RUnlock()
	if !exists {
		return "", errors.New("invalid credentials")
	}
	expected := hashPassword(user.Salt, password)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(user.PasswordHash)) != 1 {
		return "", errors.New("invalid credentials")
	}
	return a.issueJWT(email)
}

func (a *authService) createResetToken(email string) (string, error) {
	email = normalizeEmail(email)
	resetToken := randomToken(32)
	a.mu.Lock()
	defer a.mu.Unlock()
	if user, exists := a.users[email]; exists {
		user.ResetTokenHash = hashPassword(user.Salt, resetToken)
		user.ResetTokenExpiry = time.Now().Add(15 * time.Minute)
	}
	return resetToken, nil
}

func (a *authService) resetPassword(email, resetToken, newPassword string) error {
	email = normalizeEmail(email)
	if err := validateCredentials(email, newPassword); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	user, exists := a.users[email]
	if !exists || user.ResetTokenHash == "" || time.Now().After(user.ResetTokenExpiry) {
		return errors.New("invalid or expired reset token")
	}
	expected := hashPassword(user.Salt, resetToken)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(user.ResetTokenHash)) != 1 {
		return errors.New("invalid or expired reset token")
	}
	user.Salt = randomToken(16)
	user.PasswordHash = hashPassword(user.Salt, newPassword)
	user.ResetTokenHash = ""
	user.ResetTokenExpiry = time.Time{}
	return nil
}

func (a *authService) authorized(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		return false
	}
	_, err := a.validateJWT(token)
	return err == nil
}

func (a *authService) issueJWT(subject string) (string, error) {
	now := time.Now()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := map[string]interface{}{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(a.tokenTTL).Unix(),
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
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
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
