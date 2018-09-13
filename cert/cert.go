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

	"github.com/hashicorp/golang-lru"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

// CACert is the self-signed root certificate
var CACert *tls.Certificate

// CreateCACert creates and returns a new CA certificate key pair
func CreateCACert(caName string) ([]byte, []byte, error) {

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, errors.Context(err, "generating new RSA key")
	}

	// create certificate structure with proper values
	notBefore := time.Now()
	notAfter := notBefore.Add(9999 * 24 * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Context(err, "generating serial number")
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
		return nil, nil, errors.Context(err, "creating CA cert")
	}

	encodedKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	encodedCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return encodedKey, encodedCert, nil
}

func writeCACert(path string, key, cert []byte) error {
	dir := homedir.MustExpand(path)

	err := os.MkdirAll(dir, 0700)
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
		return errors.Context(err, "writing cert.pem")
	}
	certOut.Write(cert)
	certOut.Close()

	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Context(err, "writing key.pem")
	}
	keyOut.Write(key)
	keyOut.Close()

	return nil
}

// Cache is a struct to hold the dynamic certificates and a lock
type Cache struct {
	lock  sync.Mutex
	cache *lru.ARCCache
}

// NewCache holds the dynamically generated host certificates
func NewCache() (*Cache, error) {
	err := loadCertLegacy()
	if err != nil {
		return nil, errors.Context(err, "couldn't load root certificate")
	}

	cache, err := lru.NewARC(1024)
	if err != nil {
		return nil, errors.Context(err, "couldn't create a new cache")
	}

	return &Cache{cache: cache}, nil
}

// GetCertificate implements the required function for tls config
func (c *Cache) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	name := clientHello.ServerName

	if val, ok := c.cache.Get(name); ok {
		return val.(*tls.Certificate), nil
	}

	cert, err := IssueCert(CACert, name, nil)
	if err != nil {
		return nil, err
	}

	c.cache.Add(name, cert)

	return cert, nil
}

// IssueCert generates a signed Key/Cert pair for the given CACert with the given name
func IssueCert(parent *tls.Certificate, commonName string, ipAddress net.IP) (*tls.Certificate, error) {

	// start by generating private key
	// privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
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
		return nil, fmt.Errorf("could not create certificate: %v", err)
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
	dir := homedir.MustExpand(rootDir)

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
	if err := writeCACert(supportDir, key, cert); err != nil {
		return err
	}
	return InstallCert(filepath.Join(supportDir, "cert.pem"))
}

func loadCertLegacy() error {
	_, err := LoadCACert(supportDir)
	return err
}
