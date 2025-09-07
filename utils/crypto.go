package utils

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"os"

	"github.com/psyb0t/ctxerrors"
)

func CreateClientCertAndCertPoolFromFiles(
	clientCertPath string,
	clientKeyPath string,
	caCertPath string,
) (tls.Certificate, *x509.CertPool, error) {
	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to load client certificate")
	}

	caCertPool := x509.NewCertPool()

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to read CA certificate")
	}

	if !caCertPool.AppendCertsFromPEM(caCert) {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to append CA certificate to pool")
	}

	return clientCert, caCertPool, nil
}

func CreateClientCertAndCertPoolFromBase64(
	clientCertBase64 string,
	clientKeyBase64 string,
	caCertBase64 string,
) (tls.Certificate, *x509.CertPool, error) {
	clientCertBytes, err := base64.StdEncoding.DecodeString(clientCertBase64)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to decode base64 client certificate")
	}

	clientKeyBytes, err := base64.StdEncoding.DecodeString(clientKeyBase64)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to decode base64 client key")
	}

	clientCert, err := tls.X509KeyPair(clientCertBytes, clientKeyBytes)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to load client certificate from bytes")
	}

	caCertPool := x509.NewCertPool()

	caCertBytes, err := base64.StdEncoding.DecodeString(caCertBase64)
	if err != nil {
		return tls.Certificate{}, nil, ctxerrors.Wrap(err, "failed to decode base64 CA certificate")
	}

	if !caCertPool.AppendCertsFromPEM(caCertBytes) {
		return tls.Certificate{}, nil, ctxerrors.New("failed to append CA certificate to pool")
	}

	return clientCert, caCertPool, nil
}
