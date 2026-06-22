package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetChecker проверяет, что IP из заголовка X-Real-IP входит в доверенную подсеть.
type TrustedSubnetChecker struct {
	network *net.IPNet
}

// NewTrustedSubnetChecker создаёт проверку для CIDR trustedSubnet.
// Пустое значение trustedSubnet запрещает доступ для любого запроса.
func NewTrustedSubnetChecker(trustedSubnet string) (*TrustedSubnetChecker, error) {
	if trustedSubnet == "" {
		return &TrustedSubnetChecker{}, nil
	}
	_, network, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return nil, err
	}
	return &TrustedSubnetChecker{network: network}, nil
}

// Allowed возвращает true, если X-Real-IP запроса входит в доверенную подсеть.
func (c *TrustedSubnetChecker) Allowed(r *http.Request) bool {
	if c == nil || c.network == nil {
		return false
	}
	ip := net.ParseIP(r.Header.Get("X-Real-IP"))
	if ip == nil {
		return false
	}
	return c.network.Contains(ip)
}
