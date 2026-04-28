# Jugendschutz bei ChatGameLab

## Warum zusätzlicher Jugendschutz?

Die KI-Anbieter (z.B. OpenAI) haben eigene Schutzmechanismen. Als pädagogische Plattform wollen wir darüber hinaus Verantwortung übernehmen und einen eigenen, zusätzlichen Jugendschutz aufsetzen. Dieser wird bei jeder KI-Interaktion als verbindliche Regel mitgegeben.

## Schutzstufen

Es gibt drei Quellen für Jugendschutz-Constraints, die in einer festen Kaskade ausgewertet werden:

1. **Workshop-Constraint** — vom Workshopleiter pro Workshop konfigurierbar.
2. **Orga-Constraint** — vom Organisationsleiter pro Organisation konfigurierbar.
3. **Site-Constraint** — vom Admin pro Altersgruppe (U13 / U13p / U18) konfigurierbar. Wird bei der Site-Installation einmalig gesetzt und ist immer befüllt.

### Site-weite Defaults (Beispiele)

#### 13–17, ohne Zustimmung der Eltern (U13)

> Erlaube spürbare Konflikte und stärkere Spannung; greife bei realistischer, detaillierter oder eskalierender Gewalt ein und verhindere Verherrlichung und Überforderung.

#### 13–17, mit Zustimmung der Eltern (U13p) und 18+ (U18)

> Erlaube realistische, auch deutlich intensivere Inhalte; unterbinde exzessive oder verherrlichte Gewalt sowie extreme, entwürdigende oder diskriminierende Darstellungen konsequent.

(Standardmäßig identisch; können vom Admin getrennt überschrieben werden.)

#### Unter 13

> Strengster Constraint. Nutzer unter 13 dürfen kein eigenes Konto anlegen — sie nutzen ChatGameLab nur über eine Jugendeinrichtung.

## Wer bekommt welchen Constraint?

Der aktive Constraint wird **pro Spielzug live** aufgelöst. Es gilt immer **genau ein** Constraint. Ändert sich die Rolle des Nutzers, ändert sich automatisch der Constraint ab dem nächsten Spielzug.

### Angemeldete Nutzer

Einheitliche Kaskade — die erste **nicht-leere** Stufe gewinnt:

1. Workshop-Constraint (wenn der Nutzer in einem Workshop-Kontext ist)
2. Orga-Constraint (wenn der Nutzer einer Organisation angehört)
3. Site-Constraint nach Altersgruppe des Nutzers

Da Site-Constraints immer befüllt sind, erhalten **angemeldete Nutzer immer einen Constraint** — niemals leer.

| Rolle | Effekt |
|---|---|
| **Head / Staff** (Orga-Personal) | Workshop (falls aktiv) → Orga → Site nach Alter |
| **Participant** (Workshop-Teilnehmer, auch via Link beigetreten) | Workshop → Orga → Site nach Alter |
| **Individual** (freier Nutzer ohne Orga) | direkt Site nach Alter |

**Hintergrund:** Die Site-Altersgruppen sind primär für `individual` gedacht. Sobald ein Nutzer Teil einer Orga ist, übernimmt die Orga die Kontrolle — wenn sie einen Constraint setzt, gilt der, unabhängig vom Alter des Nutzers. Setzt die Orga keinen Constraint, fällt der Schutz auf die Site-Altersregel zurück.

### Gäste (nicht angemeldet, Spiel via geteiltem Link)

Eigene Kaskade — die erste **nicht-leere** Stufe gewinnt:

1. Workshop-Constraint (aus dem Share, falls das Spiel zu einem Workshop gehört)
2. Orga-Constraint (aus dem Share, falls das Spiel zu einer Orga gehört)
3. **Constraint des Autors** — die `ResolveUserConstraint`-Kaskade (Workshop → Orga → Site nach Alter) angewandt auf den Account, der den Share erstellt hat.

Da Site-Constraints für jede Altersgruppe gesetzt sind, endet die Autoren-Kaskade in Stufe 3 immer mit einem Constraint. Gäste erhalten somit ebenfalls **immer** einen Constraint — den, den der Autor selbst beim Spielen bekäme.

Begründung: Wir kennen den Spieler nicht, also gilt ersatzweise der Schutz, den derjenige für sich selbst gewählt hat, der das Spiel veröffentlicht hat. Öffnet hingegen ein **angemeldeter** Nutzer einen geteilten Link, läuft die Auflösung über die Kaskade für angemeldete Nutzer — sein eigener Constraint, nicht der des Autors.

## Registrierungsablauf

### 1. Nutzertyp wählen

- **"Anmeldung als Nutzer"** — weiter zur Altersabfrage.
- **"Anmeldung als Fachkraft / Organisation"** — Informationen zur Registrierung als Organisation (Kontakt per E-Mail). Die Person wird gebeten, sich zunächst als normaler Nutzer zu registrieren.

### 2. Altersabfrage (Selbstauskunft)

1. **Unter 13 Jahre** — kein eigenes Konto möglich; Nutzung nur über eine Jugendeinrichtung.
2. **13–17 Jahre, ohne Zustimmung der Eltern** — strengere Schutzeinstellungen (U13).
3. **13–17 Jahre, mit Zustimmung der Eltern** — Standard-Schutzeinstellungen (U13p).
4. **18 Jahre oder älter** — Standard-Schutzeinstellungen (U18).

## Selbstauskunft statt technische Altersverifikation

Eine technische Altersverifikation (z.B. über den Besitz eines API-Keys) ist rechtlich nicht stichhaltig. Die Selbstauskunft ist der rechtlich sauberere Weg: Wer falsche Angaben zu seinem Alter macht, setzt sich selbst ins Unrecht.

## Code-Referenz

- `ResolveUserConstraint` in `server/db/game.go` — Kaskade für angemeldete Nutzer.
- `ResolveShareConstraint` in `server/db/game.go` — Kaskade für Gäste via Share-Token.
- Aufrufer: `server/game/session_creation.go` (angemeldet), `server/game/guest.go` (Gast). Auflösung passiert pro Spielzug.
