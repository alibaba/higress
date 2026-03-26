package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPolicyFingerprint(t *testing.T) {
	t.Run("TextModerationPlus default config", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = TextModerationPlus
		config.SetDefaultValues()

		fp := config.BuildPolicyFingerprint("")
		parts := strings.Split(fp, ":")
		assert.Len(t, parts, 3)
		assert.Equal(t, TextModerationPlus, parts[0])
		assert.Equal(t, DefaultTextModerationPlusTextInputCheckService, parts[1])
		assert.Equal(t, HighRisk, parts[2]) // default RiskLevelBar
	})

	t.Run("MultiModalGuard default config", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = MultiModalGuard
		config.SetDefaultValues()

		fp := config.BuildPolicyFingerprint("")
		parts := strings.Split(fp, ":")
		assert.Len(t, parts, 9)
		assert.Equal(t, MultiModalGuard, parts[0])
		assert.Equal(t, DefaultMultiModalGuardTextInputCheckService, parts[1])
		assert.Equal(t, DefaultMultiModalGuardImageInputCheckService, parts[2])
		assert.Equal(t, MaxRisk, parts[3])  // ContentModerationLevelBar
		assert.Equal(t, MaxRisk, parts[4])  // PromptAttackLevelBar
		assert.Equal(t, S4Sensitive, parts[5]) // SensitiveDataLevelBar
		assert.Equal(t, MaxRisk, parts[6])  // MaliciousUrlLevelBar
		assert.Equal(t, MaxRisk, parts[7])  // ModelHallucinationLevelBar
		assert.Equal(t, "false", parts[8])  // CheckRequestImage default
	})

	t.Run("MultiModalGuard with CheckRequestImage true", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = MultiModalGuard
		config.SetDefaultValues()
		config.CheckRequestImage = true

		fp := config.BuildPolicyFingerprint("")
		assert.True(t, strings.HasSuffix(fp, ":true"))
	})

	t.Run("different action produces different fingerprint", func(t *testing.T) {
		c1 := AISecurityConfig{}
		c1.Action = TextModerationPlus
		c1.SetDefaultValues()

		c2 := AISecurityConfig{}
		c2.Action = MultiModalGuard
		c2.SetDefaultValues()

		assert.NotEqual(t, c1.BuildPolicyFingerprint(""), c2.BuildPolicyFingerprint(""))
	})

	t.Run("different RiskLevelBar produces different fingerprint for TextModerationPlus", func(t *testing.T) {
		c1 := AISecurityConfig{}
		c1.Action = TextModerationPlus
		c1.SetDefaultValues()
		c1.RiskLevelBar = HighRisk

		c2 := AISecurityConfig{}
		c2.Action = TextModerationPlus
		c2.SetDefaultValues()
		c2.RiskLevelBar = MediumRisk

		assert.NotEqual(t, c1.BuildPolicyFingerprint(""), c2.BuildPolicyFingerprint(""))
	})

	t.Run("consumer-specific overrides change fingerprint", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = TextModerationPlus
		config.SetDefaultValues()
		config.ConsumerRiskLevel = []map[string]interface{}{
			{
				"matcher":      Matcher{Exact: "vip"},
				"riskLevelBar": "low",
			},
		}

		fpDefault := config.BuildPolicyFingerprint("regular")
		fpVip := config.BuildPolicyFingerprint("vip")
		assert.NotEqual(t, fpDefault, fpVip)
		assert.Contains(t, fpVip, "low")
	})

	t.Run("consumer-specific overrides for MultiModalGuard", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = MultiModalGuard
		config.SetDefaultValues()
		config.ConsumerRiskLevel = []map[string]interface{}{
			{
				"matcher":                    Matcher{Exact: "vip"},
				"contentModerationLevelBar":  "low",
			},
		}

		fpDefault := config.BuildPolicyFingerprint("regular")
		fpVip := config.BuildPolicyFingerprint("vip")
		assert.NotEqual(t, fpDefault, fpVip)
	})

	t.Run("same config same consumer produces stable fingerprint", func(t *testing.T) {
		config := AISecurityConfig{}
		config.Action = TextModerationPlus
		config.SetDefaultValues()

		fp1 := config.BuildPolicyFingerprint("user1")
		fp2 := config.BuildPolicyFingerprint("user1")
		assert.Equal(t, fp1, fp2)
	})
}

func TestSetDefaultValues_CheckRecordTTL(t *testing.T) {
	config := AISecurityConfig{}
	config.SetDefaultValues()
	assert.Equal(t, DefaultCheckRecordTTL, config.CheckRecordTTL)
}
