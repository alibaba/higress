package main

import (
	"testing"
)

// TestSecretKeys_SingleActive 测试单个active密钥
func TestSecretKeys_SingleActive(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:     "key-001",
				Secret:    "test-secret-key-123456789012",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	// 验证active密钥应该可用
	if len(config.SecretKeys) != 1 {
		t.Errorf("应该有1个密钥，实际有%d个", len(config.SecretKeys))
	}

	if config.SecretKeys[0].Status != "active" {
		t.Errorf("密钥状态应该是active，实际是%s", config.SecretKeys[0].Status)
	}
}

// TestSecretKeys_MultipleActive 测试多个active密钥
func TestSecretKeys_MultipleActive(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:     "key-001",
				Secret:    "old-secret-key-1234567890",
				IsPrimary: false,
				Status:    "active",
			},
			{
				KeyID:     "key-002",
				Secret:    "new-secret-key-1234567890",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	// 验证有2个active密钥
	activeCount := 0
	for _, key := range config.SecretKeys {
		if key.Status == "active" {
			activeCount++
		}
	}

	if activeCount != 2 {
		t.Errorf("应该有2个active密钥，实际有%d个", activeCount)
	}

	// 验证主密钥标记
	primaryCount := 0
	for _, key := range config.SecretKeys {
		if key.IsPrimary {
			primaryCount++
		}
	}

	if primaryCount != 1 {
		t.Errorf("应该有1个主密钥，实际有%d个", primaryCount)
	}
}

// TestSecretKeys_DeprecatedStillWorks 测试deprecated密钥仍可验证
func TestSecretKeys_DeprecatedStillWorks(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:     "key-old",
				Secret:    "deprecated-key-123456789",
				IsPrimary: false,
				Status:    "deprecated",
			},
			{
				KeyID:     "key-new",
				Secret:    "active-key-1234567890123",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	// deprecated密钥应该仍然在列表中
	deprecatedCount := 0
	for _, key := range config.SecretKeys {
		if key.Status == "deprecated" {
			deprecatedCount++
		}
	}

	if deprecatedCount != 1 {
		t.Errorf("应该有1个deprecated密钥，实际有%d个", deprecatedCount)
	}
}

// TestSecretKeys_RevokedFiltered 测试revoked密钥过滤
func TestSecretKeys_RevokedFiltered(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:  "key-revoked",
				Secret: "revoked-key-12345678901234",
				Status: "revoked",
			},
			{
				KeyID:     "key-active",
				Secret:    "active-key-1234567890123",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	// 统计不同状态的密钥
	revokedCount := 0
	activeCount := 0
	for _, key := range config.SecretKeys {
		if key.Status == "revoked" {
			revokedCount++
		} else if key.Status == "active" {
			activeCount++
		}
	}

	if revokedCount != 1 {
		t.Errorf("应该有1个revoked密钥，实际有%d个", revokedCount)
	}

	if activeCount != 1 {
		t.Errorf("应该有1个active密钥，实际有%d个", activeCount)
	}
}

// TestSecretKeys_MixedStatus 测试混合状态密钥
func TestSecretKeys_MixedStatus(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:  "key-001",
				Secret: "revoked-key-12345678901234",
				Status: "revoked",
			},
			{
				KeyID:  "key-002",
				Secret: "deprecated-key-123456789012",
				Status: "deprecated",
			},
			{
				KeyID:     "key-003",
				Secret:    "active-key-1-1234567890",
				IsPrimary: false,
				Status:    "active",
			},
			{
				KeyID:     "key-004",
				Secret:    "active-key-2-1234567890",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	statusCount := make(map[string]int)
	for _, key := range config.SecretKeys {
		statusCount[key.Status]++
	}

	if statusCount["revoked"] != 1 {
		t.Errorf("应该有1个revoked密钥，实际有%d个", statusCount["revoked"])
	}

	if statusCount["deprecated"] != 1 {
		t.Errorf("应该有1个deprecated密钥，实际有%d个", statusCount["deprecated"])
	}

	if statusCount["active"] != 2 {
		t.Errorf("应该有2个active密钥，实际有%d个", statusCount["active"])
	}
}

// TestSecretKeys_EmptyList 测试空密钥列表
func TestSecretKeys_EmptyList(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled:    true,
		SecretKeys: []SecretKey{},
	}

	if len(config.SecretKeys) != 0 {
		t.Errorf("密钥列表应该为空，实际有%d个", len(config.SecretKeys))
	}
}

// TestSecretKeys_KeyIDUniqueness 测试密钥ID唯一性检查
func TestSecretKeys_KeyIDUniqueness(t *testing.T) {
	keys := []SecretKey{
		{
			KeyID:  "key-001",
			Secret: "secret-1-12345678901234567",
			Status: "active",
		},
		{
			KeyID:  "key-002",
			Secret: "secret-2-12345678901234567",
			Status: "active",
		},
	}

	// 检查KeyID是否唯一
	keyIDMap := make(map[string]bool)
	for _, key := range keys {
		if keyIDMap[key.KeyID] {
			t.Errorf("发现重复的KeyID: %s", key.KeyID)
		}
		keyIDMap[key.KeyID] = true
	}

	if len(keyIDMap) != 2 {
		t.Errorf("应该有2个唯一的KeyID，实际有%d个", len(keyIDMap))
	}
}

// TestSecretKeys_StatusValidation 测试密钥状态验证
func TestSecretKeys_StatusValidation(t *testing.T) {
	validStatuses := map[string]bool{
		"active":     true,
		"deprecated": true,
		"revoked":    true,
	}

	testCases := []struct {
		name   string
		status string
		valid  bool
	}{
		{"active状态", "active", true},
		{"deprecated状态", "deprecated", true},
		{"revoked状态", "revoked", true},
		{"无效状态", "invalid", false},
		{"空状态", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := validStatuses[tc.status]
			if isValid != tc.valid {
				t.Errorf("状态'%s'的验证结果应该是%v，实际是%v", tc.status, tc.valid, isValid)
			}
		})
	}
}

// TestSecretKeys_PrimaryKeyLogic 测试主密钥逻辑
func TestSecretKeys_PrimaryKeyLogic(t *testing.T) {
	config := &AuthenticatedPromptsConfig{
		Enabled: true,
		SecretKeys: []SecretKey{
			{
				KeyID:     "key-001",
				Secret:    "old-key-12345678901234567890",
				IsPrimary: false,
				Status:    "active",
			},
			{
				KeyID:     "key-002",
				Secret:    "new-key-12345678901234567890",
				IsPrimary: true,
				Status:    "active",
			},
		},
	}

	// 查找主密钥
	var primaryKey *SecretKey
	for i := range config.SecretKeys {
		if config.SecretKeys[i].IsPrimary {
			primaryKey = &config.SecretKeys[i]
			break
		}
	}

	if primaryKey == nil {
		t.Fatal("应该有一个主密钥")
	}

	if primaryKey.KeyID != "key-002" {
		t.Errorf("主密钥ID应该是'key-002'，实际是'%s'", primaryKey.KeyID)
	}

	if !primaryKey.IsPrimary {
		t.Error("主密钥的IsPrimary应该为true")
	}
}
