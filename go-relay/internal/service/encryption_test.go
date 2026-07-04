package service

import "testing"

const testHexKey = "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := Encrypt("hello world", testHexKey)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	plain, err := Decrypt(enc.Ciphertext, enc.IV, enc.AuthTag, testHexKey)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if plain != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", plain)
	}
}

func TestEncryptProducesDistinctIVs(t *testing.T) {
	enc1, _ := Encrypt("same plaintext", testHexKey)
	enc2, _ := Encrypt("same plaintext", testHexKey)
	if enc1.IV == enc2.IV {
		t.Error("expected distinct IVs across encryptions")
	}
	if enc1.Ciphertext == enc2.Ciphertext {
		t.Error("expected distinct ciphertext across encryptions with different IVs")
	}
}

func TestDecryptRejectsTamperedAuthTag(t *testing.T) {
	enc, _ := Encrypt("secret", testHexKey)
	tampered := enc.AuthTag[:len(enc.AuthTag)-2] + "00"
	if _, err := Decrypt(enc.Ciphertext, enc.IV, tampered, testHexKey); err == nil {
		t.Error("expected error decrypting with tampered auth tag")
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	enc, _ := Encrypt("secret", testHexKey)
	otherKey := "112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00"
	if _, err := Decrypt(enc.Ciphertext, enc.IV, enc.AuthTag, otherKey); err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

func TestGetEncryptionKeyRejectsMissing(t *testing.T) {
	if _, err := getEncryptionKey(""); err == nil {
		t.Error("expected error for empty key")
	}
}

func TestGetEncryptionKeyRejectsMalformed(t *testing.T) {
	if _, err := getEncryptionKey("not-a-valid-key"); err == nil {
		t.Error("expected error for malformed key")
	}
}

func TestGetEncryptionKeyAcceptsBase64(t *testing.T) {
	// 32 bytes base64-encoded (44 chars with padding)
	const b64Key = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	if _, err := getEncryptionKey(b64Key); err != nil {
		t.Errorf("expected base64 key to be accepted, got %v", err)
	}
}

func TestIsEncryptionAvailable(t *testing.T) {
	if IsEncryptionAvailable("") {
		t.Error("expected IsEncryptionAvailable to be false for empty key")
	}
	if !IsEncryptionAvailable(testHexKey) {
		t.Error("expected IsEncryptionAvailable to be true for valid key")
	}
}
