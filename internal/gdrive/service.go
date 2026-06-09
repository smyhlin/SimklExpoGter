package gdrive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SimklExpoGter/internal/config"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	DefaultFolderName = "SimklExpoGter Backups"
	authURL           = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL          = "https://oauth2.googleapis.com/token"
	folderMimeType    = "application/vnd.google-apps.folder"
)

type UploadedFile struct {
	ID          string
	Name        string
	WebViewLink string
}

type UploadResult struct {
	Token            config.OAuthToken
	FolderID         string
	FolderName       string
	FolderURL        string
	UploadFolderID   string
	UploadFolderName string
	UploadFolderURL  string
	Files            []UploadedFile
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) AuthURL(clientID, clientSecret, redirectURI, state, verifier string) (string, error) {
	oauthConfig, err := s.authConfig(clientID, clientSecret, redirectURI)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(state) == "" {
		return "", errors.New("missing Google Drive authorization state")
	}
	if strings.TrimSpace(verifier) == "" {
		return "", errors.New("missing Google Drive PKCE verifier")
	}

	return oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "consent"),
	), nil
}

func (s *Service) ExchangeCode(
	ctx context.Context,
	clientID, clientSecret, code, redirectURI, verifier string,
) (config.OAuthToken, error) {
	oauthConfig, err := s.authConfig(clientID, clientSecret, redirectURI)
	if err != nil {
		return config.OAuthToken{}, err
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return config.OAuthToken{}, errors.New("missing Google Drive authorization code")
	}
	if strings.TrimSpace(verifier) == "" {
		return config.OAuthToken{}, errors.New("missing Google Drive PKCE verifier")
	}

	token, err := oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return config.OAuthToken{}, err
	}

	return oauthTokenFromSDK(token), nil
}

func (s *Service) UploadFiles(
	ctx context.Context,
	settings config.GoogleDriveSettings,
	localPaths []string,
) (UploadResult, error) {
	driveService, token, err := s.authorizedDriveService(ctx, settings)
	if err != nil {
		return UploadResult{}, err
	}

	folderID, resolvedFolderName, err := s.ensureFolder(ctx, driveService, settings)
	if err != nil {
		return UploadResult{}, err
	}

	trimmedPaths := make([]string, 0, len(localPaths))
	for _, localPath := range localPaths {
		localPath = strings.TrimSpace(localPath)
		if localPath == "" {
			continue
		}
		trimmedPaths = append(trimmedPaths, localPath)
	}

	result := UploadResult{
		Token:      token,
		FolderID:   folderID,
		FolderName: resolvedFolderName,
		FolderURL:  folderURL(folderID),
		Files:      make([]UploadedFile, 0, len(trimmedPaths)),
	}

	if len(trimmedPaths) == 0 {
		return result, nil
	}

	uploadFolderID, uploadFolderName, err := s.createChildFolder(
		ctx,
		driveService,
		folderID,
		buildUploadFolderName(trimmedPaths, time.Now().UTC()),
	)
	if err != nil {
		return UploadResult{}, err
	}

	result.UploadFolderID = uploadFolderID
	result.UploadFolderName = uploadFolderName
	result.UploadFolderURL = folderURL(uploadFolderID)

	for _, localPath := range trimmedPaths {

		file, err := os.Open(localPath)
		if err != nil {
			return UploadResult{}, err
		}

		uploaded, err := driveService.Files.
			Create(&drive.File{
				Name:    filepath.Base(localPath),
				Parents: []string{uploadFolderID},
			}).
			Fields("id", "name", "webViewLink").
			Media(file).
			Do()
		_ = file.Close()
		if err != nil {
			return UploadResult{}, err
		}

		result.Files = append(result.Files, UploadedFile{
			ID:          uploaded.Id,
			Name:        uploaded.Name,
			WebViewLink: uploaded.WebViewLink,
		})
	}

	return result, nil
}

