package dev

// based on https://github.com/puma/puma-dev/blob/master/dev/ssl.go

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

const supportDir = "~/Library/Application Support/com.github.moomerman.phx-dev"

// CACert is the self-signed root certificate
var CACert *tls.Certificate

// LoadCert loads the self-signed certificate
func LoadCert() error {
	dir := homedir.MustExpand(supportDir)

	keyPath := filepath.Join(dir, "key.pem")
	certPath := filepath.Join(dir, "cert.pem")

	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}

	CACert = &tlsCert
	return nil
}

// CreateCert creates a new self-signed root certificate
func CreateCert() error {
	dir := homedir.MustExpand(supportDir)

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	keyPath := filepath.Join(dir, "key.pem")
	certPath := filepath.Join(dir, "cert.pem")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return errors.Context(err, "generating new RSA key")
	}

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(9999 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return errors.Context(err, "generating serial number")
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Developer Certificate"},
			CommonName:   "phx-dev CA",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, cert, cert, priv.Public(), priv)

	if err != nil {
		return errors.Context(err, "creating CA cert")
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return errors.Context(err, "writing cert.pem")
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Context(err, "writing key.pem")
	}

	pem.Encode(
		keyOut,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	keyOut.Close()

	return installCert(certPath)
}

// CertCache is a struct to hold the dynamic certificates and a lock
type CertCache struct {
	lock  sync.Mutex
	cache *lru.ARCCache
}

// NewCertCache holds the dynamically generated host certificates
func NewCertCache() *CertCache {
	cache, err := lru.NewARC(1024)
	if err != nil {
		panic(err)
	}

	return &CertCache{
		cache: cache,
	}
}

// GetCertificate implements the required function for tls config
func (c *CertCache) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	name := clientHello.ServerName

	if val, ok := c.cache.Get(name); ok {
		return val.(*tls.Certificate), nil
	}

	cert, err := makeCert(CACert, name)
	if err != nil {
		return nil, err
	}

	c.cache.Add(name, cert)

	return cert, nil
}

func makeCert(parent *tls.Certificate, name string) (*tls.Certificate, error) {

	// start by generating private key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"phx-dev Signed"},
			CommonName:   name,
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	cert.DNSNames = append(cert.DNSNames, name)

	x509parent, err := x509.ParseCertificate(parent.Certificate[0])
	if err != nil {
		return nil, err
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, cert, x509parent, privKey.Public(), parent.PrivateKey)

	if err != nil {
		return nil, fmt.Errorf("could not create certificate: %v", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privKey,
		Leaf:        cert,
	}

	return tlsCert, nil
}

func installCert(cert string) error {
	fmt.Printf("* Adding certification to system keychain as trusted\n")
	fmt.Printf("! There is probably a dialog open that you must type your password into\n")

	keychain := "/Library/Keychains/System.keychain"

	command := "do shell script \"security add-trusted-cert -d -r trustRoot -k '" + keychain + "' '" + cert + "'\" with administrator privileges"
	cmd := exec.Command("osascript", "-e", command)
	err := cmd.Run()

	if err != nil {
		return err
	}

	fmt.Printf("* Certificates setup complete\n")

	return nil
}
