package storage

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/mixin"
	"github.com/kilip/sbctl/internal/config"
	"github.com/kilip/sbctl/internal/memory"
)

func setupTestDB(t *testing.T) (*ent.Client, context.Context) {
	cfg := &config.Config{}
	cfg.Db.Path = "file::memory:?cache=shared"
	cfg.Db.Driver = "sqlite"

	client, err := Open(cfg)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		client.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	return client, ctx
}

func TestIntegration_AC001_Migration(t *testing.T) {
	client, _ := setupTestDB(t)
	defer client.Close()
	// Migration runs successfully in setupTestDB
	t.Log("Auto-migration provisioned all tables successfully (AC-001)")
}

func TestIntegration_AC002_UUIDv7MessageOrdering(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	// Setup User and Account
	u, _ := CreateUser(ctx, client, "Toni", "user")
	acc, _ := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)
	sess, _ := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)

	// Create messages sequentially with short pauses
	for i := 0; i < 5; i++ {
		content, _ := json.Marshal(map[string]string{"text": "msg"})
		_, err := CreateMessage(ctx, client, sess.ID, "user", content, "done", 0, nil)
		if err != nil {
			t.Fatalf("failed to create message: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Retrieve messages
	retrieved, err := GetMessagesBySession(ctx, client, sess.ID, 0)
	if err != nil {
		t.Fatalf("failed to retrieve messages: %v", err)
	}

	if len(retrieved) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(retrieved))
	}

	// Verify sorting (UUID v7 string comparison is lexicographically identical to chronological order)
	for i := 0; i < 4; i++ {
		if strings.Compare(retrieved[i].ID.String(), retrieved[i+1].ID.String()) >= 0 {
			t.Errorf("messages not in chronological order: index %d >= %d", i, i+1)
		}
	}
}

func TestIntegration_AC003_SoftDelete(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	u, err := CreateUser(ctx, client, "Toni", "user")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Standard retrieval works
	ru, err := GetUserByID(ctx, client, u.ID)
	if err != nil || ru == nil {
		t.Fatalf("user not found: %v", err)
	}

	// Soft delete
	err = SoftDeleteUser(ctx, client, u.ID)
	if err != nil {
		t.Fatalf("failed to soft delete: %v", err)
	}

	// Standard retrieval must fail
	ru, err = GetUserByID(ctx, client, u.ID)
	if err == nil || ru != nil {
		t.Error("expected soft deleted user to be excluded from standard query")
	}

	// Retrieval with bypass context must succeed
	bypassCtx := mixin.SkipSoftDelete(ctx)
	ru, err = GetUserByID(bypassCtx, client, u.ID)
	if err != nil || ru == nil {
		t.Errorf("failed to retrieve soft-deleted record with bypass context: %v", err)
	}
}

func TestIntegration_AC005_AuditLog(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	u, _ := CreateUser(ctx, client, "Toni", "user")
	acc, _ := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)
	sess, _ := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)

	// Create message triggers AuditLog automatically
	content, _ := json.Marshal(map[string]string{"text": "msg"})
	msg, err := CreateMessage(ctx, client, sess.ID, "user", content, "done", 0, nil)
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Check AuditLog table
	logs, err := client.AuditLog.Query().All(ctx)
	if err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("expected 1 audit log entry, got %d", len(logs))
	} else {
		if logs[0].Action != "message_create" {
			t.Errorf("expected action 'message_create', got %q", logs[0].Action)
		}
		if logs[0].EntityID != msg.ID {
			t.Errorf("expected entity ID %s, got %s", msg.ID, logs[0].EntityID)
		}
	}

	// Attempting to delete AuditLog must fail
	err = client.AuditLog.DeleteOneID(logs[0].ID).Exec(ctx)
	if err == nil || !strings.Contains(err.Error(), "auditlog is immutable") {
		t.Errorf("expected delete audit log to fail with immutability error, got: %v", err)
	}

	// Attempting to update AuditLog must fail
	_, err = client.AuditLog.UpdateOneID(logs[0].ID).SetAction("hacked").Save(ctx)
	if err == nil || !strings.Contains(err.Error(), "auditlog is immutable") {
		t.Errorf("expected update audit log to fail with immutability error, got: %v", err)
	}
}

func TestIntegration_AC006_MessageToolRoleValidation(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	u, _ := CreateUser(ctx, client, "Toni", "user")
	acc, _ := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)
	sess, _ := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)

	// Tool role without tool_call_id and result should fail validation
	badContent, _ := json.Marshal(map[string]string{"text": "nothing"})
	_, err := CreateMessage(ctx, client, sess.ID, "tool", badContent, "done", 0, nil)
	if err == nil || !strings.Contains(err.Error(), "missing tool_call_id") {
		t.Errorf("expected tool message creation without required fields to fail, got: %v", err)
	}

	// Tool role with tool_call_id and result should succeed
	goodContent, _ := json.Marshal(map[string]string{
		"tool_call_id": "call-1",
		"result":       "success",
	})
	_, err = CreateMessage(ctx, client, sess.ID, "tool", goodContent, "done", 0, nil)
	if err != nil {
		t.Fatalf("tool message with correct fields failed creation: %v", err)
	}
}

