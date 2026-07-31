package mailadapter

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

var ErrDisabled = errors.New("email delivery is disabled")

type Config struct {
	Driver      string
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	Security    string
	Timeout     time.Duration
}

type Status struct {
	Driver      string `json:"driver"`
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	FromAddress string `json:"from_address,omitempty"`
	FromName    string `json:"from_name,omitempty"`
	Security    string `json:"security,omitempty"`
}

type InvitationMessage struct {
	RecipientEmail string
	Organization   string
	Role           string
	InvitationURL  string
	ExpiresAt      time.Time
}

type Sender interface {
	Status() Status
	SendInvitation(context.Context, InvitationMessage) error
}

func New(cfg Config) (Sender, error) {
	cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
	if cfg.Driver == "" || cfg.Driver == "disabled" {
		return disabledSender{}, nil
	}
	if cfg.Driver != "smtp" {
		return nil, fmt.Errorf("unsupported email driver %q", cfg.Driver)
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Security = strings.ToLower(strings.TrimSpace(cfg.Security))
	if cfg.Host == "" || cfg.Port < 1 || cfg.Port > 65535 {
		return nil, errors.New("SMTP host or port is invalid")
	}
	if cfg.Security != "starttls" && cfg.Security != "tls" && cfg.Security != "none" {
		return nil, errors.New("SMTP security must be starttls, tls or none")
	}
	from, err := mail.ParseAddress(cfg.FromAddress)
	if err != nil || from.Address == "" {
		return nil, errors.New("SMTP from address is invalid")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return nil, errors.New("SMTP password is required when username is set")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	cfg.FromAddress = from.Address
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	return &smtpSender{cfg: cfg}, nil
}

type disabledSender struct{}

func (disabledSender) Status() Status {
	return Status{Driver: "disabled", Enabled: false, Configured: false}
}

func (disabledSender) SendInvitation(context.Context, InvitationMessage) error {
	return ErrDisabled
}

type smtpSender struct {
	cfg Config
}

func (s *smtpSender) Status() Status {
	return Status{
		Driver:      "smtp",
		Enabled:     true,
		Configured:  true,
		FromAddress: s.cfg.FromAddress,
		FromName:    s.cfg.FromName,
		Security:    s.cfg.Security,
	}
}

func (s *smtpSender) SendInvitation(ctx context.Context, message InvitationMessage) error {
	recipient, err := mail.ParseAddress(strings.TrimSpace(message.RecipientEmail))
	if err != nil || recipient.Address == "" {
		return errors.New("recipient address is invalid")
	}
	if strings.TrimSpace(message.InvitationURL) == "" {
		return errors.New("invitation URL is required")
	}

	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	address := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect SMTP server: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if s.cfg.Security == "tls" {
		tlsConnection := tls.Client(connection, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.cfg.Host})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return fmt.Errorf("negotiate SMTP TLS: %w", err)
		}
		connection = tlsConnection
	}

	client, err := smtp.NewClient(connection, s.cfg.Host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()
	if s.cfg.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := client.Mail(s.cfg.FromAddress); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	if _, err := io.Copy(writer, strings.NewReader(buildInvitationMessage(s.cfg, recipient.Address, message))); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP session: %w", err)
	}
	return nil
}

func buildInvitationMessage(cfg Config, recipient string, message InvitationMessage) string {
	from := (&mail.Address{Name: cfg.FromName, Address: cfg.FromAddress}).String()
	subject := mime.QEncoding.Encode("UTF-8", fmt.Sprintf("加入 %s 的成员邀请", cleanText(message.Organization)))
	role := invitationRoleLabel(message.Role)
	body := fmt.Sprintf(
		"你好：\r\n\r\n你收到了加入 %s 的成员邀请，角色为%s。\r\n\r\n请在 %s 前打开以下链接完成加入：\r\n%s\r\n\r\n如果你不认识该组织，请忽略此邮件。\r\n",
		cleanText(message.Organization),
		role,
		message.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC"),
		strings.TrimSpace(message.InvitationURL),
	)
	headers := []string{
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"From: " + from,
		"To: " + (&mail.Address{Address: recipient}).String(),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func invitationRoleLabel(role string) string {
	switch role {
	case "administrator":
		return "管理员"
	case "editor":
		return "编辑者"
	default:
		return "成员"
	}
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

// readSMTPLine is kept small and internal so adapter tests can share the same
// CRLF-aware behavior as a real SMTP server without introducing a dependency.
func readSMTPLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}
