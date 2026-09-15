// Package auth validates Cognito access tokens and provisions invited users.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

type UserProfile struct {
	Username string
	Email    string
}

// Authenticator performs the user-facing Cognito password flows. It deliberately
// has no persistence: passwords and temporary Cognito sessions only live for the
// duration of the HTTPS request/response exchange.
type Authenticator struct {
	client   *cognitoidentityprovider.Client
	clientID string
}

type SignInResult struct {
	AccessToken string
	Challenge   string
	Session     string
}

func NewAuthenticator(client *cognitoidentityprovider.Client, clientID string) *Authenticator {
	return &Authenticator{client: client, clientID: clientID}
}

func (a *Authenticator) SignIn(ctx context.Context, email, password string) (SignInResult, error) {
	out, err := a.client.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
		ClientId:       aws.String(a.clientID),
		AuthParameters: map[string]string{"USERNAME": email, "PASSWORD": password},
	})
	if err != nil {
		return SignInResult{}, err
	}
	result := SignInResult{Challenge: string(out.ChallengeName), Session: aws.ToString(out.Session)}
	if out.AuthenticationResult != nil {
		result.AccessToken = aws.ToString(out.AuthenticationResult.AccessToken)
	}
	return result, nil
}

func (a *Authenticator) CompleteNewPassword(ctx context.Context, email, password, session string) (SignInResult, error) {
	out, err := a.client.RespondToAuthChallenge(ctx, &cognitoidentityprovider.RespondToAuthChallengeInput{
		ClientId: aws.String(a.clientID), ChallengeName: types.ChallengeNameTypeNewPasswordRequired, Session: aws.String(session),
		ChallengeResponses: map[string]string{"USERNAME": email, "NEW_PASSWORD": password},
	})
	if err != nil {
		return SignInResult{}, err
	}
	result := SignInResult{Challenge: string(out.ChallengeName), Session: aws.ToString(out.Session)}
	if out.AuthenticationResult != nil {
		result.AccessToken = aws.ToString(out.AuthenticationResult.AccessToken)
	}
	return result, nil
}

func (a *Authenticator) RequestPasswordReset(ctx context.Context, email string) error {
	_, err := a.client.ForgotPassword(ctx, &cognitoidentityprovider.ForgotPasswordInput{ClientId: aws.String(a.clientID), Username: aws.String(email)})
	return err
}

func (a *Authenticator) ConfirmPasswordReset(ctx context.Context, email, code, password string) error {
	_, err := a.client.ConfirmForgotPassword(ctx, &cognitoidentityprovider.ConfirmForgotPasswordInput{ClientId: aws.String(a.clientID), Username: aws.String(email), ConfirmationCode: aws.String(code), Password: aws.String(password)})
	return err
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

// CreateWithPassword creates an email-sign-in account without sending an
// invitation, then makes the administrator-supplied password immediately usable.
func (i *Inviter) CreateWithPassword(ctx context.Context, email, password string) (string, error) {
	out, err := i.client.AdminCreateUser(ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:    aws.String(i.userPoolID),
		Username:      aws.String(email),
		MessageAction: types.MessageActionTypeSuppress,
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
		},
	})
	if err != nil {
		return "", err
	}
	if _, err := i.client.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
		UserPoolId: aws.String(i.userPoolID),
		Username:   aws.String(email),
		Password:   aws.String(password),
		Permanent:  true,
	}); err != nil {
		return "", err
	}
	for _, attribute := range out.User.Attributes {
		if aws.ToString(attribute.Name) == "sub" {
			return aws.ToString(attribute.Value), nil
		}
	}
	return "", errors.New("Cognito did not return a user subject")
}

// InviteOrFind creates an account when it is new, or returns the Cognito
// subject for an existing account. Bootstrap uses this so recreating an empty
// application database does not require deleting the administrator in Cognito.
func (i *Inviter) InviteOrFind(ctx context.Context, email string) (string, error) {
	userID, err := i.Invite(ctx, email)
	if err == nil {
		return userID, nil
	}
	var exists *types.UsernameExistsException
	if !errors.As(err, &exists) {
		return "", err
	}
	out, err := i.client.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(i.userPoolID),
		Username:   aws.String(email),
	})
	if err != nil {
		return "", err
	}
	for _, attribute := range out.UserAttributes {
		if aws.ToString(attribute.Name) == "sub" {
			return aws.ToString(attribute.Value), nil
		}
	}
	return "", errors.New("Cognito did not return a user subject")
}

// Email returns the email address for a stored Cognito subject. User access
// records retain the subject for authorization, but administrators should not
// need to see it in the application.
func (i *Inviter) Profile(ctx context.Context, userID string) (UserProfile, error) {
	out, err := i.client.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(i.userPoolID),
		Filter:     aws.String(fmt.Sprintf(`sub = "%s"`, userID)),
		Limit:      aws.Int32(1),
	})
	if err != nil {
		return UserProfile{}, err
	}
	if len(out.Users) == 0 {
		return UserProfile{}, errors.New("Cognito user not found")
	}
	profile := UserProfile{Username: aws.ToString(out.Users[0].Username)}
	for _, attribute := range out.Users[0].Attributes {
		if aws.ToString(attribute.Name) == "email" {
			profile.Email = aws.ToString(attribute.Value)
		}
	}
	return profile, nil
}

func (i *Inviter) Email(ctx context.Context, userID string) (string, error) {
	profile, err := i.Profile(ctx, userID)
	if err != nil {
		return "", err
	}
	if profile.Email == "" {
		return "", errors.New("Cognito user does not have an email address")
	}
	return profile.Email, nil
}

// Status reports whether a Cognito user is currently able to sign in.
func (i *Inviter) Status(ctx context.Context, username string) (string, error) {
	out, err := i.client.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(i.userPoolID),
		Username:   aws.String(username),
	})
	if err != nil {
		return "", err
	}
	if !out.Enabled {
		return "Disabled", nil
	}
	return "Active", nil
}

// Disable prevents a user from signing in. Their access assignment and
// attendance history stay intact so an administrator can retain the record.
func (i *Inviter) Disable(ctx context.Context, username string) error {
	_, err := i.client.AdminDisableUser(ctx, &cognitoidentityprovider.AdminDisableUserInput{
		UserPoolId: aws.String(i.userPoolID),
		Username:   aws.String(username),
	})
	return err
}
func Bearer(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
