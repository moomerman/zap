package cert_test

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/moomerman/zap/cert"
)

func TestCACertificateIssue(t *testing.T) {
	dir := t.TempDir()
	name := "hoot.dev"
	ip := "127.0.0.1"

	keyBytes, certBytes, err := cert.CreateCACert("Edge CA")
	if err != nil {
		t.Fatal(err)
	}

	caCert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := cert.IssueCert(&caCert, name, net.ParseIP(ip))
	if err != nil {
		t.Fatal(err)
	}

	certOut, err := os.Create(filepath.Join(dir, name+".crt"))
	if err != nil {
		t.Fatal(err)
	}

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	certOut.Close()

	keyOut, err := os.OpenFile(filepath.Join(dir, name+".key"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}

	bytes := x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: bytes})

	keyOut.Close()

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := leaf.VerifyHostname(name); err != nil {
		t.Error(err)
	}
}
