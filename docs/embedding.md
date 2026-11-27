# Embedding a game on your website

A game embedded on your website plays in a frame from ChatGameLab: just the
game, with nothing of ChatGameLab around it. Your page decides who may play,
through a callback ChatGameLab asks before each start. Without a callback that
says yes, no game starts. Styling the game with your own CSS is optional.

Your page never sees the conversation: it stays inside the frame. The
microphone prompt names ChatGameLab.

This guide is for the operator of the page. An embed needs four things:

1. the page registered and approved (part 1),
2. the snippet on the page, with your IDs filled in (part 2),
3. the callback on your server, answering yes (part 3),
4. the notices on your page (part 4).

Styling (part 5) comes on top.

## 1. Before the first start

A game embeds only on its organisation's **verified page**, once an admin has
approved it.

1. **Register the page.** The organisation's head or staff register it under
   My organisation → Websites, with two addresses:
   - the **page address**, where the game is shown, such as
     `https://example.org/play`;
   - the **callback address**, which ChatGameLab asks before each start, such
     as `https://example.org/cgl/callback`. Your server answers there; part 3
     says what it receives and answers.

   Both are on the same site (scheme and host), `https` or plain `http`;
   which sites may show the game is the admin's decision. An organisation
   has one verified page.
   Registering gives the page its **page ID**, 16 hex characters, which
   ChatGameLab sends with every callback. The page ID stays when the
   addresses change, so the page's code keeps working.
2. **Approval.** An admin approves the page's site. The approval holds for the
   scheme and host: any port and path on that site may show the game and
   answer the callback. A change of scheme or host needs approving again;
   `http://example.org` and `https://example.org` are two sites, as are
   `www.example.org` and `example.org`.
3. **Turn embedding on.** On the game's Sharing page, turn on embedding, pick
   the verified page and the key that pays for the plays, and copy the
   snippet. The snippet holds the game's **embed token**. The token stays
   the same for good, also when embedding is off for a while.

Until the page is approved, the game starts nowhere. The token alone starts
nothing: every start needs your callback's yes (part 3).

## 2. The snippet

You copy the snippet from the game's Sharing page ("Copy snippet"): it holds
your game's token and the ChatGameLab address already. It looks like this:

*Example:*

```html
<style data-cgl>
  /* your CSS */
</style>
<div data-cgl-embed data-token="Xq3…" data-player="YOUR-VISITOR-ID" data-request="YOUR-REQUEST-ID" style="height: 32rem"></div>
<script src="https://…/api/engine/playback-engine/embed.js" async></script>
```

Put it where the game should appear, then fill in what is yours:

- **`YOUR-VISITOR-ID` and `YOUR-REQUEST-ID`**: your server writes its ID of
  the visitor and of this page view into each page it delivers (see below).
  These two placeholders are the part of the snippet you always change. Your
  callback gets both back and decides on them (part 3).
- **The height**: give the element the height you want.

The `<style data-cgl>` block stays empty unless you style the game (part 5).

The loader script turns each `[data-cgl-embed]` element into the game's frame,
filling the element, and hands your `<style data-cgl>` blocks over to it.
Several games on one page each get their own element and share one loader
script.

### Attributes

The snippet carries `data-token`, `data-player` and `data-request`, and
`data-mode` when a way to play is preselected on the Sharing page. The others
you add yourself.

| Attribute on `[data-cgl-embed]` | Required | Meaning |
|---|---|---|
| `data-token` | yes | The game's embed token, set by the Sharing page. |
| `data-player` | yes | Your ID of the visitor. The same player ID plays as the same guest; another ID is another guest. |
| `data-request` | yes | Your ID of this page view. Your callback checks it, so a copied snippet starts nothing. |
| `data-lang` | no | The player's language, such as `de`; the browser's otherwise. |
| `data-mode` | no | `live` or `rounds`, for a game that plays both ways: the player is not asked. Where only one way works, that one plays. |
| `data-css` | no | An `https` address of a stylesheet, loaded as well as the `<style data-cgl>` blocks. |
| `data-title` | no | The frame's title for screen readers; "Game" otherwise. |

**Player and request IDs** are your own: up to 128 characters of letters,
digits and `.` `_` `:` `-`. Anything else, or a missing ID, starts nothing,
and your callback is not asked. Use IDs that say nothing about the visitor,
such as a random ID kept in your session, never a name or an email address.
ChatGameLab keeps the player ID only as a hash with the game's embed link: a
player ID names a different guest in each game.

A request ID is made fresh for each page view; part 3 shows two ways to do
it.

