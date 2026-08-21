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
	RecipientEmail  string
	Organization    string
	Role            string
	InvitationURL   string
	ExpiresAt       time.Time
	SubjectTemplate string
	BodyTemplate    string
}

type ApplicationDecisionMessage struct {
	RecipientEmail  string
	Organization    string
	ApplicantName   string
	ApplicationType string
	Decision        string
	Reason          string
}

type ContentReviewMessage struct {
	RecipientEmail string
	RecipientName  string
	Organization   string
	EventType      string
	ContentTitle   string
	RequesterName  string
	ReviewerName   string
	Note           string
	Feedback       string
}

type Sender interface {
	Status() Status
	SendInvitation(context.Context, InvitationMessage) error
	SendApplicationDecision(context.Context, ApplicationDecisionMessage) error
	SendContentReview(context.Context, ContentReviewMessage) error
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

// Probe verifies network reachability, transport security and authentication
// without sending a message. It is used by the administrator-facing
// integration test action.
func Probe(ctx context.Context, cfg Config) error {
	sender, err := New(cfg)
	if err != nil {
		return err
	}
	smtpAdapter, ok := sender.(*smtpSender)
	if !ok {
		return ErrDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, smtpAdapter.cfg.Timeout)
	defer cancel()
	client, err := smtpAdapter.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP probe: %w", err)
	}
	return nil
}

type disabledSender struct{}

func (disabledSender) Status() Status {
	return Status{Driver: "disabled", Enabled: false, Configured: false}
}

func (disabledSender) SendInvitation(context.Context, InvitationMessage) error {
	return ErrDisabled
}

func (disabledSender) SendApplicationDecision(context.Context, ApplicationDecisionMessage) error {
	return ErrDisabled
}

func (disabledSender) SendContentReview(context.Context, ContentReviewMessage) error {
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

	body := invitationBody(message)
	subject := invitationSubject(message)
	return s.sendText(ctx, recipient.Address, subject, body)
}

func (s *smtpSender) SendApplicationDecision(ctx context.Context, message ApplicationDecisionMessage) error {
	recipient, err := mail.ParseAddress(strings.TrimSpace(message.RecipientEmail))
	if err != nil || recipient.Address == "" {
		return errors.New("recipient address is invalid")
	}
	decision := "申请状态已更新"
	if message.Decision == "approved" {
		decision = "申请已通过"
	} else if message.Decision == "rejected" {
		decision = "申请未通过"
	}
	body := fmt.Sprintf("你好 %s：\r\n\r\n你提交给 %s 的%s已更新为：%s。\r\n\r\n处理说明：%s\r\n", cleanText(message.ApplicantName), cleanText(message.Organization), applicationTypeLabel(message.ApplicationType), decision, cleanText(message.Reason))
	return s.sendText(ctx, recipient.Address, cleanText(message.Organization)+" 的申请处理结果", body)
}

func (s *smtpSender) SendContentReview(ctx context.Context, message ContentReviewMessage) error {
	recipient, err := mail.ParseAddress(strings.TrimSpace(message.RecipientEmail))
	if err != nil || recipient.Address == "" {
		return errors.New("recipient address is invalid")
	}
	organization := cleanText(message.Organization)
	title := cleanText(message.ContentTitle)
	requester := cleanText(message.RequesterName)
	reviewer := cleanText(message.ReviewerName)
	note := cleanText(message.Note)
	feedback := cleanText(message.Feedback)
	greeting := "你好"
	if name := cleanText(message.RecipientName); name != "" {
		greeting += " " + name
	}

	var subject, body string
	switch message.EventType {
	case "content.review_submitted":
		subject = fmt.Sprintf("【%s】内容待审核：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 提交了内容《%s》的发布审核。\r\n\r\n提交说明：%s\r\n\r\n请登录管理工作台完成审核。\r\n", greeting, requester, title, fallbackText(note, "无"))
	case "content.review_rejected":
		subject = fmt.Sprintf("【%s】内容审核已退回：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 已退回你提交的内容《%s》。\r\n\r\n审核反馈：%s\r\n\r\n请修改草稿后重新提交审核。\r\n", greeting, reviewer, title, fallbackText(feedback, "未填写"))
	case "content.published":
		subject = fmt.Sprintf("【%s】内容已上线：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 已审核并发布内容《%s》。\r\n", greeting, reviewer, title)
	case "content.archive_requested":
		subject = fmt.Sprintf("【%s】内容申请下线：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 申请下线内容《%s》。\r\n\r\n申请说明：%s\r\n\r\n请登录管理工作台处理。\r\n", greeting, requester, title, fallbackText(note, "无"))
	case "content.archive_rejected":
		subject = fmt.Sprintf("【%s】下线申请未通过：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 未通过内容《%s》的下线申请。\r\n\r\n审核反馈：%s\r\n", greeting, reviewer, title, fallbackText(feedback, "未填写"))
	case "content.archived":
		subject = fmt.Sprintf("【%s】内容已下线：%s", organization, title)
		body = fmt.Sprintf("%s：\r\n\r\n%s 已将内容《%s》下线。原作者现在可以重新编辑并提交审核。\r\n", greeting, reviewer, title)
	default:
		return errors.New("unsupported content review event")
	}
	return s.sendText(ctx, recipient.Address, subject, body)
}

