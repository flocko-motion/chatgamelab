// package: mailtest / send one test e-mail with the configured SMTP settings
// type:    cli
// job:     "cgl mailtest <to>" sends a short test mail via the SMTP_* settings, so a new mail account can be checked locally or on the server.
// limits:  no retries; never prints the password.
package mailtest

import (
	"fmt"
	"os"
	"strings"

	"cgl/functional"
	"cgl/mail"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var envFile string

var Cmd = &cobra.Command{
	Use:   "mailtest <recipient>",
	Short: "Send a test e-mail with the SMTP_* settings",
	Long: "Sends one short test e-mail to <recipient> using SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD and SMTP_FROM.\n" +
		"With --env-file the settings from that file take precedence over the environment and .env.",
	Args:         cobra.ExactArgs(1),
	RunE:         run,
	SilenceUsage: true,
}

func init() {
	Cmd.Flags().StringVar(&envFile, "env-file", "", "file with SMTP_* settings that overrides the environment")
}

func run(cmd *cobra.Command, args []string) error {
	if envFile != "" {
		if err := godotenv.Overload(envFile); err != nil {
			return fmt.Errorf("reading %s: %w", envFile, err)
		}
	}
	host := strings.TrimSpace(functional.EnvOrDefault("SMTP_HOST", ""))
	if host == "" {
		return fmt.Errorf("SMTP_HOST is empty, nothing would be sent")
	}
	password := "(empty)"
	if functional.EnvOrDefault("SMTP_PASSWORD", "") != "" {
		password = "(set)"
	}
	fmt.Printf("Server:   %s:%s\n", host, functional.EnvOrDefault("SMTP_PORT", "587"))
	fmt.Printf("Login:    %s, password %s\n", functional.EnvOrDefault("SMTP_USER", "(none)"), password)
	fmt.Printf("From:     %s\n", functional.EnvOrDefault("SMTP_FROM", "(default)"))
	fmt.Printf("To:       %s\n", args[0])

	err := mail.Send(mail.Message{
		To:      args[0],
		Subject: "ChatGameLab Testmail",
		Body: "Diese Mail wurde mit \"cgl mailtest\" verschickt.\n" +
			"Wenn sie angekommen ist, funktioniert der Mailversand von ChatGameLab.",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFAILED: %v\n", err)
		return err
	}
	fmt.Println("\nOK: the mail server accepted the mail.")
	return nil
}