### Starting and reloading

Each frame that opens starts a new game, after asking your callback. Once
started, the frame's address names that game: a reload of the frame carries on
with it without asking again.

## 3. The callback

Every embed needs a callback: games embed only on a verified page, and a
verified page starts a game only on its callback's yes. Before each start,
ChatGameLab asks your callback address whether this visitor may play now:

```http
POST /cgl/callback HTTP/1.1
Host: example.org
Content-Type: application/json

{"pageId": "3f9a1c0b7d2e4a65", "playerId": "v-81c2f0e9", "requestId": "r-5d07a3b1c4e2"}
```

| Field | Meaning |
|---|---|
| `pageId` | Your verified page's page ID. It tells you the request is about your page. |
| `playerId` | The `data-player` of the frame that wants to start. |
| `requestId` | The `data-request` of that frame. |

To let the game start, answer **`200 OK`** with this JSON body:

```json
{"ok": true}
```

Every other answer means no, and the game does not start: another status,
`{"ok": false}`, a body without `"ok": true`, a body that is not JSON. The
player sees that the page did not let the game start.

Rules for the answer:

- **Within 5 seconds.** A slower answer counts as no.
- **No redirects.** A `301` or `302` counts as no: the callback address itself
  answers.
- **On the public internet,** on the page's site. ChatGameLab does not call
  private or local addresses.

### What your callback checks

The callback is where your page decides who may play. A sound check:

1. `pageId` is your page ID.
2. `requestId` is a token of your choice that your server made for a page
   view, recently and for this player (see the patterns below).
3. `playerId` is a token of your choice signifying the visitor that page view
   was for, and that visitor may play (signed in, a member, under a daily
   limit: your rules). This can be an anonymized token (e.g. hash of your
   users id) or even random.

Then answer `{"ok": true}`. Since only your server can make valid request
IDs, a snippet copied to another page starts nothing.

### Ways to build it

Two patterns keep a copied snippet from starting anything. Either fits on any
server; the sketches below show the idea, not a finished implementation.

**Stored request IDs.** When your server renders the page, it makes a random
request ID, writes it into the snippet and remembers it with the player ID for
a minute or two. The callback looks it up, checks the player ID, and forgets
it. A request ID works once.

**Signed request IDs.** Nothing is stored: the request ID carries the time and
a signature over player ID and time, made with a secret only your server
knows, such as `1764067200.9f2c…` (seconds, a dot, a hex HMAC). The callback
recomputes the signature and checks that the time is recent. Within that
window a request ID could start a second game; store the used ones if that
matters.

Either way, **the page holding the snippet is not cached**: a cached page
hands every visitor the same IDs.

#### WordPress

- **Fill in the IDs per page view** with a small shortcode or block that
  prints the snippet, rather than pasting it into the page as static HTML.
  Exclude that page from caching plugins.
- **Player ID:** for a signed-in user, a hash of the user ID with a site
  secret, such as `hash_hmac('sha256', get_current_user_id(), wp_salt())`,
  cut to 32 characters. Visitors who are not signed in get a random ID kept
  in a cookie, or no game, as you choose.
- **Request ID:** `wp_generate_uuid4()`, remembered with
  `set_transient('cgl_' . $request, $player, 2 * MINUTE_IN_SECONDS)`.
- **Callback:** a REST route, `register_rest_route('cgl/v1', '/callback', …)`,
  registered as `https://example.org/wp-json/cgl/v1/callback`. It reads the
  transient, deletes it, compares the player ID, and returns
  `['ok' => true]`. Register the address exactly as WordPress serves it: a
  redirect for a missing or extra slash counts as no.

#### Python

The stored pattern, with Flask; a real server keeps `open_requests` in its
session store or cache rather than in memory:

```python
from flask import Flask, request, jsonify

app = Flask(__name__)
PAGE_ID = "3f9a1c0b7d2e4a65"
open_requests = {}  # request ID -> player ID, filled when the page is rendered

@app.post("/cgl/callback")
def cgl_callback():
    ask = request.get_json(silent=True) or {}
    player = open_requests.pop(ask.get("requestId"), None)
    ok = ask.get("pageId") == PAGE_ID and player is not None and player == ask.get("playerId")
    return jsonify(ok=ok)
```

The signed pattern needs only `hmac.compare_digest` against
`hmac.new(SECRET, f"{player}.{time}".encode(), "sha256").hexdigest()` and a
check that `time` lies within the last two minutes.

## 4. Notices on your page

Your visitors meet the game on your page, so your page tells them what they
need to know before they play.

