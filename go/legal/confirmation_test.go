package legal

import (
	"strings"
	"testing"
	"time"
)

// A consent is confirmed to the person in their language and, as a copy, to
// the operator in German; both carry the whole declaration without Markdown.
func TestConfirmations(t *testing.T) {
	c := Consent{
		PersonName: "Erika Muster", PersonFunction: "Leitung", Email: "erika@example.org",
		Organisation: "JZ Nord", Operator: "Stadt", Street: "Weg 1", PostalCode: "80331", City: "München",
		At: time.Date(2026, 11, 20, 9, 30, 0, 0, time.UTC), Version: CurrentDeclaration().Version,
		NewOrganisation: true,
	}
	msgs := Confirmations(c, "en", "copy@example.org")
	if len(msgs) != 2 {
		t.Fatalf("want 2 mails, got %d", len(msgs))
	}
	person, operator := msgs[0], msgs[1]
	if person.To != "erika@example.org" || !strings.Contains(person.Subject, "“JZ Nord” is active") {
		t.Errorf("mail to the person: %q %q", person.To, person.Subject)
	}
	if !strings.Contains(person.Body, "binding in German") {
		t.Error("an English mail says the declaration follows in German")
	}
	if operator.To != "copy@example.org" || !strings.HasPrefix(operator.Subject, "Kopie: ChatGameLab: Organisation „JZ Nord“ freigeschaltet") {
		t.Errorf("copy: %q %q", operator.To, operator.Subject)
	}
	for _, m := range msgs {
		if !strings.Contains(m.Body, "10. Ich nehme zustimmend zur Kenntnis") || strings.Contains(m.Body, "**") {
			t.Errorf("%s: the declaration is missing or carries Markdown", m.To)
		}
		if !strings.Contains(m.Body, CurrentDeclaration().Hash) {
			t.Errorf("%s: no checksum", m.To)
		}
	}
	if !strings.Contains(operator.Body, "20.11.2026, 10:30 Uhr") {
		t.Error("the date is Berlin's")
	}

	renewed := Confirmations(Consent{Organisation: "JZ Nord", Version: "1.0"}, "de", "")
	if len(renewed) != 1 || !strings.Contains(renewed[0].Subject, "Einverständniserklärung für „JZ Nord“ bestätigt (Version 1.0)") {
		t.Errorf("renewed consent without a copy: %+v", renewed)
	}
}
