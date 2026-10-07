package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/moomerman/zap/internal/homedir"
)

// CACert is the self-signed root certificate
var CACert *tls.Certificate

// CreateCACert creates and returns a new CA certificate key pair
func CreateCACert(caName string) ([]byte, []byte, error) {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generating new RSA key: %w", err)
	}

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(9999 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("generating serial number: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"github.com/moomerman/zap/cert"},
			CommonName:   caName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, priv.Public(), priv)
	if err != nil {
		return nil, nil, fmt.Errorf("creating CA cert: %w", err)
	}

	encodedKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	encodedCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return encodedKey, encodedCert, nil
}

func writeCACert(path string, key, cert []byte) error {
	dir, err := homedir.Expand(path)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	keyPath := filepath.Join(dir, "key.pem")
	certPath := filepath.Join(dir, "cert.pem")

	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		log.Println("[cert] private key exists, skipping install", keyPath)
		return nil
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("writing cert.pem: %w", err)
	}
	certOut.Write(cert)
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("writing key.pem: %w", err)
	}
	keyOut.Write(key)
	keyOut.Close()

	return nil
}

// Cache holds the certificates issued for each host
type Cache struct {
	lock  sync.Mutex
	certs map[string]*tls.Certificate
}

// NewCache holds the dynamically generated host certificates
func NewCache() (*Cache, error) {
	err := loadCertLegacy()
	if err != nil {
		return nil, fmt.Errorf("couldn't load root certificate: %w", err)
	}

	return &Cache{certs: make(map[string]*tls.Certificate)}, nil
}

// GetCertificate implements the required function for tls config
func (c *Cache) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	name := clientHello.ServerName

	if cert, ok := c.certs[name]; ok {
		return cert, nil
	}

	cert, err := IssueCert(CACert, name, nil)
	if err != nil {
		return nil, err
	}

	c.certs[name] = cert

	return cert, nil
}

// IssueCert generates a signed Key/Cert pair for the given CACert with the given name
func IssueCert(parent *tls.Certificate, commonName string, ipAddress net.IP) (*tls.Certificate, error) {

	// start by generating private key
	// privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"github.com/moomerman/zap/cert"},
			CommonName:   commonName,
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	cert.DNSNames = append(cert.DNSNames, commonName)
	if ipAddress != nil {
		cert.IPAddresses = append(cert.IPAddresses, ipAddress)
	}

	x509parent, err := x509.ParseCertificate(parent.Certificate[0])
	if err != nil {
		return nil, err
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, cert, x509parent, privKey.Public(), parent.PrivateKey)

	if err != nil {
		return nil, fmt.Errorf("could not create certificate: %w", err)
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privKey,
		Leaf:        cert,
	}

	return tlsCert, nil
}

// EncodeCert is a helper to encode the given certificate
func EncodeCert(cert *tls.Certificate) ([]byte, []byte, error) {
	encodedCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyBytes := x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))
	encodedKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	return encodedKey, encodedCert, nil
}

// LoadCACert loads a certificate key pair into memory
func LoadCACert(rootDir string) (*tls.Certificate, error) {
	dir, err := homedir.Expand(rootDir)
	if err != nil {
		return nil, err
	}

	keyPath := filepath.Join(dir, "key.pem")
	certPath := filepath.Join(dir, "cert.pem")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	CACert = &cert
	return &cert, nil
}

// these two legacy functions are here for backward compatibility and should
// eventually be removed (for zap)

// CreateCertLegacy creates a new self-signed root certificate
func CreateCertLegacy() error {
	key, cert, err := CreateCACert("zap CA")
	if err != nil {
		return err
	}
	return writeCACert(supportDir, key, cert)
	// return InstallCert(filepath.Join(supportDir, "cert.pem"))
}

func loadCertLegacy() error {
	_, err := LoadCACert(supportDir)
	return err
}
