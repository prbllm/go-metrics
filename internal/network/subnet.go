package network

import "net"

// IsIPInTrustedSubnet проверяет, входит ли IP в доверенную подсеть.
// Если trustedSubnet пуст, возвращает true (проверка отключена).
// Ошибка возвращается только при невалидном CIDR.
func IsIPInTrustedSubnet(ipStr, trustedSubnet string) (allowed bool, err error) {
	if trustedSubnet == "" {
		return true, nil
	}
	if ipStr == "" {
		return false, nil
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, nil
	}
	_, ipNet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return false, err
	}
	return ipNet.Contains(ip), nil
}
