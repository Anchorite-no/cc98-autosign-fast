package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const defaultEnvTemplate = `# WebVPN 账号
WEBVPN_USER=
WEBVPN_PASS=

# 需要签到的 CC98 账号数量
CC98_ACCOUNT_COUNT=1

# 第 1 个 CC98 账号
CC98_USER_1=
CC98_PASS_1=
`

type Account struct {
	Index    int
	Username string
	Password string
}

type Config struct {
	EnvPath    string
	WebVPNUser string
	WebVPNPass string
	Accounts   []Account
}

type ConfigValidationError struct {
	MissingFields   []string
	InvalidMessages []string
}

func (e *ConfigValidationError) Error() string {
	return "配置未完成"
}

func EnsureEnvFile(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("检查配置文件失败: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultEnvTemplate), 0o600); err != nil {
		return false, fmt.Errorf("生成配置模板失败: %w", err)
	}
	return true, nil
}

func LoadConfig(envPath string) (Config, error) {
	values, err := parseEnvFile(envPath)
	if err != nil {
		return Config{}, err
	}
	return buildConfig(values, envPath)
}

func buildConfig(values map[string]string, envPath string) (Config, error) {
	missing := make([]string, 0)
	invalid := make([]string, 0)

	webvpnUser := strings.TrimSpace(values["WEBVPN_USER"])
	webvpnPass := strings.TrimSpace(values["WEBVPN_PASS"])
	if webvpnUser == "" {
		missing = append(missing, "WEBVPN_USER")
	}
	if webvpnPass == "" {
		missing = append(missing, "WEBVPN_PASS")
	}

	accountCount := 0
	accountCountRaw := strings.TrimSpace(values["CC98_ACCOUNT_COUNT"])
	switch {
	case accountCountRaw == "":
		missing = append(missing, "CC98_ACCOUNT_COUNT")
	default:
		num, err := strconv.Atoi(accountCountRaw)
		if err != nil || num <= 0 {
			invalid = append(invalid, "CC98_ACCOUNT_COUNT 必须是正整数")
		} else {
			accountCount = num
		}
	}

	accounts := make([]Account, 0, accountCount)
	for i := 1; i <= accountCount; i++ {
		userKey := fmt.Sprintf("CC98_USER_%d", i)
		passKey := fmt.Sprintf("CC98_PASS_%d", i)
		username := strings.TrimSpace(values[userKey])
		password := strings.TrimSpace(values[passKey])

		if username == "" {
			missing = append(missing, userKey)
		}
		if password == "" {
			missing = append(missing, passKey)
		}
		if username != "" && password != "" {
			accounts = append(accounts, Account{
				Index:    i,
				Username: username,
				Password: password,
			})
		}
	}

	if len(missing) > 0 || len(invalid) > 0 {
		slices.Sort(missing)
		return Config{}, &ConfigValidationError{
			MissingFields:   missing,
			InvalidMessages: invalid,
		}
	}

	return Config{
		EnvPath:    envPath,
		WebVPNUser: webvpnUser,
		WebVPNPass: webvpnPass,
		Accounts:   accounts,
	}, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("找不到配置文件: %s", path)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if lineNo == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "export ") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
		}

		key, rawValue, ok := strings.Cut(trimmed, "=")
		if !ok {
			return nil, fmt.Errorf("配置文件第 %d 行缺少 '='", lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("配置文件第 %d 行变量名为空", lineNo)
		}

		value, err := parseEnvValue(strings.TrimSpace(rawValue))
		if err != nil {
			return nil, fmt.Errorf("配置文件第 %d 行解析失败: %w", lineNo, err)
		}
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描配置文件失败: %w", err)
	}
	return values, nil
}

func parseEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", err
		}
		return value, nil
	}

	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return raw[1 : len(raw)-1], nil
	}

	if commentIndex := strings.Index(raw, " #"); commentIndex >= 0 {
		raw = raw[:commentIndex]
	}
	return strings.TrimSpace(raw), nil
}
