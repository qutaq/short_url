package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"time"
)

func listenAndServe(srv *http.Server, enableHTTPS bool) error {
	if !enableHTTPS {
		return srv.ListenAndServe()
	}

	tlsConfig, err := newTLSConfig()
	if err != nil {
		return err
	}

	listener, err := tls.Listen("tcp", srv.Addr, tlsConfig)
	if err != nil {
		return err
	}

	log.Printf("HTTPS enabled")
	return srv.Serve(listener)
}

func newTLSConfig() (*tls.Config, error) {
	cert, err := newSelfSignedCertificate()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func newSelfSignedCertificate() (tls.Certificate, error) {
	// Создаём шаблон сертификата.
	cert := &x509.Certificate{
		// Указываем уникальный номер сертификата.
		SerialNumber: big.NewInt(1658),
		// Заполняем базовую информацию о владельце сертификата.
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		// Разрешаем использование сертификата для localhost, 127.0.0.1 и ::1.
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost"},
		// Сертификат верен, начиная со времени создания.
		NotBefore: time.Now(),
		// Время жизни сертификата - 10 лет.
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// Устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации.
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,

		BasicConstraintsValid: true,
	}

	// Создаём новый приватный RSA-ключ.
	// Для генерации ключа и сертификата используется rand.Reader как источник случайных данных.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Создаём сертификат x.509.
	certDER, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Кодируем сертификат и ключ в формате PEM,
	// который используется для хранения и обмена криптографическими ключами.
	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return tls.Certificate{}, err
	}

	var privateKeyPEM bytes.Buffer
	if err := pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPEM.Bytes(), privateKeyPEM.Bytes())
}