func TestIntegration_AC007_MemoryExtractionFlow(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	u, _ := CreateUser(ctx, client, "Toni", "user")
	acc, _ := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)
	sess, _ := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)

	msgContent, _ := json.Marshal(map[string]string{"text": "My name is Toni and I live in Jakarta"})
	msg, _ := CreateMessage(ctx, client, sess.ID, "user", msgContent, "done", 0, nil)

	// Extract facts
	facts, err := memory.ExtractFacts(ctx, sess, msg)
	if err != nil {
		t.Fatalf("failed to extract facts: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}

	// Persist fact
	metaJSON, _ := json.Marshal(facts[0].Metadata)
	mem, err := SaveMemory(ctx, client, u.ID, sess.ID, msg.ID, facts[0].Content, metaJSON)
	if err != nil {
		t.Fatalf("failed to save memory: %v", err)
	}

	if mem.UserID != u.ID || mem.SessionID != sess.ID || mem.MessageID != msg.ID {
		t.Error("memory record saved with incorrect association keys")
	}

	// Test GetMemoryByUser metadata filters
	memories, err := GetMemoryByUser(ctx, client, u.ID, map[string]string{"type": "semantic_fact"})
	if err != nil {
		t.Fatalf("failed to query memory by user: %v", err)
	}
	if len(memories) != 1 {
		t.Errorf("expected 1 memory matching metadata filter, got %d", len(memories))
	}

	// Delete Memory (Physical delete)
	err = DeleteMemory(ctx, client, mem.ID)
	if err != nil {
		t.Fatalf("failed to physically delete memory: %v", err)
	}

	memories, _ = GetMemoryByUser(ctx, client, u.ID, nil)
	if len(memories) != 0 {
		t.Errorf("expected memory to be physically deleted, but still found %d records", len(memories))
	}
}

func TestIntegration_AC008_AC009_AutoProfileProvisionAndDefaultConstraint(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	// Creating User automatically provisions a default profile
	u, err := CreateUser(ctx, client, "Toni", "user")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	profiles, err := client.Profile.Query().All(ctx)
	if err != nil {
		t.Fatalf("failed to query profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("expected exactly 1 profile auto-provisioned, got %d", len(profiles))
	}

	vaultProfile := profiles[0]
	if vaultProfile.Name != "vault" || vaultProfile.WorkingDir != "~/brain" || !vaultProfile.IsDefault {
		t.Errorf("auto-provisioned profile has incorrect settings: %+v", vaultProfile)
	}

	// Create second profile "sbctl" with is_default = false
	sbctlProfile, err := CreateProfile(ctx, client, u.ID, "sbctl", "~/other", false)
	if err != nil {
		t.Fatalf("failed to create second profile: %v", err)
	}

	// Switch sbctl to is_default = true
	err = SetDefaultProfile(ctx, client, sbctlProfile.ID)
	if err != nil {
		t.Fatalf("failed to set second profile as default: %v", err)
	}

	// Verify vault is no longer default, sbctl is default
	pVault, _ := client.Profile.Get(ctx, vaultProfile.ID)
	pSbctl, _ := client.Profile.Get(ctx, sbctlProfile.ID)

	if pVault.IsDefault {
		t.Error("expected vault profile default flag to be unset (false)")
	}
	if !pSbctl.IsDefault {
		t.Error("expected sbctl profile default flag to be set (true)")
	}
}

func TestIntegration_AC010_NullProfileResolution(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	u, _ := CreateUser(ctx, client, "Toni", "user")
	acc, _ := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)

	// Create session with null profile_id
	sess, err := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	dir, err := ResolveSessionProfileDir(ctx, client, sess)
	if err != nil {
		t.Fatalf("failed to resolve session profile dir: %v", err)
	}

	// Resolves to default user profile: "~/brain"
	if dir != "~/brain" {
		t.Errorf("expected resolved profile dir to be '~/brain', got %q", dir)
	}
}

func TestIntegration_CoverageEnhancements(t *testing.T) {
	client, ctx := setupTestDB(t)
	defer client.Close()

	// 1. AuditLog: CreateAuditLog
	u, err := CreateUser(ctx, client, "Toni", "user")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	acc, err := CreateAccount(ctx, client, u.ID, "telegram", "12345", "active", nil)
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	sess, err := CreateSession(ctx, client, acc.ID, nil, "Episodic", "Test", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	content, _ := json.Marshal(map[string]string{"text": "msg"})
	msg, err := CreateMessage(ctx, client, sess.ID, "user", content, "done", 0, nil)
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	auditJSON, _ := json.Marshal(map[string]string{"info": "test"})
	audit, err := CreateAuditLog(ctx, client, u.ID, "custom_action", "user", u.ID, auditJSON)
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}
	if audit.Action != "custom_action" {
		t.Errorf("expected audit action 'custom_action', got %q", audit.Action)
	}

	// 2. Message: UpdateMessageStatus & GetMessagesBySession with limit
	updatedMsg, err := UpdateMessageStatus(ctx, client, msg.ID, "awaiting_approval")
	if err != nil {
		t.Fatalf("failed to update message status: %v", err)
	}
	if updatedMsg.Status != "awaiting_approval" {
		t.Errorf("expected status 'awaiting_approval', got %q", updatedMsg.Status)
	}

	// GetMessagesBySession with limit > 0
	msgs, err := GetMessagesBySession(ctx, client, sess.ID, 1)
	if err != nil {
		t.Fatalf("failed to get messages by session with limit: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}

	// 3. Profile: SoftDeleteProfile
	// Non-default profile soft-delete
	pNonDefault, err := CreateProfile(ctx, client, u.ID, "extra", "~/extra", false)
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}
	err = SoftDeleteProfile(ctx, client, pNonDefault.ID)
	if err != nil {
		t.Fatalf("failed to soft delete non-default profile: %v", err)
	}
	// Try soft deleting default profile (should fail)
	pDefault, err := GetDefaultProfile(ctx, client, u.ID)
	if err != nil {
		t.Fatalf("failed to get default profile: %v", err)
	}
	err = SoftDeleteProfile(ctx, client, pDefault.ID)
	if err == nil {
		t.Error("expected error soft deleting default profile, got nil")
	}
	// Error path for non-existing profile
	err = SoftDeleteProfile(ctx, client, uuid.New())
	if err == nil {
		t.Error("expected error soft deleting non-existing profile, got nil")
	}

	// 4. Session: GetSessionByID, GetSessionByKey, SoftDeleteSession, ResolveSessionProfileDir (with non-null profile ID)
	sByID, err := GetSessionByID(ctx, client, sess.ID)
	if err != nil {
		t.Fatalf("failed to get session by ID: %v", err)
	}
	if sByID.ID != sess.ID {
		t.Errorf("expected session ID %s, got %s", sess.ID, sByID.ID)
	}

	// ResolveSessionProfileDir with non-null profile ID
	// Set profile_id on session
	sessWithProfile, err := CreateSession(ctx, client, acc.ID, &pDefault.ID, "Episodic", "Test with Profile", nil)
	if err != nil {
		t.Fatalf("failed to create session with profile: %v", err)
	}
	dir, err := ResolveSessionProfileDir(ctx, client, sessWithProfile)
	if err != nil {
		t.Fatalf("failed to resolve profile dir: %v", err)
	}
	if dir != pDefault.WorkingDir {
		t.Errorf("expected profile dir %q, got %q", pDefault.WorkingDir, dir)
	}

	// GetSessionByKey
	sessionKey := "unique-key-123"
	metaWithKey, _ := json.Marshal(map[string]interface{}{"session_key": sessionKey})
	sessWithKey, err := CreateSession(ctx, client, acc.ID, nil, "KeySession", "Test Key", metaWithKey)
	if err != nil {
		t.Fatalf("failed to create session with key: %v", err)
	}
	sByKey, err := GetSessionByKey(ctx, client, sessionKey)
	if err != nil {
		t.Fatalf("failed to get session by key: %v", err)
	}
	if sByKey.ID != sessWithKey.ID {
		t.Errorf("expected session ID %s, got %s", sessWithKey.ID, sByKey.ID)
	}

	// SoftDeleteSession
	err = SoftDeleteSession(ctx, client, sess.ID)
	if err != nil {
		t.Fatalf("failed to soft delete session: %v", err)
	}
	_, err = GetSessionByID(ctx, client, sess.ID)
	if err == nil {
		t.Error("expected error getting soft deleted session, got nil")
	}

	// 5. Account: GetAccountByPlatformID, SoftDeleteAccount
	aByPlatform, err := GetAccountByPlatformID(ctx, client, "telegram", "12345")
	if err != nil {
		t.Fatalf("failed to get account by platform and external ID: %v", err)
	}
	if aByPlatform.ID != acc.ID {
		t.Errorf("expected account ID %s, got %s", acc.ID, aByPlatform.ID)
	}

	err = SoftDeleteAccount(ctx, client, acc.ID)
	if err != nil {
		t.Fatalf("failed to soft delete account: %v", err)
	}
	_, err = GetAccountByPlatformID(ctx, client, "telegram", "12345")
	if err == nil {
		t.Error("expected error getting soft deleted account, got nil")
	}
}
