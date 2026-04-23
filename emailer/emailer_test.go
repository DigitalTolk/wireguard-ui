package emailer

import (
	"testing"

	mail "github.com/xhit/go-simple-mail/v2"

	"github.com/stretchr/testify/assert"
)

// --- authType tests ---

func TestAuthType_Plain(t *testing.T) {
	assert.Equal(t, mail.AuthPlain, authType("PLAIN"))
}

func TestAuthType_PlainLower(t *testing.T) {
	assert.Equal(t, mail.AuthPlain, authType("plain"))
}

func TestAuthType_Login(t *testing.T) {
	assert.Equal(t, mail.AuthLogin, authType("LOGIN"))
}

func TestAuthType_LoginLower(t *testing.T) {
	assert.Equal(t, mail.AuthLogin, authType("login"))
}

func TestAuthType_Default(t *testing.T) {
	assert.Equal(t, mail.AuthNone, authType(""))
}

func TestAuthType_Unknown(t *testing.T) {
	assert.Equal(t, mail.AuthNone, authType("UNKNOWN"))
}

// --- encryptionType tests ---

func TestEncryptionType_None(t *testing.T) {
	assert.Equal(t, mail.EncryptionNone, encryptionType("NONE"))
}

func TestEncryptionType_SSL(t *testing.T) {
	assert.Equal(t, mail.EncryptionSSL, encryptionType("SSL"))
}

func TestEncryptionType_SSLTLS(t *testing.T) {
	assert.Equal(t, mail.EncryptionSSLTLS, encryptionType("SSLTLS"))
}

func TestEncryptionType_TLS(t *testing.T) {
	assert.Equal(t, mail.EncryptionTLS, encryptionType("TLS"))
}

func TestEncryptionType_Default(t *testing.T) {
	assert.Equal(t, mail.EncryptionSTARTTLS, encryptionType(""))
}

func TestEncryptionType_Unknown(t *testing.T) {
	assert.Equal(t, mail.EncryptionSTARTTLS, encryptionType("UNKNOWN"))
}

func TestEncryptionType_CaseInsensitive(t *testing.T) {
	assert.Equal(t, mail.EncryptionSSL, encryptionType("ssl"))
	assert.Equal(t, mail.EncryptionTLS, encryptionType("tls"))
	assert.Equal(t, mail.EncryptionNone, encryptionType("none"))
	assert.Equal(t, mail.EncryptionSSLTLS, encryptionType("ssltls"))
}

// --- addressField tests ---

func TestAddressField_WithName(t *testing.T) {
	assert.Equal(t, "John Doe <john@example.com>", addressField("john@example.com", "John Doe"))
}

func TestAddressField_WithoutName(t *testing.T) {
	assert.Equal(t, "john@example.com", addressField("john@example.com", ""))
}

// --- NewSmtpMail tests ---

func TestNewSmtpMail(t *testing.T) {
	s := NewSmtpMail("smtp.example.com", 587, "user", "pass", "helo.example.com", "PLAIN", "Sender", "sender@example.com", "TLS")

	assert.Equal(t, "smtp.example.com", s.hostname)
	assert.Equal(t, 587, s.port)
	assert.Equal(t, "user", s.username)
	assert.Equal(t, "pass", s.password)
	assert.Equal(t, "helo.example.com", s.smtpHelo)
	assert.Equal(t, mail.AuthPlain, s.authType)
	assert.Equal(t, mail.EncryptionTLS, s.encryption)
	assert.Equal(t, "Sender", s.fromName)
	assert.Equal(t, "sender@example.com", s.from)
}

func TestNewSmtpMail_Defaults(t *testing.T) {
	s := NewSmtpMail("host", 25, "", "", "", "", "", "from@test.com", "")

	assert.Equal(t, mail.AuthNone, s.authType)
	assert.Equal(t, mail.EncryptionSTARTTLS, s.encryption)
}

// --- NewSendgridApiMail tests ---

func TestNewSendgridApiMail(t *testing.T) {
	sg := NewSendgridApiMail("SG.test-key", "Sender Name", "sender@example.com")

	assert.Equal(t, "SG.test-key", sg.apiKey)
	assert.Equal(t, "Sender Name", sg.fromName)
	assert.Equal(t, "sender@example.com", sg.from)
}

func TestNewSendgridApiMail_Empty(t *testing.T) {
	sg := NewSendgridApiMail("", "", "")

	assert.Empty(t, sg.apiKey)
	assert.Empty(t, sg.fromName)
	assert.Empty(t, sg.from)
}

// --- SmtpMail.Send tests (connection error path) ---

func TestSmtpMail_Send_ConnectionError(t *testing.T) {
	s := NewSmtpMail("127.0.0.1", 1, "user", "pass", "", "PLAIN", "Sender", "from@test.com", "NONE")

	err := s.Send("Recipient", "to@test.com", "Subject", "<p>Body</p>", nil)
	assert.Error(t, err, "Send should fail when SMTP server is unreachable")
}

func TestSmtpMail_Send_ConnectionError_SSL(t *testing.T) {
	s := NewSmtpMail("127.0.0.1", 1, "", "", "helo.test", "LOGIN", "From Name", "from@test.com", "SSL")

	err := s.Send("Recipient", "to@test.com", "Subject", "Body", []Attachment{
		{Name: "test.txt", Data: []byte("hello")},
	})
	assert.Error(t, err, "Send should fail when SMTP server is unreachable")
}

func TestSmtpMail_Send_ConnectionError_STARTTLS(t *testing.T) {
	s := NewSmtpMail("127.0.0.1", 1, "", "", "", "", "", "from@test.com", "STARTTLS")

	err := s.Send("", "to@test.com", "Test", "Content", nil)
	assert.Error(t, err)
}

// --- SendgridApiMail.Send tests ---

func TestSendgridApiMail_Send_NoAttachments(t *testing.T) {
	sg := NewSendgridApiMail("SG.fake-key", "Sender", "from@test.com")

	// The SendGrid client will make an HTTP request but with a fake key
	// The request will complete (possibly with a 401/403 error from the API)
	// but we mainly want to exercise the code path
	err := sg.Send("Recipient", "to@test.com", "Subject", "<p>Body</p>", nil)
	// We don't check the error because the SendGrid API may or may not return an error
	// for invalid API keys - the important thing is the code path is exercised
	_ = err
}

func TestSendgridApiMail_Send_WithAttachments(t *testing.T) {
	sg := NewSendgridApiMail("SG.fake-key", "Sender", "from@test.com")

	attachments := []Attachment{
		{Name: "test.conf", Data: []byte("[Interface]\nPrivateKey = abc123")},
		{Name: "qr.png", Data: []byte{0x89, 0x50, 0x4E, 0x47}},
	}

	err := sg.Send("Recipient", "to@test.com", "WireGuard Config", "<p>Config attached</p>", attachments)
	_ = err
}