func fallbackText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *smtpSender) sendText(ctx context.Context, recipient, subject, body string) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	client, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Mail(s.cfg.FromAddress); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	if _, err := io.Copy(writer, strings.NewReader(buildTextMessage(s.cfg, recipient, subject, body))); err != nil {
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

func (s *smtpSender) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect SMTP server: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if s.cfg.Security == "tls" {
		tlsConnection := tls.Client(connection, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.cfg.Host})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("negotiate SMTP TLS: %w", err)
		}
		connection = tlsConnection
	}

	client, err := smtp.NewClient(connection, s.cfg.Host)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("initialize SMTP client: %w", err)
	}
	if s.cfg.Security == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.cfg.Host}); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	return client, nil
}

func buildInvitationMessage(cfg Config, recipient string, message InvitationMessage) string {
	return buildTextMessage(cfg, recipient, invitationSubject(message), invitationBody(message))
}

func buildTextMessage(cfg Config, recipient, subject, body string) string {
	from := (&mail.Address{Name: cfg.FromName, Address: cfg.FromAddress}).String()
	subject = mime.QEncoding.Encode("UTF-8", cleanText(subject))
	body = strings.ReplaceAll(body, "\n", "\r\n")
	body = strings.ReplaceAll(body, "\r\r\n", "\r\n")
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

func invitationSubject(message InvitationMessage) string {
	values := map[string]string{
		"organization": cleanText(message.Organization),
		"role":         invitationRoleLabel(message.Role),
		"invite_url":   strings.TrimSpace(message.InvitationURL),
		"expires_at":   message.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC"),
	}
	subject := strings.TrimSpace(message.SubjectTemplate)
	if subject == "" {
		subject = fmt.Sprintf("加入 %s 的成员邀请", values["organization"])
	}
	return applyTemplate(subject, values)
}

func invitationBody(message InvitationMessage) string {
	values := map[string]string{
		"organization": cleanText(message.Organization),
		"role":         invitationRoleLabel(message.Role),
		"invite_url":   strings.TrimSpace(message.InvitationURL),
		"expires_at":   message.ExpiresAt.UTC().Format("2006-01-02 15:04 UTC"),
	}
	body := strings.TrimSpace(message.BodyTemplate)
	if body == "" {
		body = "你好：\n\n你收到了加入 {{organization}} 的成员邀请，角色为{{role}}。\n\n请在 {{expires_at}} 前打开以下链接完成加入：\n{{invite_url}}\n\n如果你不认识该组织，请忽略此邮件。\n"
	}
	return applyTemplate(body, values)
}

func applyTemplate(template string, values map[string]string) string {
	for key, value := range values {
		template = strings.ReplaceAll(template, "{{"+key+"}}", value)
	}
	return template
}

func applicationTypeLabel(applicationType string) string {
	if applicationType == "membership" {
		return "成员加入申请"
	}
	return "服务器白名单申请"
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
