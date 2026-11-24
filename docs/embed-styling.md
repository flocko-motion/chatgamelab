# Styling an embedded game

A game embedded on your website plays in a frame from ChatGameLab: just the
game, with nothing of ChatGameLab around it. It lays itself out by the room
the frame gives it. You style it with your own CSS, against the names on this
page. These names are the contract: the inside of the frame may change, the
names stay.

Your CSS reaches only the inside of the frame. Your page never sees the
conversation, and the microphone prompt names ChatGameLab.

## The snippet

The editor's Sharing page gives you the snippet for your game. With the loader
script, which hands your CSS over and sizes the frame:

```html
<style data-cgl>
  :root { --cgl-accent: #0a7d5a; --cgl-font: Georgia, serif; }
  [data-part="input"] { border-radius: 0; }
</style>

<div data-cgl-embed data-token="YOUR-TOKEN" style="height: 32rem"></div>

<script src="https://YOUR-CHATGAMELAB/api/engine/playback-engine/embed.js" async></script>
```

| Attribute on `[data-cgl-embed]` | Meaning |
|---|---|
| `data-token` | The game's embed token. Required. |
| `data-css` | An `https` address of a stylesheet, loaded as well. |
| `data-lang` | The player's language, such as `de`; the browser's otherwise. |
| `data-player`, `data-request` | For a verified page: your ID of the visitor and of this page view. |
| `data-mode` | `live` or `rounds`, for a game that can be played both ways: the player is not asked. Where only one way works, that one plays. |
| `data-title` | The frame's title for screen readers; "Game" otherwise. |

Without the loader, use the frame directly and give it a height:

```html
<iframe
  src="https://YOUR-CHATGAMELAB/api/engine/playback-engine/?new&share=YOUR-TOKEN&css=https%3A%2F%2Fexample.org%2Fgame.css"
  allow="microphone; autoplay"
  style="border: 0; width: 100%; height: 36rem"></iframe>
```

## Handing over your CSS

- **`<style data-cgl>` blocks** on your page: the loader hands every one of
  them to every frame on the page, once the frame is ready.
- **A stylesheet address**: `data-css` with the loader, `css=` in the frame's
  address without it. It must be `https`.

Your CSS comes after the game's own styles, so a rule of the same weight wins.

## Sizes

The game lays itself out by the size of its frame:

| Frame | Shows |
|---|---|
| Small: under 15rem wide or 10rem high | The controls alone: one button to talk, or the input |
| Medium: under 40rem wide or 30rem high | The conversation and its controls |
| Large | The game with its pictures and status values |

## Parts

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

## Variables

Set them on `:root`.

| Variable | Changes |
|---|---|
| `--cgl-accent` | The colour of the send button's edge and hover, the voice buttons' edge, and the player's lines |
| `--cgl-font` | The font of the page and the messages |
| `--cgl-radius` | The corner radius of messages and the input |
| `--cgl-bg` | The background behind the conversation |
| `--cgl-fg` | The colour of the game's messages |

## What your CSS may load

Fonts and pictures in your CSS (`url()`, `@font-face`) load only from:

- the ChatGameLab site itself,
- `data:` addresses,
- for a game embedded on a verified page, that page's own website.

Stylesheets your CSS imports (`@import`) load only from the ChatGameLab site,
the website of the stylesheet named in `css=` or `data-css`, and a verified
page's own website.

Anything else is blocked by the browser. This keeps CSS from sending what the
player types to a third site. A font from another site is embedded as a
`data:` address, or served from your own website on a verified page.

## Sizing

Give the element a height, in your CSS or inline, and the frame fills it.
Without one the frame is 36rem high.

## Example

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

<div data-cgl-embed data-token="YOUR-TOKEN" style="height: 32rem"></div>
<script src="https://YOUR-CHATGAMELAB/api/engine/playback-engine/embed.js" async></script>
```
