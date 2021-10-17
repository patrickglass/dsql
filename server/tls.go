package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"

	"github.com/rs/zerolog/log"
)

func GenKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate private key")
	}
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	notBefore := time.Now()
	validFor := 365 * 24 * time.Hour
	notAfter := notBefore.Add(validFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate serial number")
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Development"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		IsCA: true,

		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create certificate")
	}
	return ed25519.PublicKey(derBytes), priv
}
