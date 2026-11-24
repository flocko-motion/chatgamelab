# Embed test kit

A blank website on another origin than ChatGameLab, to try an embedded game
on, with the callback a verified page registers. Standard library Python, no
install.

```sh
make embed-kit                       # http://localhost:8000, snippets from http://localhost:7100
python3 scripts/embed-kit/kit.py --port 8000 --platform http://localhost:7100
```

A game embeds only on its organisation's verified page, once an admin has
approved it:

1. Run the platform (`./schemaf.sh dev all`). Sign in as the head of an
   organisation and make the game in one of its workshops.
2. In My organisation → Websites, register the page `http://localhost:8000/`
   with the callback `http://localhost:8000/callback`. Plain http to the own
   machine is accepted in development only.
3. As an admin, approve the page under Admin → Organisations, "Pages that
   embed games".
4. On the game's Sharing page, turn on embedding, pick the key that pays and
   copy the snippet.
5. Run `make embed-kit`, open http://localhost:8000, paste the snippet, fill
   in a player ID and press **Run**. The kit hands its CSS box over as the
   host's CSS (`docs/embed-styling.md`).
6. Resize the stage: a small frame shows only the controls, a medium one the
   conversation, a large one the whole game.

## The callback

- Each Run asks the kit's callback, which answers as the page says (yes, no,
  or nothing for 8 seconds; the platform waits 5) and lists what it was
  asked.
- Until the page is approved, and after its address or callback changes,
  nothing starts.
- The same player ID plays as the same guest; another player ID is another
  guest. Without a player ID the game does not start.