func (s *Service) authConfig(clientID, clientSecret, redirectURI string) (*oauth2.Config, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	redirectURI = strings.TrimSpace(redirectURI)

	switch {
	case clientID == "":
		return nil, errors.New("missing Google Drive client ID")
	case clientSecret == "":
		return nil, errors.New("missing Google Drive client secret")
	case redirectURI == "":
		return nil, errors.New("missing Google Drive redirect URI")
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       []string{drive.DriveFileScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}, nil
}

func (s *Service) authorizedDriveService(
	ctx context.Context,
	settings config.GoogleDriveSettings,
) (*drive.Service, config.OAuthToken, error) {
	token := sdkTokenFromOAuth(settings.Token)
	if token == nil {
		return nil, config.OAuthToken{}, errors.New("missing saved Google Drive token")
	}

	oauthConfig, err := s.authConfig(settings.ClientID, settings.ClientSecret, "http://127.0.0.1")
	if err != nil {
		return nil, config.OAuthToken{}, err
	}

	source := oauthConfig.TokenSource(ctx, token)
	refreshedToken, err := source.Token()
	if err != nil {
		return nil, config.OAuthToken{}, err
	}

	httpClient := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(refreshedToken, source))
	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, config.OAuthToken{}, err
	}

	return driveService, oauthTokenFromSDK(refreshedToken), nil
}

func (s *Service) ensureFolder(
	ctx context.Context,
	driveService *drive.Service,
	settings config.GoogleDriveSettings,
) (string, string, error) {
	resolvedFolderName := folderName(settings)
	folderID := strings.TrimSpace(settings.FolderID)

	if folderID != "" {
		folder, err := driveService.Files.Get(folderID).Fields("id", "name").Context(ctx).Do()
		if err == nil {
			return folder.Id, firstNonEmpty(folder.Name, resolvedFolderName), nil
		}
	}

	folder, err := driveService.Files.
		Create(&drive.File{
			Name:     resolvedFolderName,
			MimeType: folderMimeType,
		}).
		Fields("id", "name").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", err
	}

	return folder.Id, firstNonEmpty(folder.Name, resolvedFolderName), nil
}

func (s *Service) createChildFolder(
	ctx context.Context,
	driveService *drive.Service,
	parentFolderID string,
	name string,
) (string, string, error) {
	folder, err := driveService.Files.
		Create(&drive.File{
			Name:     name,
			MimeType: folderMimeType,
			Parents:  []string{parentFolderID},
		}).
		Fields("id", "name").
		Context(ctx).
		Do()
	if err != nil {
		return "", "", err
	}

	return folder.Id, firstNonEmpty(folder.Name, name), nil
}

func folderName(settings config.GoogleDriveSettings) string {
	return firstNonEmpty(settings.FolderName, DefaultFolderName)
}

func folderURL(folderID string) string {
	folderID = strings.TrimSpace(folderID)
	if folderID == "" {
		return ""
	}

	return "https://drive.google.com/drive/folders/" + folderID
}

func buildUploadFolderName(localPaths []string, now time.Time) string {
	if stamp := extractBackupStamp(localPaths); stamp != "" {
		return "backup-" + stamp
	}

	return "backup-" + now.UTC().Format("20060102-150405")
}

func extractBackupStamp(localPaths []string) string {
	for _, localPath := range localPaths {
		filename := strings.TrimSpace(filepath.Base(localPath))
		if filename == "" {
			continue
		}

		stem := strings.TrimSuffix(filename, filepath.Ext(filename))
		if len(stem) < len("20060102-150405") {
			continue
		}

		stamp := stem[len(stem)-len("20060102-150405"):]
		if isBackupStamp(stamp) {
			return stamp
		}
	}

	return ""
}

func isBackupStamp(value string) bool {
	if len(value) != len("20060102-150405") {
		return false
	}

	for index, char := range value {
		switch index {
		case 8:
			if char != '-' {
				return false
			}
		default:
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}

func oauthTokenFromSDK(token *oauth2.Token) config.OAuthToken {
	if token == nil {
		return config.OAuthToken{}
	}

	return config.OAuthToken{
		AccessToken:  strings.TrimSpace(token.AccessToken),
		TokenType:    strings.TrimSpace(token.TokenType),
		RefreshToken: strings.TrimSpace(token.RefreshToken),
		Expiry:       token.Expiry.UTC(),
	}
}

func sdkTokenFromOAuth(token config.OAuthToken) *oauth2.Token {
	if strings.TrimSpace(token.AccessToken) == "" && strings.TrimSpace(token.RefreshToken) == "" {
		return nil
	}

	return &oauth2.Token{
		AccessToken:  strings.TrimSpace(token.AccessToken),
		TokenType:    strings.TrimSpace(token.TokenType),
		RefreshToken: strings.TrimSpace(token.RefreshToken),
		Expiry:       token.Expiry,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
