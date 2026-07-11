package cisco

import (
	"net/url"
	"strings"
	"time"

	ocservUser "github.com/mmtaee/ocserv-dashboard/common/ocserv/user"
)

const CertificateTokenTTL = 10 * time.Minute

const (
	certificateDownloadPath     = "/api/customers/setup/cisco/certificate/"
	certificateImportLaunchPath = "/api/customers/setup/cisco/launch/certificate/"
	connectionCreateLaunchPath  = "/api/customers/setup/cisco/launch/connection/"
)

type SetupInput struct {
	Username            string
	CertificatePassword string
	ConnectionName      string
	ServerAddress       string
	ServerPort          int
	PublicAPIBaseURL    string
	SecretKey           string
	Now                 time.Time
}

type Setup struct {
	CertificateImportURI string
	CertificateImportURL string
	ConnectionCreateURI  string
	ConnectionCreateURL  string
	CertificatePassword  string
	ConnectionName       string
	ServerAddress        string
	ServerPort           int
	ExpiresAt            time.Time
}

func BuildSetup(input SetupInput) (Setup, error) {
	connectionName, err := ocservUser.NormalizeProfileConnectionName(
		input.ConnectionName,
	)
	if err != nil {
		return Setup{}, err
	}

	serverAddress, err := ocservUser.NormalizeProfileServerAddress(
		input.ServerAddress,
	)
	if err != nil {
		return Setup{}, err
	}

	serverPort, err := ocservUser.NormalizeProfileServerPort(
		input.ServerPort,
	)
	if err != nil {
		return Setup{}, err
	}

	expiresAt := input.Now.Add(CertificateTokenTTL)

	token, err := CreateCertificateToken(
		input.Username,
		expiresAt,
		input.SecretKey,
	)
	if err != nil {
		return Setup{}, err
	}

	certificateImportURI, err := BuildCertificateImportURI(
		input.PublicAPIBaseURL,
		token,
	)
	if err != nil {
		return Setup{}, err
	}

	connectionCreateURI, err := BuildConnectionCreateURI(
		connectionName,
		serverAddress,
		serverPort,
		input.Username,
	)
	if err != nil {
		return Setup{}, err
	}

	return Setup{
		CertificateImportURI: certificateImportURI,
		CertificateImportURL: BuildCertificateImportLaunchURL(
			input.PublicAPIBaseURL,
			token,
		),
		ConnectionCreateURI: connectionCreateURI,
		ConnectionCreateURL: BuildConnectionCreateLaunchURL(
			input.PublicAPIBaseURL,
			token,
		),
		CertificatePassword: input.CertificatePassword,
		ConnectionName:      connectionName,
		ServerAddress:       serverAddress,
		ServerPort:          serverPort,
		ExpiresAt:           expiresAt,
	}, nil
}

func BuildCertificateImportURI(
	publicAPIBaseURL string,
	token string,
) (string, error) {
	certificateURL := buildPublicSetupURL(
		publicAPIBaseURL,
		certificateDownloadPath,
		token,
	)

	return ocservUser.BuildAnyConnectImportURI(certificateURL)
}

func BuildConnectionCreateURI(
	connectionName string,
	serverAddress string,
	serverPort int,
	username string,
) (string, error) {
	return ocservUser.BuildAnyConnectCreateURI(
		connectionName,
		serverAddress,
		serverPort,
		username,
	)
}

func BuildCertificateImportLaunchURL(
	publicAPIBaseURL string,
	token string,
) string {
	return buildPublicSetupURL(
		publicAPIBaseURL,
		certificateImportLaunchPath,
		token,
	)
}

func BuildConnectionCreateLaunchURL(
	publicAPIBaseURL string,
	token string,
) string {
	return buildPublicSetupURL(
		publicAPIBaseURL,
		connectionCreateLaunchPath,
		token,
	)
}

func buildPublicSetupURL(
	publicAPIBaseURL string,
	path string,
	token string,
) string {
	return strings.TrimRight(publicAPIBaseURL, "/") +
		path +
		url.PathEscape(token)
}
