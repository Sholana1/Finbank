package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	identityv1 "github.com/sholana1/fintechdemo/gen/identity/v1"
	"github.com/sholana1/fintechdemo/identity/internal/app"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type userRegister interface {
	Execute(ctx context.Context, in app.RegisterUserInput) (domain.User, error)
}

type userLogin interface {
	Execute(ctx context.Context, in app.LoginInput) (app.TokenPair, error)
}

type sessionRefresh interface {
	Execute(ctx context.Context, in app.RefreshSessionInput) (app.TokenPair, error)
}

type userLogout interface {
	Execute(ctx context.Context, in app.LogoutInput) error
}

type UseCases struct {
	RegisterUser   userRegister
	Login          userLogin
	RefreshSession sessionRefresh
	Logout         userLogout
}

type Handler struct {
	identityv1.UnimplementedIdentityServiceServer
	useCases UseCases
	logger   *slog.Logger
}

func NewHandler(useCases UseCases, logger *slog.Logger) *Handler {
	return &Handler{useCases: useCases, logger: logger}
}

func (h *Handler) RegisterUser(ctx context.Context, req *identityv1.RegisterUserRequest) (*identityv1.RegisterUserResponse, error) {
	user, err := h.useCases.RegisterUser.Execute(ctx, app.RegisterUserInput{
		Email:       req.GetEmail(),
		PhoneNumber: req.GetPhoneNumber(),
		Password:    req.GetPassword(),
	})
	if err != nil {
		return nil, h.toStatus("RegisterUser", err)
	}
	return &identityv1.RegisterUserResponse{
		UserId: user.ID,
		Status: toProtoStatus(user.Status),
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	pair, err := h.useCases.Login.Execute(ctx, app.LoginInput{
		Identifier: req.GetIdentifier(),
		Password:   req.GetPassword(),
		DeviceID:   req.GetDeviceId(),
		IPAddress:  req.GetIpAddress(),
		UserAgent:  req.GetUserAgent(),
	})
	if err != nil {
		return nil, h.toStatus("Login", err)
	}
	return &identityv1.LoginResponse{Tokens: toProtoTokens(pair)}, nil
}

func (h *Handler) RefreshSession(ctx context.Context, req *identityv1.RefreshSessionRequest) (*identityv1.RefreshSessionResponse, error) {
	pair, err := h.useCases.RefreshSession.Execute(ctx, app.RefreshSessionInput{
		RefreshToken: req.GetRefreshToken(),
	})
	if err != nil {
		return nil, h.toStatus("RefreshSession", err)
	}
	return &identityv1.RefreshSessionResponse{Tokens: toProtoTokens(pair)}, nil
}

func (h *Handler) Logout(ctx context.Context, req *identityv1.LogoutRequest) (*identityv1.LogoutResponse, error) {
	err := h.useCases.Logout.Execute(ctx, app.LogoutInput{
		SessionID: req.GetSessionId(),
		UserID:    req.GetUserId(),
	})
	if err != nil {
		return nil, h.toStatus("Logout", err)
	}
	return &identityv1.LogoutResponse{}, nil
}

// toStatus maps a domain error to a gRPC status. Anything it does not
// recognise is logged in full and returned as a bare "internal error".
func (h *Handler) toStatus(method string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPhoneNumber),
		errors.Is(err, domain.ErrWeakPassword),
		errors.Is(err, domain.ErrMissingDeviceID):
		return status.Error(codes.InvalidArgument, rootMessage(err))

	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrPhoneNumberTaken):
		return status.Error(codes.AlreadyExists, rootMessage(err))

	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, domain.ErrInvalidCredentials.Error())

	// Both refresh failures give the caller the same answer. Only our
	// own logs say that a reuse was detected.
	case errors.Is(err, domain.ErrInvalidRefreshToken),
		errors.Is(err, domain.ErrRefreshTokenReuse):
		return status.Error(codes.Unauthenticated, domain.ErrInvalidRefreshToken.Error())

	case errors.Is(err, domain.ErrAccountLocked):
		return status.Error(codes.ResourceExhausted, domain.ErrAccountLocked.Error())

	case errors.Is(err, domain.ErrAccountDisabled):
		return status.Error(codes.PermissionDenied, domain.ErrAccountDisabled.Error())

	case errors.Is(err, domain.ErrSessionNotFound):
		return status.Error(codes.NotFound, domain.ErrSessionNotFound.Error())

	case errors.Is(err, domain.ErrLoginUnavailable):
		h.logger.Error("request failed", "method", method, "error", err)
		return status.Error(codes.Unavailable, domain.ErrLoginUnavailable.Error())

	default:
		h.logger.Error("request failed", "method", method, "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

func rootMessage(err error) string {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err.Error()
		}
		err = inner
	}
}

func toProtoTokens(pair app.TokenPair) *identityv1.SessionTokens {
	return &identityv1.SessionTokens{
		AccessToken:           pair.AccessToken,
		AccessTokenExpiresAt:  timestamppb.New(pair.AccessTokenExpiresAt),
		RefreshToken:          pair.RefreshToken,
		RefreshTokenExpiresAt: timestamppb.New(pair.RefreshTokenExpiresAt),
		SessionId:             pair.SessionID,
		UserId:                pair.UserID,
	}
}

func toProtoStatus(s domain.UserStatus) identityv1.UserStatus {
	switch s {
	case domain.UserStatusPendingVerification:
		return identityv1.UserStatus_USER_STATUS_PENDING_VERIFICATION
	case domain.UserStatusActive:
		return identityv1.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusSuspended:
		return identityv1.UserStatus_USER_STATUS_SUSPENDED
	case domain.UserStatusClosed:
		return identityv1.UserStatus_USER_STATUS_CLOSED
	default:
		return identityv1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}
