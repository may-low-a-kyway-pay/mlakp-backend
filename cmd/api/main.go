package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mlakp-backend/internal/app"
	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/config"
	"mlakp-backend/internal/dashboard"
	"mlakp-backend/internal/debts"
	"mlakp-backend/internal/email"
	"mlakp-backend/internal/expenses"
	"mlakp-backend/internal/groups"
	"mlakp-backend/internal/httpapi/handlers"
	"mlakp-backend/internal/notifications"
	"mlakp-backend/internal/otp"
	"mlakp-backend/internal/payments"
	"mlakp-backend/internal/postgres"
	"mlakp-backend/internal/postgres/sqlc"
	"mlakp-backend/internal/sessions"
	"mlakp-backend/internal/users"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	dbPool, err := postgres.OpenPool(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Wire dependencies manually to keep startup explicit as the domain grows.
	queries := sqlc.New(dbPool)
	userRepository := users.NewRepository(queries)
	passwordHasher := auth.BcryptHasher{}
	userService := users.NewService(userRepository, passwordHasher)
	tokenManager := auth.NewTokenManager(cfg.TokenIssuer, cfg.TokenAudience, cfg.TokenSecret, cfg.AccessTokenTTL)
	sessionRepository := sessions.NewRepository(queries)
	sessionService := sessions.NewService(sessionRepository, cfg.RefreshTokenTTL)
	groupRepository := groups.NewRepository(dbPool, queries)
	groupService := groups.NewService(groupRepository)
	notificationHub := notifications.NewHub()
	notificationRepository := notifications.NewRepository(queries)
	notificationService := notifications.NewService(notificationRepository, notificationHub)
	expenseRepository := expenses.NewRepository(dbPool, queries)
	expenseService := expenses.NewService(expenseRepository, notificationService)
	debtRepository := debts.NewRepository(queries)
	debtService := debts.NewService(debtRepository, notificationService)
	paymentRepository := payments.NewRepository(dbPool, queries)
	paymentService := payments.NewService(paymentRepository, notificationService)
	dashboardRepository := dashboard.NewRepository(queries)
	dashboardService := dashboard.NewService(dashboardRepository)

	emailProvider := email.NewProvider(email.Config{
		APIKey:    cfg.PostmarkAPIKey,
		FromEmail: cfg.PostmarkFromEmail,
		FromName:  cfg.PostmarkFromName,
		Endpoint:  "https://api.postmarkapp.com/email",
	})

	otpRepository := otp.NewRepository(dbPool)
	otpService := otp.NewService(otpRepository, otp.Config{
		ExpiryMinutes:     cfg.OTPExpiryMinutes,
		RequestCooldown:   cfg.OTPRequestCooldown,
		MaxAttempts:       cfg.OTPMaxAttempts,
		RequestsPerWindow: cfg.OTPRequestsPerWindow,
		WindowMins:        cfg.OTPRequestWindowMins,
	})

	server := &http.Server{
		Addr: fmt.Sprintf(":%s", cfg.AppPort),
		Handler: app.NewRouter(logger, app.RouterDeps{
			AuthHandler:         handlers.NewAuthHandler(userService, tokenManager, sessionService),
			UserHandler:         handlers.NewUserHandler(userService),
			GroupHandler:        handlers.NewGroupHandler(groupService, userService),
			ExpenseHandler:      handlers.NewExpenseHandler(expenseService),
			DebtHandler:         handlers.NewDebtHandler(debtService),
			PaymentHandler:      handlers.NewPaymentHandler(paymentService),
			DashboardHandler:    handlers.NewDashboardHandler(dashboardService),
			NotificationHandler: handlers.NewNotificationHandler(notificationService, notificationHub, cfg.CORSOrigins),
			OTPHandler:          handlers.NewOTPHandler(userService, otpService, emailProvider, sessionService, tokenManager),
			UsersService:        userService,
			TokenManager:        tokenManager,
			SessionService:      sessionService,
			AppEnv:              cfg.AppEnv,
			CORSOrigins:         cfg.CORSOrigins,
			ReadinessChecker:    dbPool,
		}),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting api server", "addr", server.Addr, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("api server stopped with error", "error", err)
			os.Exit(1)
		}
	case signal := <-shutdownCh:
		logger.Info("shutdown signal received", "signal", signal.String())

		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("api server shutdown failed", "error", err)
			os.Exit(1)
		}

		logger.Info("api server stopped")
	}
}
