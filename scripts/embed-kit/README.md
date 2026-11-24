# Embed test kit

A blank website on another origin than ChatGameLab, to try an embedded game
on, with the callback a verified page registers. Standard library Python, no
install.

```sh
make embed-kit                       # http://localhost:8000, snippets from http://localhost:7100
python3 scripts/embed-kit/kit.py --port 8000 --platform http://localhost:7100
```

1. Run the platform (`./schemaf.sh dev all`) and turn embedding on for a game:
   on its Sharing page, or from the browser console on the platform, signed in:

   ```js
   const keys = await (await fetch('/api/apikeys')).json();
   const share = keys.shares.find(s => s.self && s.mayShare).id;
   const r = await fetch('/api/games/GAME-ID/embed', {method: 'PUT',
     headers: {'Content-Type': 'application/json'},
     body: JSON.stringify({enabled: true, mode: 'any', sponsorKeyShareId: share})});
   (await r.json()).embed.token
   ```

   For a verified page, `mode: 'verified'` with `verifiedPageId` (the page's
   `id` in the Websites list).
2. Open http://localhost:8000, paste the embed snippet (or just the token) and
   press **Run**. The kit hands its CSS box over as the host's CSS
   (`docs/embed-styling.md`).
3. Resize the stage: a small frame shows only the controls, a medium one the
   conversation, a large one the whole game.

## A verified page

- Register a page in My organisation → Websites: the page
  `http://localhost:8000/`, the callback `http://localhost:8000/callback`.
  Plain http to the own machine is accepted in development only.
- Embed a game of that organisation for the page, fill in a player ID and
  press **Run**: the platform asks the kit's callback, which answers as the
  page says (yes, no, or nothing for 8 seconds; the platform waits 5) and
  lists what it was asked.
- The same player ID plays as the same guest; another player ID is another
  guest. Without a player ID the game does not start.
