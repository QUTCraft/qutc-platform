package mailadapter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestDisabledSenderIsExplicit(t *testing.T) {
	sender, err := New(Config{Driver: "disabled"})
	if err != nil {
		t.Fatalf("create disabled sender: %v", err)
	}
	status := sender.Status()
	if status.Enabled || status.Configured || status.Driver != "disabled" {
		t.Fatalf("disabled status = %+v", status)
	}
	if !errors.Is(sender.SendInvitation(context.Background(), InvitationMessage{}), ErrDisabled) {
		t.Fatal("disabled sender did not return ErrDisabled")
	}
}

func TestSMTPSenderDeliversInvitation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for fake SMTP: %v", err)
	}
	defer listener.Close()
	messageReceived := make(chan string, 1)
	serverError := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverError <- acceptErr
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		writeReply := func(reply string) error {
			if _, writeErr := writer.WriteString(reply + "\r\n"); writeErr != nil {
				return writeErr
			}
			return writer.Flush()
		}
		if replyErr := writeReply("220 fake-smtp ESMTP"); replyErr != nil {
			serverError <- replyErr
			return
		}
		var message strings.Builder
		inData := false
		for {
			line, readErr := readSMTPLine(reader)
			if readErr != nil {
				serverError <- readErr
				return
			}
			if inData {
				if line == "." {
					inData = false
					messageReceived <- message.String()
					if replyErr := writeReply("250 queued"); replyErr != nil {
						serverError <- replyErr
						return
					}
					continue
				}
				message.WriteString(line)
				message.WriteString("\n")
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				if replyErr := writeReply("250 fake-smtp"); replyErr != nil {
					serverError <- replyErr
					return
				}
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				if replyErr := writeReply("250 ok"); replyErr != nil {
					serverError <- replyErr
					return
				}
			case line == "DATA":
				inData = true
				if replyErr := writeReply("354 end with dot"); replyErr != nil {
					serverError <- replyErr
					return
				}
			case line == "QUIT":
				_ = writeReply("221 bye")
				return
			default:
				serverError <- fmt.Errorf("unexpected SMTP command %q", line)
				return
			}
		}
	}()

	address := listener.Addr().(*net.TCPAddr)
	sender, err := New(Config{
		Driver:      "smtp",
		Host:        "127.0.0.1",
		Port:        address.Port,
		FromAddress: "noreply@example.test",
		FromName:    "QUTCraft",
		Security:    "none",
		Timeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("create SMTP sender: %v", err)
	}
	err = sender.SendInvitation(context.Background(), InvitationMessage{
		RecipientEmail: "member@example.test",
		Organization:   "QUTCraft Commons",
		Role:           "member",
		InvitationURL:  "https://portal.example.test/invite/token",
		ExpiresAt:      time.Date(2026, time.August, 4, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("send invitation: %v", err)
	}
	select {
	case received := <-messageReceived:
		if !strings.Contains(received, "member@example.test") || !strings.Contains(received, "https://portal.example.test/invite/token") {
			t.Fatalf("SMTP message omitted invitation details: %s", received)
		}
	case err := <-serverError:
		t.Fatalf("fake SMTP server: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("fake SMTP server did not receive message")
	}
}

func TestSMTPConfigurationValidation(t *testing.T) {
	base := Config{
		Driver:      "smtp",
		Host:        "smtp.example.test",
		Port:        587,
		FromAddress: "noreply@example.test",
		Security:    "starttls",
		Timeout:     time.Second,
	}
	if _, err := New(base); err != nil {
		t.Fatalf("valid SMTP config rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"host":        func(cfg *Config) { cfg.Host = "" },
		"port":        func(cfg *Config) { cfg.Port = 0 },
		"sender":      func(cfg *Config) { cfg.FromAddress = "invalid" },
		"security":    func(cfg *Config) { cfg.Security = "magic" },
		"credentials": func(cfg *Config) { cfg.Username = "user"; cfg.Password = "" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			mutate(&cfg)
			if _, err := New(cfg); err == nil {
				t.Fatal("invalid SMTP config was accepted")
			}
		})
	}
}

func TestBuildInvitationMessageSanitizesHeaders(t *testing.T) {
	raw := buildInvitationMessage(
		Config{FromAddress: "noreply@example.test", FromName: "QUTCraft"},
		"member@example.test",
		InvitationMessage{
			Organization:  "Example\r\nBcc: attacker@example.test",
			Role:          "editor",
			InvitationURL: "https://portal.example.test/invite/token",
			ExpiresAt:     time.Date(2026, time.August, 4, 8, 0, 0, 0, time.UTC),
		},
	)
	if strings.Contains(raw, "\r\nBcc:") {
		t.Fatal("organization text injected a message header")
	}
	if !strings.Contains(raw, "https://portal.example.test/invite/token") || !strings.Contains(raw, "编辑者") {
		t.Fatalf("invitation message omitted required details: %s", raw)
	}
}
