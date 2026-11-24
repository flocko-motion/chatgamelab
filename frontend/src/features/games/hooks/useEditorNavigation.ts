import { useState } from "react";
import { editorPages, pageDone, type EditorPage } from "../lib/editorPages";
import type { GameEditor } from "./useGameEditor";
import { useGenreFields, useVoiceSlots } from "./useGenreFields";

/** Where the editor is: the genre question, the overview, a section, or deleting. */
export type EditorView =
  | { kind: "genre" }
  | { kind: "overview" }
  | { kind: "page"; page: EditorPage }
  | { kind: "delete" };

/**
 * The editor's way around: a new game from nothing starts with the genre
 * question, picking a genre opens Card, and from then on the overview opens
 * any section in any order and each returns to it. A section is green once
 * its required fields are filled in (one that requires nothing, once it has
 * been opened); while creating, the first section not yet green is the next
 * one, and the game can be created once every section is green.
 */
export function useEditorNavigation(editor: GameEditor) {
  const shows = useGenreFields(editor.values.genre);
  const speaks = useVoiceSlots(editor.values.genre).length > 0;
  const pages = editorPages(!editor.creating, shows, speaks);
  const [genreChosen, setGenreChosen] = useState(false);
  const [chosen, setChosen] = useState<EditorView>({ kind: "overview" });
  const [opened, setOpened] = useState<Set<EditorPage>>(new Set());

  const askGenre = editor.fresh && !genreChosen;
  // A section the genre does not have, or deleting what may not be deleted,
  // falls back to the overview.
  let view: EditorView = askGenre ? { kind: "genre" } : chosen;
  if (view.kind === "page" && !pages.includes(view.page)) view = { kind: "overview" };
  const mayDelete = !editor.creating && !!editor.game?.mayDelete;
  if (view.kind === "delete" && !mayDelete) view = { kind: "overview" };

  const done = (page: EditorPage) => pageDone(page, editor.values, shows, opened.has(page), !editor.creating);
  const next = pages.find((page) => !done(page)) ?? null;

  const open = (page: EditorPage) => {
    setOpened((seen) => new Set(seen).add(page));
    setChosen({ kind: "page", page });
  };

  return {
    view,
    pages,
    shows,
    done,
    /** The section to do next while creating; null when all are green. */
    next: editor.creating ? next : null,
    allDone: next === null,
    mayDelete,
    open,
    /** Back to the overview, from a section or from deleting. */
    back: () => setChosen({ kind: "overview" }),
    openDelete: () => setChosen({ kind: "delete" }),
    /** The genre is picked: the game's card comes next. */
    pickGenre: (genre: string) => {
      editor.set("genre", genre);
      setGenreChosen(true);
      open("card");
    },
  };
}

export type EditorNavigation = ReturnType<typeof useEditorNavigation>;
