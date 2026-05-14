package youtube

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	ytapi "google.golang.org/api/youtube/v3"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
)

var scopes = []string{
	ytapi.YoutubeUploadScope,
	ytapi.YoutubeReadonlyScope,
	ytapi.YoutubeScope,
}

func OAuthConfig(cfg *config.Config, account *db.Account) *oauth2.Config {
	clientID := cfg.GoogleClientID
	clientSecret := cfg.GoogleClientSecret

	if account != nil && account.ClientID != "" && account.ClientID != "app" {
		clientID = account.ClientID
		clientSecret = account.ClientSecret
	}

	return &oauth2.Config{
		ClientID:	clientID,
		ClientSecret:	clientSecret,
		Endpoint:	google.Endpoint,
		RedirectURL:	cfg.CallbackURL(),
		Scopes:		scopes,
	}
}

func AppOAuthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:	cfg.GoogleClientID,
		ClientSecret:	cfg.GoogleClientSecret,
		Endpoint:	google.Endpoint,
		RedirectURL:	cfg.CallbackURL(),
		Scopes:		scopes,
	}
}

func GetAuthURL(cfg *config.Config, account *db.Account, state string) string {
	oc := OAuthConfig(cfg, account)
	return oc.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func GetNewAccountAuthURL(cfg *config.Config, state string) string {
	oc := AppOAuthConfig(cfg)
	return oc.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func ExchangeCode(cfg *config.Config, account *db.Account, code string) error {
	oc := OAuthConfig(cfg, account)
	token, err := oc.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}
	channelID, channelName := FetchChannelInfo(oc, token)
	return db.UpsertToken(
		account.ID,
		token.AccessToken,
		token.RefreshToken,
		token.TokenType,
		token.Expiry,
		channelID,
		channelName,
	)
}

func ExchangeCodeNewAccount(cfg *config.Config, code string) (int64, error) {
	oc := AppOAuthConfig(cfg)
	token, err := oc.Exchange(context.Background(), code)
	if err != nil {
		return 0, fmt.Errorf("exchange code: %w", err)
	}

	channelID, channelName, chErr := FetchChannelInfoDetailed(oc, token)
	if chErr != nil {
		log.Printf("Warning: channel info fetch failed: %v (creating account anyway)", chErr)
	}

	accountID, err := db.InsertAccount("app", "")
	if err != nil {
		return 0, fmt.Errorf("create account: %w", err)
	}

	name := channelName
	if name == "" {
		name = fmt.Sprintf("Account #%d", accountID)
	}
	db.DB.Exec("UPDATE accounts SET name = ? WHERE id = ?", name, accountID)

	err = db.UpsertToken(
		accountID,
		token.AccessToken,
		token.RefreshToken,
		token.TokenType,
		token.Expiry,
		channelID,
		channelName,
	)
	if err != nil {
		return 0, fmt.Errorf("save token: %w", err)
	}
	return accountID, nil
}

func FetchChannelInfo(oc *oauth2.Config, token *oauth2.Token) (string, string) {
	id, name, _ := FetchChannelInfoDetailed(oc, token)
	return id, name
}

func FetchChannelInfoDetailed(oc *oauth2.Config, token *oauth2.Token) (string, string, error) {
	ctx := context.Background()
	client := oc.Client(ctx, token)
	svc, err := ytapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", "", fmt.Errorf("create youtube service: %w", err)
	}
	resp, err := svc.Channels.List([]string{"snippet"}).Mine(true).Do()
	if err != nil {
		return "", "", fmt.Errorf("channels.list API call: %w (ensure YouTube Data API v3 is enabled in Google Cloud Console)", err)
	}
	if len(resp.Items) == 0 {
		return "", "", fmt.Errorf("no YouTube channel found for this Google account")
	}
	ch := resp.Items[0]
	return ch.Id, ch.Snippet.Title, nil
}

func GetOAuth2Client(cfg *config.Config, account *db.Account, tok *db.Token) (*http.Client, error) {
	oc := OAuthConfig(cfg, account)
	oauthToken := &oauth2.Token{
		AccessToken:	tok.AccessToken,
		RefreshToken:	tok.RefreshToken,
		TokenType:	tok.TokenType,
		Expiry:		tok.Expiry,
	}
	src := oc.TokenSource(context.Background(), oauthToken)
	newToken, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token for account %d: %w", account.ID, err)
	}

	if newToken.AccessToken != tok.AccessToken {
		db.UpsertToken(account.ID, newToken.AccessToken, newToken.RefreshToken,
			newToken.TokenType, newToken.Expiry, tok.ChannelID, tok.ChannelName)
	}
	return oc.Client(context.Background(), newToken), nil
}

func AuthorizeCLI(cfg *config.Config, account *db.Account) error {
	oc := OAuthConfig(cfg, account)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprintf(w, "<h2>Error: no code received</h2>")
			return
		}
		codeCh <- code
		fmt.Fprintf(w, "<h2>Authorization successful! You can close this tab.</h2>")
	})

	cliPort := "8090"
	oc.RedirectURL = "http://localhost:" + cliPort + "/callback"

	listener, err := net.Listen("tcp", ":"+cliPort)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", cliPort, err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Close()

	authURL := oc.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("\nAccount #%d\nOpen this URL:\n\n  %s\n\n", account.ID, authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authorization timed out")
	}

	token, err := oc.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}
	channelID, channelName := FetchChannelInfo(oc, token)
	return db.UpsertToken(account.ID, token.AccessToken, token.RefreshToken,
		token.TokenType, token.Expiry, channelID, channelName)
}
