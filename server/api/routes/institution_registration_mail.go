// package: routes / organisation registration mails
// type:    logic
// job:     composes the confirmation-link mail and the consent confirmation mail (person + copy to the operator address).
// limits:  sending itself lives in mail; the declaration text comes from legal.
package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	_ "time/tzdata" // dates in mails use Europe/Berlin even without system tzdata

	"cgl/legal"
	"cgl/log"
	"cgl/mail"
	"cgl/obj"
)

// InstitutionConfirmPath is the web route that reads the token from the link.
const InstitutionConfirmPath = "/organization/confirm"

var berlin = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func sendApplicationLinkMail(app *obj.InstitutionApplication, token string) {
	if app.Email == nil {
		return
	}
	link := fmt.Sprintf("%s%s?token=%s", mail.PublicURL(), InstitutionConfirmPath, url.QueryEscape(token))
	name := ""
	if app.PersonName != nil {
		name = *app.PersonName
	}
	body := fmt.Sprintf(`Hallo %s,

du hast „%s“ als Organisation im ChatGameLab registriert.

Bitte bestätige deine E-Mail-Adresse über diesen Link:
%s

Der Link ist 48 Stunden gültig. Erst nach der Bestätigung wird die Organisation angelegt und du wirst Organisationsleitung.

Falls du das nicht warst, kannst du diese Mail ignorieren.

Dein ChatGameLab-Team
JFF – Jugend Film Fernsehen e.V.
`, name, app.InstitutionName, link)

	if err := mail.Send(mail.Message{
		To:      *app.Email,
		Subject: "ChatGameLab: Bitte bestätige die Registrierung deiner Organisation",
		Body:    body,
	}); err != nil {
		log.Error("confirmation link mail failed", "application_id", app.ID, "error", err)
	}
}

// consentMailBody is the confirmation with the full declaration text, sent to the person and as a copy.
func consentMailBody(c *obj.InstitutionConsent, d legal.Declaration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Hallo %s,\n\n", c.PersonName)
	fmt.Fprintf(&b, "die Organisation „%s“ ist im ChatGameLab freigeschaltet. Du bist Organisationsleitung und kannst jetzt Workshops anlegen.\n\n", c.InstitutionName)
	b.WriteString("Du hast im Namen deiner Einrichtung der folgenden Einverständniserklärung zugestimmt.\n\n")
	b.WriteString("Einrichtung:      " + c.InstitutionName + "\n")
	b.WriteString("Träger:           " + c.Operator + "\n")
	b.WriteString("Anschrift:        " + c.Street + ", " + c.PostalCode + " " + c.City + "\n")
	b.WriteString("Erklärt von:      " + c.PersonName + " (" + c.PersonFunction + "), " + c.Email + "\n")
	b.WriteString("Datum:            " + c.CreatedAt.In(berlin).Format("02.01.2006, 15:04 Uhr") + "\n")
	b.WriteString("Version:          " + d.Version + " (Stand " + formatDate(d.Date) + ")\n")
	b.WriteString("Prüfsumme:        " + d.Hash + "\n\n")
	b.WriteString("----------------------------------------------------------------------\n\n")
	b.WriteString(d.Text)
	b.WriteString("\n\n----------------------------------------------------------------------\n\n")
	b.WriteString("Dein ChatGameLab-Team\nJFF – Jugend Film Fernsehen e.V.\n")
	return b.String()
}

func formatDate(isoDate string) string {
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("02.01.2006")
}

func sendConsentConfirmationMails(c *obj.InstitutionConsent) {
	d, ok := legal.DeclarationByVersion(c.DeclarationVersion)
	if !ok {
		log.Error("consent mail: unknown declaration version", "version", c.DeclarationVersion)
		return
	}
	subject := "ChatGameLab: Organisation „" + c.InstitutionName + "“ freigeschaltet – Einverständniserklärung"
	body := consentMailBody(c, d)

	if err := mail.Send(mail.Message{To: c.Email, Subject: subject, Body: body}); err != nil {
		log.Error("consent confirmation mail failed", "institution_id", c.InstitutionID, "error", err)
	}
	if notify := mail.NotifyAddress(); notify != "" {
		if err := mail.Send(mail.Message{To: notify, Subject: "Kopie: " + subject, Body: body}); err != nil {
			log.Error("consent copy mail failed", "institution_id", c.InstitutionID, "error", err)
		}
	}
}
