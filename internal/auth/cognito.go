// Package auth validates Cognito access tokens and provisions invited users.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
)

type Config struct{ Region, UserPoolID, ClientID string }
type Principal struct{ Subject, Email string }
type contextKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(contextKey{}).(Principal)
	return p, ok
}

type Verifier struct {
	issuer, clientID, keysURL string
	client                    *http.Client
	mu                        sync.RWMutex
	keys                      map[string]*rsa.PublicKey
	expires                   time.Time
}

func NewVerifier(c Config) (*Verifier, error) {
	if c.Region == "" || c.UserPoolID == "" || c.ClientID == "" {
		return nil, errors.New("COGNITO_REGION, COGNITO_USER_POOL_ID, and COGNITO_CLIENT_ID are required")
	}
	issuer := "https://cognito-idp." + c.Region + ".amazonaws.com/" + c.UserPoolID
	return &Verifier{issuer: issuer, clientID: c.ClientID, keysURL: issuer + "/.well-known/jwks.json", client: &http.Client{Timeout: 5 * time.Second}}, nil
}
func (v *Verifier) Verify(ctx context.Context, tokenString string) (Principal, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != "RS256" {
			return nil, errors.New("unexpected signing algorithm")
		}
		kid, _ := t.Header["kid"].(string)
		return v.key(ctx, kid)
	})
	if err != nil || !token.Valid {
		return Principal{}, errors.New("invalid access token")
	}
	if claims["iss"] != v.issuer || claims["token_use"] != "access" || claims["client_id"] != v.clientID {
		return Principal{}, errors.New("invalid access token claims")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return Principal{}, errors.New("access token has no subject")
	}
	email, _ := claims["email"].(string)
	return Principal{Subject: sub, Email: email}, nil
}
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Now().Before(v.expires)
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, errors.New("unknown signing key")
	}
	return key, nil
}
func (v *Verifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("could not retrieve signing keys")
	}
	var body struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err = json.NewDecoder(res.Body).Decode(&body); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range body.Keys {
		if j.Kty != "RSA" {
			continue
		}
		n, e1 := base64.RawURLEncoding.DecodeString(j.N)
		e, e2 := base64.RawURLEncoding.DecodeString(j.E)
		if e1 != nil || e2 != nil {
			continue
		}
		exponent := 0
		for _, b := range e {
			exponent = exponent<<8 | int(b)
		}
		keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
	}
	v.mu.Lock()
	v.keys = keys
	v.expires = time.Now().Add(time.Hour)
	v.mu.Unlock()
	return nil
}

type Inviter struct {
	client     *cognitoidentityprovider.Client
	userPoolID string
}

func NewInviter(client *cognitoidentityprovider.Client, userPoolID string) *Inviter {
	return &Inviter{client: client, userPoolID: userPoolID}
}
func (i *Inviter) Invite(ctx context.Context, email string) (string, error) {
	out, err := i.client.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{UserPoolId: aws.String(i.userPoolID), Username: aws.String(email), DesiredDeliveryMediums: []types.DeliveryMediumType{types.DeliveryMediumTypeEmail}, UserAttributes: []types.AttributeType{{Name: aws.String("email"), Value: aws.String(email)}, {Name: aws.String("email_verified"), Value: aws.String("true")}}})
	if err != nil {
		return "", err
	}
	for _, attribute := range out.User.Attributes {
		if aws.ToString(attribute.Name) == "sub" {
			return aws.ToString(attribute.Value), nil
		}
	}
	return "", errors.New("Cognito did not return a user subject")
}
func Bearer(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