### Privacy

Your privacy policy names ChatGameLab as a service your page uses and says
what reaches it:

- **The game runs on ChatGameLab.** The frame and its script load from the
  ChatGameLab instance in the snippet, which sees the visitor's IP address
  like any web server.
- **The visitor ID you send.** ChatGameLab stores it hashed with the game's
  token, as the guest the visitor plays as, and passes it to your callback.
  An ID with no meaning outside your page, such as a random one, keeps the
  visitor anonymous to ChatGameLab.
- **What the visitor types or says.** The conversation is stored on
  ChatGameLab as the visitor's game and goes to the AI provider of the key
  that pays for the plays, such as OpenAI or Mistral. Spoken input reaches
  the provider as audio. The organisation that picked the key knows which
  provider it is.

### Youth protection

- **The game talks with an AI.** Your page says so before the game starts:
  the story, the characters and any images are generated as the visitor plays.
- **Who the game is for.** Every embedded game plays under the youth-protection
  rule of its workshop or organisation, or, when neither sets one, the rule
  of the account that first turned embedding on. ChatGameLab knows nothing
  about your visitors' ages, so your page states the age the game is meant
  for, and any age check belongs in your callback.
- **Where to turn.** Your page names a contact for visitors who meet content
  they find harmful.

## 5. Styling

The game lays itself out by the room the frame gives it. You style it with
your own CSS, against the names below. These names are the contract: the
inside of the frame may change, the names stay. Your CSS reaches only the
inside of the frame.

### Handing over your CSS

- **`<style data-cgl>` blocks** on your page: the loader hands every one of
  them to every frame on the page, once the frame is ready.
- **A stylesheet address** in `data-css`. It must be `https`.

Your CSS comes after the game's own styles, so a rule of the same weight wins.

### Sizes

The game lays itself out by the size of its frame:

| Frame | Shows |
|---|---|
| Small: under 15rem wide or 10rem high | The controls alone: one button to talk, or the input |
| Medium: under 40rem wide or 30rem high | The conversation and its controls |
| Large | The game with its pictures and status values |

### Parts

Select parts as `[data-part="…"]`.

| Part | What it is | Shown at |
|---|---|---|
| `status` | The status values | large |
| `status-field` | One status value | large |
| `scene` | The area around the conversation and the input | every size |
| `feed` | The conversation | medium, large |
| `message` | A message of the game | medium, large |
| `player-message` | A line of the player's | medium, large |
| `picture` | A picture in the story | large |
| `system-message` | A line from the game engine, such as why a game is over | medium, large |
| `divider` | The line between turns | large |
| `compose` | The input row | every size |
| `input` | The text field | medium, large |
| `send` | The send button | medium, large |
| `talk` | The button that starts and pauses a voice conversation | every size |
| `hold` | The button held to speak | every size |
| `notice` | What the game says while it connects or waits | every size |
| `preparing` | The steps still running while the game prepares, a line each | every size |
| `launch` | The panel shown when a game cannot start | every size |
| `options` | The choice of how to play, before a game starts | every size |

### Variables

Set them on `:root`.

| Variable | Changes |
|---|---|
| `--cgl-accent` | The colour of the send button's edge and hover, the voice buttons' edge, and the player's lines |
| `--cgl-font` | The font of the page and the messages |
| `--cgl-radius` | The corner radius of messages and the input |
| `--cgl-bg` | The background behind the conversation |
| `--cgl-fg` | The colour of the game's messages |

### What your CSS may load

Fonts and pictures in your CSS (`url()`, `@font-face`) load only from:

- the ChatGameLab site itself,
- `data:` addresses,
- for a game embedded on a verified page, that page's own website.

Stylesheets your CSS imports (`@import`) load only from the ChatGameLab site,
the website of the stylesheet named in `data-css`, and a verified
page's own website.

Anything else is blocked by the browser. This keeps CSS from sending what the
player types to a third site. A font from another site is embedded as a
`data:` address, or served from your own website on a verified page.

### Sizing

Give the element a height, in your CSS or inline, and the frame fills it.
Without one the frame is 36rem high.

### Example

The style block of the copied snippet, filled in:

```html
<style data-cgl>
  :root {
    --cgl-accent: #b4235a;
    --cgl-font: "Atkinson Hyperlegible", sans-serif;
    --cgl-radius: 1.25rem;
    --cgl-bg: #fff8f2;
  }
  [data-part="player-message"] { font-style: italic; }
  [data-part="send"] { background: #b4235a; color: white; }
</style>
```
