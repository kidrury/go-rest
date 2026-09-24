package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	secret         []byte
	issuer         string
	audience       string
	accessTokenTTL time.Duration
}

const (
	AccessTokenType = "access+jwt"
	AccessTokenAlg  = "HS256"
)

var ErrInvalidToken = errors.New("invalid access token")

type AccessClaims struct {
	jwt.RegisteredClaims
}

func NewTokenManager(
	secret, issuer, audience string, accessTokenTTL time.Duration,
) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("secret length can't be less than 32 characters")
	}
	if issuer == "" {
		return nil, errors.New("issuer is required")
	}

	if audience == "" {
		return nil, errors.New("audience is required")
	}

	if accessTokenTTL <= 0 {
		return nil, errors.New("access token TTL must be greater than zero")
	}

	return &TokenManager{
		secret:         []byte(secret),
		issuer:         issuer,
		audience:       audience,
		accessTokenTTL: accessTokenTTL,
	}, nil
}

func (tm *TokenManager) IssueAccessToken(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("userID is required for issuing access tokens")
	}

	expiresAt := time.Now().Add(tm.accessTokenTTL)

	claim := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    tm.issuer,
			Audience:  jwt.ClaimStrings{tm.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	token.Header["typ"] = AccessTokenType

	signedToken, err := token.SignedString(tm.secret)

	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (tm *TokenManager) VerifyAccessToken(tokenString string) (*AccessClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	claims := &AccessClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != AccessTokenAlg {
			return nil, ErrInvalidToken
		}

		if t.Header["typ"] != AccessTokenType {
			return nil, ErrInvalidToken
		}

		return tm.secret, nil
	},
		jwt.WithValidMethods([]string{AccessTokenAlg}),
		jwt.WithIssuer(tm.issuer),
		jwt.WithAudience(tm.audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Subject == "" {
		return nil, ErrInvalidToken
	}

	if claims.ID == "" {
		return nil, ErrInvalidToken
	}

	if claims.IssuedAt == nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
