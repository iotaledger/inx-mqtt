package broker

import (
	"crypto/tls"
	"os"

	"github.com/iotaledger/hive.go/ierrors"
)

func NewTLSConfig(tcpTLSCertificatePath string, tcpTLSPrivateKeyPath string) (*tls.Config, error) {

	if _, err := os.Stat(tcpTLSCertificatePath); err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return nil, ierrors.Errorf("TCP TLS certificate file not found (%s)", tcpTLSCertificatePath)
		}

		return nil, ierrors.Errorf("unable to check TCP TLS certificate file (%s): %w", tcpTLSCertificatePath, err)
	}

	if _, err := os.Stat(tcpTLSPrivateKeyPath); err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return nil, ierrors.Errorf("TCP TLS private key file not found (%s)", tcpTLSPrivateKeyPath)
		}

		return nil, ierrors.Errorf("unable to check TCP TLS private key file (%s): %w", tcpTLSPrivateKeyPath, err)
	}

	tcpTLSCertificate, err := os.ReadFile(tcpTLSCertificatePath)
	if err != nil {
		return nil, ierrors.Errorf("unable to read TCP TLS certificate: %w", err)
	}

	tcpTLSPrivateKey, err := os.ReadFile(tcpTLSPrivateKeyPath)
	if err != nil {
		return nil, ierrors.Errorf("unable to read TCP TLS private key: %w", err)
	}

	cert, err := tls.X509KeyPair(tcpTLSCertificate, tcpTLSPrivateKey)
	if err != nil {
		return nil, ierrors.Errorf("loading TCP TLS configuration failed: %w", err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}, nil
}
