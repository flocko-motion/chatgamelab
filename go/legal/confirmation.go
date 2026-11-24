package legal

import (
	"strings"
	"time"
	_ "time/tzdata" // dates in mails are Berlin's, even without the system's tzdata

	"cgl/lang"
	"cgl/mail"
)

// Consent is one consent to the declaration, as its confirmation tells it:
// who agreed, for which organisation, when, and to which version.
type Consent struct {
	PersonName      string
	PersonFunction  string
	Email           string
	Organisation    string
	Operator        string
	Street          string
	PostalCode      string
	City            string
	At              time.Time
	Version         string
	NewOrganisation bool // the consent registered the organisation; otherwise it renewed an existing one's
}

// Site is who sends the confirmation: the site's name and its operator,
// as the site's branding says.
type Site struct {
	Name     string
	Operator string
}

// berlin is the time zone of the dates in mails.
var berlin = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// Confirmations are the mails that confirm c: to the person who agreed, in
// their language, and a copy in German to the operator's address (none when
// it is empty). Each carries the full declaration.
func Confirmations(c Consent, site Site, language, notify string) []mail.Message {
	msgs := []mail.Message{confirmation(c, site, language, c.Email)}
	if notify != "" {
		forOperator := confirmation(c, site, BindingLanguage, notify)
		forOperator.Subject = lang.Text(BindingLanguage, "mail.consent.copySubject", map[string]string{"subject": forOperator.Subject})
		msgs = append(msgs, forOperator)
	}
	return msgs
}

// confirmation is c's confirmation in language, to the address to.
func confirmation(c Consent, site Site, language, to string) mail.Message {
	d, ok := DeclarationByVersion(c.Version)
	if !ok {
		d = CurrentDeclaration() // a consent names a published version
	}
	vars := map[string]string{
		"name":         c.PersonName,
		"organisation": c.Organisation,
		"version":      d.Version,
		"date":         formatDate(d.Date),
		"siteName":     site.Name,
		"operatorName": site.Operator,
	}
	text := func(key string) string { return lang.Text(language, "mail.consent."+key, vars) }

	subject, opening := text("subjectRenewed"), text("renewed")
	if c.NewOrganisation {
		subject, opening = text("subjectRegistered"), text("registered")
	}

	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	line(text("greeting"))
	line("")
	line(opening)
	line("")
	line(text("agreed"))
	if language != BindingLanguage {
		line(text("bindingInGerman"))
	}
	line("")
	facts := [][2]string{
		{text("organisation"), c.Organisation},
		{text("operator"), c.Operator},
		{text("address"), c.Street + ", " + c.PostalCode + " " + c.City},
		{text("declaredBy"), c.PersonName + " (" + c.PersonFunction + "), " + c.Email},
		{text("date"), lang.Text(language, "mail.consent.dateValue", map[string]string{"date": c.At.In(berlin).Format("02.01.2006, 15:04")})},
		{text("version"), text("versionValue")},
		{text("checksum"), d.Hash},
	}
	for _, f := range facts {
		line(padded(f[0]+":", 18) + f[1])
	}
	line("")
	line(rule)
	line("")
	line(d.Title())
	line(d.Body())
	line("")
	line(rule)
	line("")
	line(text("signature"))
	return mail.Message{To: to, Subject: subject, Body: b.String()}
}

// rule separates the declaration from the rest of a mail.
const rule = "----------------------------------------------------------------------"

// padded is s, filled with spaces to width characters.
func padded(s string, width int) string {
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s + " "
}

// formatDate is a YYYY-MM-DD date as DD.MM.YYYY.
func formatDate(isoDate string) string {
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("02.01.2006")
}
