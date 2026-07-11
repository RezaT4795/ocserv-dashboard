package cisco

import (
	"net/url"
	"strings"
	"time"

	ocservUser "github.com/mmtaee/ocserv-dashboard/common/ocserv/user"
)

const CertificateTokenTTL = 10 * time.Minute

const certificateDownloadPath = "/api/customers/setup/cisco/certificate/"

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
	ConnectionCreateURI  string
	CertificatePassword  string
	ConnectionName       string
	ServerAddress        string
	ServerPort           int
	ExpiresAt            time.Time
}

func BuildSetup(input SetupInput) (Setup, error) {
	connectionName, err := ocservUser.NormalizeProfileConnectionName(input.ConnectionName)
	if err != nil {
		return Setup{}, err
	}

	serverAddress, err := ocservUser.NormalizeProfileServerAddress(input.ServerAddress)
	if err != nil {
		return Setup{}, err
	}

	serverPort, err := ocservUser.NormalizeProfileServerPort(input.ServerPort)
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

	publicAPIBaseURL := strings.TrimRight(input.PublicAPIBaseURL, "/")
	certificateURL := publicAPIBaseURL +
		certificateDownloadPath +
		url.PathEscape(token)

	certificateImportURI, err := ocservUser.BuildAnyConnectImportURI(certificateURL)
	if err != nil {
		return Setup{}, err
	}

	connectionCreateURI, err := ocservUser.BuildAnyConnectCreateURI(
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
		ConnectionCreateURI:  connectionCreateURI,
		CertificatePassword:  input.CertificatePassword,
		ConnectionName:       connectionName,
		ServerAddress:        serverAddress,
		ServerPort:           serverPort,
		ExpiresAt:            expiresAt,
	}, nil
}
