package service

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/google/uuid"
)

// SessionContext carries per-request metadata attached to issued sessions so
// users can recognise (and revoke) them later.
type SessionContext struct {
	UserAgent string
	IPAddress string
}

// defaultRefreshTokenTTL is how long a refresh token lives when
// WithRefreshTokenTTL is not supplied.
const defaultRefreshTokenTTL = 30 * 24 * time.Hour

// twoFactorChallengeTTL bounds how long a 2FA challenge stays answerable.
const twoFactorChallengeTTL = 5 * time.Minute

var errInvalidRefreshToken = errors.New("invalid refresh token")

// LoginWithSession authenticates like Login but issues a refresh-token-backed
// session, and gates fully-enrolled 2FA users behind a challenge: they get a
// short-lived challenge token instead of credentials until they answer with a
// TOTP or backup code via VerifyTwoFactorChallenge.
func (s *authService) LoginWithSession(ctx context.Context, req *models.LoginRequest, sctx SessionContext) (*models.LoginResponse, error) {
	user, err := s.authenticate(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2FA gate: no access or refresh token leaves this function until the
	// second factor is presented.
	if s.twoFactor != nil {
		enabled, err := s.twoFactor.IsEnabled(ctx, user.ID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to check 2FA enrollment")
			return nil, errors.New("failed to process login")
		}
		if enabled {
			challenge, err := issueTwoFactorChallenge(s.tokenConfig.SecretKey, user.ID, user.TenantID, twoFactorChallengeTTL)
			if err != nil {
				s.logger.WithError(err).Error("Failed to issue 2FA challenge")
				return nil, errors.New("failed to process login")
			}
			return &models.LoginResponse{
				TwoFactor: &models.TwoFactorChallenge{
					Required:       true,
					ChallengeToken: challenge,
				},
			}, nil
		}
	}

	resp, err := s.issueSession(ctx, user, sctx)
	if err != nil {
		return nil, err
	}
	s.finishLogin(ctx, user, user.Email, sctx.IPAddress, sctx.UserAgent)
	return resp, nil
}

// VerifyTwoFactorChallenge completes a 2FA-gated login: it validates the
// challenge token issued by LoginWithSession, verifies the TOTP/backup code,
// and only then issues the session.
func (s *authService) VerifyTwoFactorChallenge(ctx context.Context, req *models.TwoFactorVerifyRequest, sctx SessionContext) (*models.LoginResponse, error) {
	claims, err := parseTwoFactorChallenge(s.tokenConfig.SecretKey, req.ChallengeToken)
	if err != nil {
		return nil, err
	}

	if s.twoFactor == nil {
		return nil, errors.New("two-factor authentication is not enabled")
	}
	ok, err := s.twoFactor.Verify(ctx, claims.UserID, req.Code)
	if err != nil || !ok {
		s.logger.WithField("user_id", claims.UserID).Warn("2FA challenge failed")
		s.publishLoginFailed(ctx, claims.TenantID, claims.UserID, "", "invalid_2fa_code", sctx.IPAddress, sctx.UserAgent)
		return nil, errors.New("invalid 2fa code")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to load user for 2FA completion")
		return nil, errors.New("failed to process login")
	}
	if user.Status != models.UserStatusActive {
		return nil, errors.New("user account is not active")
	}

	resp, err := s.issueSession(ctx, user, sctx)
	if err != nil {
		return nil, err
	}
	s.finishLogin(ctx, user, user.Email, sctx.IPAddress, sctx.UserAgent)
	return resp, nil
}

// RefreshAccessToken rotates a refresh token: the presented token is revoked
// and a new access + refresh pair is issued. Presenting an already-revoked
// token is treated as theft — the whole session family for that user is
// revoked (reuse detection).
func (s *authService) RefreshAccessToken(ctx context.Context, req *models.RefreshTokenRequest, sctx SessionContext) (*models.LoginResponse, error) {
	if s.refreshTokenRepo == nil {
		return nil, errors.New("refresh tokens are not enabled")
	}

	rt, err := s.refreshTokenRepo.GetByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, errInvalidRefreshToken
	}

	// Reuse detection: a revoked token being replayed means the original or
	// the rotated copy leaked. Kill every session for the user.
	if rt.RevokedAt != nil {
		s.logger.WithField("user_id", rt.UserID).Warn("Revoked refresh token replayed — revoking all sessions")
		if err := s.refreshTokenRepo.RevokeAllByUser(ctx, rt.UserID); err != nil {
			s.logger.WithError(err).Error("Failed to revoke sessions after refresh-token reuse")
		}
		s.publishEvent(ctx, "RefreshTokenReuseDetected", map[string]interface{}{
			"tenant_id":  rt.TenantID,
			"user_id":    rt.UserID,
			"ip_address": sctx.IPAddress,
			"user_agent": sctx.UserAgent,
		})
		return nil, errInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, errInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, errInvalidRefreshToken
	}
	if user.Status != models.UserStatusActive {
		return nil, errInvalidRefreshToken
	}

	// Rotate: revoke the presented token before issuing its replacement.
	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		s.logger.WithError(err).Error("Failed to revoke rotated refresh token")
		return nil, errInvalidRefreshToken
	}

	return s.issueSession(ctx, user, sctx)
}

