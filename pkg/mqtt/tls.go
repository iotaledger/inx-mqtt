package mqtt

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/mochi-co/mqtt/server/listeners"
)

func NewTLSSettings(tcpTlsCertificatePath string, tcpTlsPrivateKeyPath string) (*listeners.TLS, error) {

	if _, err := os.Stat(tcpTlsCertificatePath); err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return nil, fmt.Errorf("TCP TLS certificate file not found (%s)", tcpTlsCertificatePath)
		}

		return nil, fmt.Errorf("unable to check TCP TLS certificate file (%s): %w", tcpTlsCertificatePath, err)
	}

	if _, err := os.Stat(tcpTlsPrivateKeyPath); err != nil {
		if os.IsNotExist(err) {
			// file does not exist
			return nil, fmt.Errorf("TCP TLS private key file not found (%s)", tcpTlsPrivateKeyPath)
		}

		return nil, fmt.Errorf("unable to check TCP TLS private key file (%s): %w", tcpTlsPrivateKeyPath, err)
	}

	tcpTlsCertificate, err := os.ReadFile(tcpTlsCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read TCP TLS certificate: %w", err)
	}

	tcpTlsPrivateKey, err := os.ReadFile(tcpTlsPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read TCP TLS private key: %w", err)
	}

	if _, err := tls.X509KeyPair(tcpTlsCertificate, tcpTlsPrivateKey); err != nil {
		return nil, fmt.Errorf("loading TCP TLS configuration failed: %w", err)
	}

	return &listeners.TLS{
		Certificate: tcpTlsCertificate,
		PrivateKey:  tcpTlsPrivateKey,
	}, nil
}
