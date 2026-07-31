package game

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strings"
)

// MaskIP 脱敏显示 IP，支持 IPv4 / IPv6
func MaskIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	// 可能带端口的 remote addr
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}
	// 去掉方括号
	ipStr = strings.Trim(ipStr, "[]")

	ip := net.ParseIP(ipStr)
	if ip == nil {
		if len(ipStr) > 8 {
			return ipStr[:4] + "***"
		}
		return "***"
	}

	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.*.*", v4[0], v4[1])
	}

	// IPv6: 保留前两段
	s := ip.String()
	parts := strings.Split(s, ":")
	if len(parts) >= 2 {
		return parts[0] + ":" + parts[1] + ":*:*"
	}
	return s[:min(4, len(s))] + ":***"
}

// ThemeColorFromIP 由 IP 生成唯一主题色 (HSL)
func ThemeColorFromIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}
	ipStr = strings.Trim(ipStr, "[]")

	sum := sha256.Sum256([]byte(ipStr))
	h := int(sum[0])<<8 | int(sum[1])
	hue := h % 360
	sat := 55 + int(sum[2])%25   // 55-79
	light := 45 + int(sum[3])%15 // 45-59
	return fmt.Sprintf("hsl(%d, %d%%, %d%%)", hue, sat, light)
}

func ClientIP(remoteAddr, xff, xri string) string {
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri != "" {
		return strings.TrimSpace(xri)
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