// Logout revokes the presented refresh token. Unknown or already-revoked
// tokens are swallowed — logout is idempotent and must not enable token
// enumeration.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if s.refreshTokenRepo == nil || refreshToken == "" {
		return nil
	}
	if err := s.refreshTokenRepo.RevokeByToken(ctx, refreshToken); err != nil {
		s.logger.WithError(err).Debug("Logout with unknown or already-revoked refresh token")
	}
	return nil
}

// LogoutAll revokes every live session for the user.
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	if s.refreshTokenRepo == nil {
		return nil
	}
	return s.refreshTokenRepo.RevokeAllByUser(ctx, userID)
}

// ListSessions returns the user's live sessions. The session matching
// currentRefreshToken is flagged so clients can label "this device".
func (s *authService) ListSessions(ctx context.Context, userID, currentRefreshToken string) ([]models.SessionResponse, error) {
	if s.refreshTokenRepo == nil {
		return nil, errors.New("refresh tokens are not enabled")
	}
	tokens, err := s.refreshTokenRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessions := make([]models.SessionResponse, len(tokens))
	for i, t := range tokens {
		sessions[i] = models.SessionResponse{
			ID:         t.ID,
			UserAgent:  t.UserAgent,
			IPAddress:  t.IPAddress,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
			Current:    currentRefreshToken != "" && t.Token == currentRefreshToken,
		}
	}
	return sessions, nil
}

// RevokeSession revokes a single session by id, but only when it belongs to
// the calling user. Foreign sessions are reported as not found.
func (s *authService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if s.refreshTokenRepo == nil {
		return errors.New("refresh tokens are not enabled")
	}
	rt, err := s.refreshTokenRepo.GetByID(ctx, sessionID)
	if err != nil || rt.UserID != userID {
		return errors.New("session not found")
	}
	return s.refreshTokenRepo.Revoke(ctx, rt.ID)
}

// VerifyUserPassword checks a user's password. Used by flows that require
// re-authentication with the first factor, such as disabling 2FA.
func (s *authService) VerifyUserPassword(ctx context.Context, userID, password string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}
	if !verifyPassword(user.PasswordHash, password) {
		return errors.New("incorrect password")
	}
	return nil
}

// issueSession creates the access token and, when session storage is
// configured, a refresh token row carrying the device metadata.
func (s *authService) issueSession(ctx context.Context, user *models.User, sctx SessionContext) (*models.LoginResponse, error) {
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate token")
		return nil, errors.New("failed to generate authentication token")
	}

	resp := &models.LoginResponse{
		User:      user.ToResponse(),
		Token:     token,
		ExpiresAt: expiresAt,
	}

	if s.refreshTokenRepo != nil {
		refreshStr, err := generateSecureToken()
		if err != nil {
			s.logger.WithError(err).Error("Failed to generate refresh token")
			return nil, errors.New("failed to generate authentication token")
		}
		ttl := s.refreshTokenTTL
		if ttl <= 0 {
			ttl = defaultRefreshTokenTTL
		}
		now := time.Now()
		rt := &models.RefreshToken{
			ID:         uuid.New().String(),
			UserID:     user.ID,
			TenantID:   user.TenantID,
			Token:      refreshStr,
			UserAgent:  sctx.UserAgent,
			IPAddress:  sctx.IPAddress,
			ExpiresAt:  now.Add(ttl),
			LastUsedAt: now,
		}
		if err := s.refreshTokenRepo.Create(ctx, rt); err != nil {
			s.logger.WithError(err).Error("Failed to persist refresh token")
			return nil, errors.New("failed to generate authentication token")
		}
		resp.RefreshToken = refreshStr
	}

	return resp, nil
}

// revokeAllSessions is the best-effort hook called on password change/reset
// and 2FA state changes. No-op when session storage is not configured.
func (s *authService) revokeAllSessions(ctx context.Context, userID string) {
	if s.refreshTokenRepo == nil {
		return
	}
	if err := s.refreshTokenRepo.RevokeAllByUser(ctx, userID); err != nil {
		s.logger.WithError(err).Error("Failed to revoke user sessions")
	}
}
