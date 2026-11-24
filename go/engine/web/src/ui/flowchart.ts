/**
 * Draws the session's wiring and lights each block as it works.
 *
 * Purely presentational: it is handed topology, phases and costs from the
 * headless core's state and owns none of them. Nothing here decides anything,
 * and the core neither imports this nor knows it exists.
 */
import cytoscape from "cytoscape";
import type { Place } from "../client.js";
import type { Phase } from "../protocol.js";
import type { Topology, TopologyEdge } from "../client.js";
import { layoutCode, parked } from "../layout.js";
import { blockLabel, settingLabel, text, wireLabel } from "../texts.js";

/** Reads a CSS custom property, so the diagram follows the page's theme. */
function token(name: string, fallback: string): string {
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

/**
 * Every class a selection puts on, cleared together: a stale mark from the last
 * selection is worse than none, because it reads as part of this one.
 */
const MARKS = "selected incident into outOf";

export type NodeClickHandler = (name: string) => void;
export type BackgroundClickHandler = () => void;
export type EdgeClickHandler = (from: string, to: string, kind: string) => void;

export class Flowchart {
  #cy: cytoscape.Core | null = null;
  #drawn = false;
  #phases = new Map<string, Phase>();
  #fading = new Map<string, ReturnType<typeof setTimeout>>();
  /**
   * How long a finished block keeps its afterglow: long enough to be seen,
   * short enough that a busy graph is not permanently lit.
   */
  static readonly afterglowMs = 2500;

  constructor(
    private readonly container: HTMLElement,
    private readonly onNodeClick: NodeClickHandler,
    private readonly onBackgroundClick: BackgroundClickHandler,
    private readonly onEdgeClick: EdgeClickHandler,
  ) {}

  /**
   * Draws the graph once. The wiring does not change during a session, so
   * redrawing it would only throw away the layout.
   */
  draw(full: Topology): void {
    if (this.#drawn) return;
    this.#drawn = true;
    const topology = withoutFailures(full);

    const fg = token("--fg", "#1b1a19");
    const dim = token("--dim", "#6b6a68");
    const line = token("--line", "#e2e0dc");
    const bg = token("--bg", "#faf9f7");
    const into = token("--wire-in", "#16a34a");
    const outOf = token("--wire-out", "#dc2626");
    const symbol = token("--gray-6", "#475569");
    // Each group's colour: its line, its calm fill, and the fill it glows with
    // while working, so activity reads as the block itself lighting up.
    const hues = {
      input: group("input", "#0284c7", "#e0f2fe", "#bae6fd"),
      preparation: group("prep", "#0d9488", "#e8f6f4", "#b5e6df"),
      loop: group("block", "#7c3aed", "#f3effe", "#dccffc"),
    };
    const flag = token("--flag", "#db2777");

    // Outputs are not blocks of their own in the drawing: each is a small
    // symbol hanging off the block that feeds it, the way a circuit diagram
    // draws ground, so the eye stays on the blocks that do the work.
    const outputs = new Set(
      topology.nodes.filter((node) => node.role === "sink").map((node) => node.name),
    );
    // The game's settings are no blocks either: each is a gear hanging above
    // the block that reads it, as inputs come in from above.
    const settings = topology.nodes.filter((node) => node.role === "setting");
    const settingNames = new Set(settings.map((node) => node.name));
    const blocks = topology.nodes.filter((node) => !outputs.has(node.name) && !settingNames.has(node.name));
    const inputs = new Set(
      topology.nodes.filter((node) => node.role === "source").map((node) => node.name),
    );
    // What the player typed, shown back to them, is no step of the story.
    const feeds = topology.edges.filter((edge) => outputs.has(edge.to) && !inputs.has(edge.from));
    // The signal that hands the turn back to the player is plumbing, not part
    // of the story's flow, and drawn it reads as the story running backwards.
    // Two wires of one kind between the same blocks, such as the two texts
    // Outline hands the picture prompt, are one connection to a reader.
    const seen = new Set<string>();
    const wires = topology.edges.filter((edge) => {
      if (outputs.has(edge.to) || (edge.kind === "state" && inputs.has(edge.to))) return false;
      // A setting's lead is drawn with its gear.
      if (settingNames.has(edge.from)) return false;
      // Unless a port tells them apart, as a voice and its description are.
      const key = `${edge.from}>${edge.to}:${edge.kind}:${edge.port ?? ""}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });

    // The genre places every block by hand. One the layout misses is parked
    // below the drawing and marked, until a developer gives it a place.
    const places = new Map<string, Place>();
    for (const node of blocks) if (node.place) places.set(node.name, node.place);
    const unplaced = blocks.filter((node) => !node.place).map((node) => node.name);
    for (const [name, place] of parked(places, unplaced)) places.set(name, place);

    // Preparation is everything the start waits for, and the start itself;
    // the rest is the game loop. Coloured apart, the two phases of a session
    // read as two parts of the drawing.
    const preparation = preparationOf(topology);

    this.#cy = cytoscape({
      container: this.container,
      // A teaching diagram, not a toy to drag apart; only a developer
      // rearranges it, to copy the layout back into the genre.
      userPanningEnabled: true,
      userZoomingEnabled: true,
      boxSelectionEnabled: false,
      autoungrabify: !topology.dev,
      elements: [
        ...blocks.map((node) => {
          // The player's inputs are drawn as symbols too, but keep their place:
          // the story starts from them.
          const kind = node.role === "source" ? inputKind(node.name, topology.edges) : null;
          const hue = kind
            ? hues.input
            : preparation.has(node.name)
              ? hues.preparation
              : hues.loop;
          return {
            data: {
              id: node.name,
              label: blockLabel(node.name),
              role: node.role,
              ...hue,
              ...(kind ? { icon: icon(`in-${kind}`, hue.hue) } : {}),
            },
            position: { ...places.get(node.name)! },
            classes: [kind ? "input" : "", node.place ? "" : "unplaced"].join(" ").trim(),
          };
        }),
        ...settings.flatMap((node) => {
          const target = topology.edges.find((edge) => edge.from === node.name)?.to;
          return [
            {
              data: {
                id: node.name,
                label: settingLabel(node.field, node.name),
                role: node.role,
                icon: icon("setting", symbol),
                hue: symbol,
                soft: bg,
                glow: bg,
              },
              classes: "setting",
              grabbable: false,
            },
            ...(target
              ? [{ data: { id: `${node.name}:stub`, source: node.name, target }, classes: "stub" }]
              : []),
          ];
        }),
        ...wires.map((edge, index) => ({
          data: {
            id: `e${index}`,
            source: edge.from,
            target: edge.to,
            kind: edge.kind,
            label: wireLabel(edge.from, edge.to, edge.kind, edge.port),
          },
        })),
        ...feeds.flatMap((edge) => {
          const id = symbolId(edge);
          return [
            {
              data: {
                id,
                output: edge.to,
                kind: edge.kind,
                icon: icon(edge.kind, symbol),
                hue: symbol,
                soft: bg,
                glow: bg,
              },
              classes: "output",
              grabbable: false,
            },
            { data: { id: `${id}:stub`, source: edge.from, target: id }, classes: "stub" },
          ];
        }),
      ],
      style: [
        {
          selector: "node",
          style: {
            label: "data(label)",
            "font-size": 9,
            "font-family": "Inter, system-ui, -apple-system, sans-serif",
            color: fg,
            "text-valign": "center",
            "text-halign": "center",
            "text-wrap": "wrap",
            shape: "round-rectangle",
            width: "label",
            height: 22,
            padding: "6px",
            "background-color": "data(soft)",
            "border-width": 1,
            "border-color": "data(hue)",
            // Every style that changes is animated, so a turn reads as motion
            // through the graph rather than as a sequence of stills.
            "transition-property":
              "border-color, border-width, background-color, overlay-opacity",
            // Slow enough that the afterglow fades rather than switches off.
            "transition-duration": 400,
            "transition-timing-function": "ease-out",
            "overlay-color": "data(hue)",
            "overlay-opacity": 0,
            "overlay-padding": 6,
          },
        },
        // The player's input, the preparation and the game loop have their
        // own colours (data(hue), data(soft)), so they are told apart before
        // any label is read. The gate is drawn as a block: where the game
        // starts matters to the engine, not to someone reading a turn.
        {
          selector: "node.output, node.input",
          style: {
            label: "",
            shape: "ellipse",
            width: 24,
            height: 24,
            padding: "0px",
            "border-width": 0,
            "background-opacity": 0,
            "background-image": "data(icon)",
            "background-fit": "contain",
          },
        },
        { selector: "node.input", style: { width: 34, height: 34 } },
        {
          // A game setting: a gear, named above it, as small as an output.
          selector: "node.setting",
          style: {
            label: "data(label)",
            shape: "ellipse",
            width: 20,
            height: 20,
            padding: "0px",
            "font-size": 7,
            color: symbol,
            "text-valign": "top",
            "text-margin-y": -2,
            "border-width": 0,
            "background-opacity": 0,
            "background-image": "data(icon)",
            "background-fit": "contain",
          },
        },
        {
          // A block the genre's layout misses: findable, so it gets placed.
          selector: "node.unplaced",
          style: { "border-style": "dashed", "border-color": flag },
        },
        {
          // The afterglow, declared before working so a block that starts
          // again while still glowing shows as working. Without it a block
          // that finishes in 40 ms is never seen at all.
          selector: "node.recent",
          style: {
            "border-width": 2,
            "background-color": "data(glow)",
            "background-opacity": 0.6,
            "overlay-opacity": 0.08,
          },
        },
        {
          selector: "node.working",
          style: {
            "border-width": 4,
            "background-color": "data(glow)",
            "background-opacity": 1,
            // A halo, so a working block is findable at a glance on a graph
            // with a dozen nodes.
            "overlay-opacity": 0.25,
            "overlay-padding": 8,
          },
        },
        {
          selector: "edge",
          style: {
            label: "data(label)",
            "font-size": 7,
            color: dim,
            "text-background-color": bg,
            "text-background-opacity": 1,
            "text-background-padding": "1px",
            width: 1,
            "line-color": line,
            "target-arrow-color": line,
            "target-arrow-shape": "triangle",
            "arrow-scale": 0.7,
            "curve-style": "bezier",
          },
        },
        {
          // The short lead from a block to its output symbol: no label, no
          // arrowhead, like the stub to a ground symbol.
          selector: "edge.stub",
          style: {
            label: "",
            "target-arrow-shape": "none",
            "curve-style": "straight",
            "line-color": dim,
          },
        },
        // Declared last on purpose: cytoscape resolves a conflict in favour of
        // the later rule, so selection styles have to follow the base ones they
        // are meant to override.
        {
          // A selected block and the edges touching it, so a reader can follow
          // what a stage is connected to while reading its detail.
          selector: "node.selected",
          style: { "border-color": fg, "border-width": 3 },
        },
        {
          selector: "edge.incident",
          style: { "line-color": fg, "target-arrow-color": fg, width: 2, color: fg },
        },
        // Selecting a block asks what reaches it and what it feeds, which are
        // different questions with the same answer shape. Colouring them apart
        // means the direction is read off the picture rather than off the
        // arrowheads, which at this size are two pixels of difference.
        //
        // From the block's own point of view: what comes in, and what goes out.
        {
          selector: "edge.into",
          style: { "line-color": into, "target-arrow-color": into, width: 2, color: into },
        },
        {
          selector: "edge.outOf",
          style: { "line-color": outOf, "target-arrow-color": outOf, width: 2, color: outOf },
        },
      ],
      // The positions are the genre's own, so nothing is laid out here. The
      // layout's own fit is off: it would measure the wires before they are
      // drawn, and #fit does the fitting.
      layout: { name: "preset", fit: false },
    });

    this.#hangOutputs();
    this.#hangSettings();
    // A block moved by a developer takes its symbols along.
    this.#cy.on("position", "node", (event) => {
      if (event.target.hasClass("output") || event.target.hasClass("setting")) return;
      this.#hangOutputs(event.target);
      this.#hangSettings(event.target);
    });

    if (topology.dev) this.#rearrangeable(topology.name);

    this.#cy.on("tap", "node", (event) => {
      // A symbol stands for its output, which is what the details describe.
      const name = event.target.data("output") ?? event.target.id();
      this.select(name);
      this.onNodeClick(name);
    });
    this.#cy.on("tap", "edge", (event) => {
      const edge = event.target;
      this.selectEdge(edge.id());
      this.onEdgeClick(edge.data("source"), edge.data("target"), edge.data("kind"));
    });
    this.#cy.on("tap", (event) => {
      // A tap on the background rather than on a block clears the selection,
      // which is how a reader gets back to the whole session's figures.
      if (event.target !== this.#cy) return;
      this.select(null);
      this.onBackgroundClick();
    });
    const cy = this.#cy;
    this.#fit();
    cy.on("viewport", () => this.#contain());

    // The box changes size when the page stacks or the host resizes the iframe,
    // and a layout fitted to the old box would sit small in a corner of the new.
    new ResizeObserver(() => {
      cy.resize();
      this.#fit();
    }).observe(this.container);
  }

  /** How far below its block an output symbol hangs, and how far apart. */
  static readonly symbolDrop = 40;
  static readonly symbolStep = 30;

  /**
   * Hangs each block's output symbols in a row under it, centred, or only
   * those of one block. Their places follow from the block's, so a layout
   * never has to name them.
   */
  #hangOutputs(only?: cytoscape.NodeSingular): void {
    const cy = this.#cy;
    if (!cy) return;
    const owners = only ? cy.collection(only) : blocksOf(cy);
    cy.batch(() => {
      owners.forEach((owner) => {
        const symbols = owner.outgoers("node.output");
        const { x, y } = owner.position();
        const below = y + owner.outerHeight() / 2 + Flowchart.symbolDrop - 16;
        symbols.forEach((symbol, i) => {
          symbol.position({
            x: x + (i - (symbols.length - 1) / 2) * Flowchart.symbolStep,
            y: below,
          });
        });
      });
    });
  }

  /** How far above its block a setting's gear stands, and how far apart. */
  static readonly settingRise = 34;
  static readonly settingStep = 44;

  /**
   * Hangs each block's settings in a row above it, centred, or only those of
   * one block: inputs come in from above, as the story runs top down.
   */
  #hangSettings(only?: cytoscape.NodeSingular): void {
    const cy = this.#cy;
    if (!cy) return;
    const owners = only ? cy.collection(only) : blocksOf(cy);
    cy.batch(() => {
      owners.forEach((owner) => {
        const gears = owner.incomers("node.setting");
        const { x, y } = owner.position();
        const above = y - owner.outerHeight() / 2 - Flowchart.settingRise + 10;
        gears.forEach((gear, i) => {
          gear.position({
            x: x + (i - (gears.length - 1) / 2) * Flowchart.settingStep,
            y: above,
          });
        });
      });
    });
  }

  /**
   * Lets a developer drag blocks anywhere and copy the layout as Go source:
   * where the blocks stand when the button is pressed is the layout.
   */
  #rearrangeable(genre: string): void {
    const cy = this.#cy;
    if (!cy) return;

    // A parked block that has been moved has a place now.
    cy.on("free", "node", (event) => event.target.removeClass("unplaced"));

    // An icon, named by its tooltip; the tooltip and the icon say how a copy went.
    const button = document.createElement("button");
    button.type = "button";
    button.className = "copy-layout";
    const say = (key: "pipeline.copyLayout" | "pipeline.copied" | "pipeline.copyFailed", glyph: string): void => {
      button.title = text(key);
      button.setAttribute("aria-label", text(key));
      button.innerHTML = `<svg viewBox="0 0 24 24" aria-hidden="true">${glyph}</svg>`;
    };
    say("pipeline.copyLayout", COPY_GLYPH);
    button.addEventListener("click", async () => {
      const layout = new Map(
        blocksOf(cy).map((node) => [node.id(), node.position()] as const),
      );
      const code = layoutCode(genre, layout);
      try {
        await navigator.clipboard.writeText(code);
        say("pipeline.copied", CHECK_GLYPH);
      } catch {
        // A page without clipboard access still hands the layout over.
        console.log(code);
        say("pipeline.copyFailed", COPY_GLYPH);
      }
      setTimeout(() => say("pipeline.copyLayout", COPY_GLYPH), 1500);
    });
    this.container.parentElement?.append(button);
  }

  /**
   * Fits the whole graph into the box, and makes that the furthest a reader can
   * zoom out: any further only shrinks the graph into empty space.
   */
  #fit(): void {
    const cy = this.#cy;
    if (!cy) return;
    cy.minZoom(1e-50);
    // Fitted to the blocks alone: measuring a wire before it is first drawn
    // caches an empty box for it, and the renderer then culls it as off-screen.
    cy.fit(cy.nodes(), Flowchart.heading);
    cy.minZoom(cy.zoom());
    cy.maxZoom(Math.max(cy.zoom(), 3));
  }

  /**
   * Keeps the graph in view. While it fits the box it cannot be pushed past an
   * edge; once zoomed larger than the box, the box always stays covered.
   */
  #contain(): void {
    const cy = this.#cy;
    if (!cy || this.#containing) return;
    // Measured by the blocks alone: the wires run between them, and a wire not
    // yet drawn has no settled geometry, so measuring it moves the box under
    // the correction and the correction never ends.
    const box = cy.nodes().renderedBoundingBox();
    const m = Flowchart.margin;
    const dx = shift(box.x1, box.x2, cy.width(), m, m);
    const dy = shift(box.y1, box.y2, cy.height(), Flowchart.heading, m);
    if (!dx && !dy) return;
    // The correction fires another viewport event, which must not correct again.
    this.#containing = true;
    try {
      cy.panBy({ x: dx, y: dy });
    } finally {
      this.#containing = false;
    }
  }
  #containing = false;

  static readonly margin = 12;
  /** Room at the top for the heading laid over the box. */
  static readonly heading = 32;

  /**
   * Marks one block and the wires touching it, or clears the marking.
   *
   * The two directions are marked apart, and from the block's own point of
   * view: what reaches it, and what it feeds. On a graph with a cycle in it —
   * the observer's correction runs back into the character — that is the
   * difference between reading the loop and guessing at it.
   */
  select(name: string | null): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().removeClass(MARKS);
      if (!name) return;

      // An output is drawn as one symbol under each block that feeds it.
      let node = cy.getElementById(name);
      if (node.empty()) node = cy.nodes(".output").filter((n) => n.data("output") === name);
      if (node.empty()) return;
      node.addClass("selected");
      node.incomers("edge").addClass("into");
      node.outgoers("edge").addClass("outOf");
    });
  }

  /** Marks one wire and the blocks it joins. */
  selectEdge(id: string): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().removeClass(MARKS);
      const edge = cy.getElementById(id);
      if (edge.empty()) return;
      edge.addClass("incident");
      edge.connectedNodes().addClass("selected");
    });
  }

  /**
   * Lights the blocks that are working. What each spent is left to the
   * details pane: in the boxes it is more than a reader takes in.
   */
  update(phases: Record<string, Phase>): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      for (const node of blocksOf(cy)) {
        const id = node.id();
        const phase = phases[id] ?? "ready";
        const previous = this.#phases.get(id);

        if (phase !== previous) {
          this.#phases.set(id, phase);
          node.toggleClass("working", phase === "working");
          if (previous === "working" && phase === "ready") this.#afterglow(node);
        }
      }
    });
  }

  /**
   * Holds a finished block lit briefly. A tool call can complete between two
   * frames, so without this the fastest blocks would be the ones nobody ever
   * sees work.
   */
  #afterglow(node: cytoscape.NodeSingular): void {
    const id = node.id();
    clearTimeout(this.#fading.get(id));
    node.addClass("recent");
    this.#fading.set(
      id,
      setTimeout(() => {
        node.removeClass("recent");
        this.#fading.delete(id);
      }, Flowchart.afterglowMs),
    );
  }
}

/** A group's colours from the page's tokens, with fallbacks. */
function group(name: string, hue: string, soft: string, glow: string) {
  return {
    hue: token(`--node-${name}`, hue),
    soft: token(`--node-${name}-soft`, soft),
    glow: token(`--node-${name}-glow`, glow),
  };
}

/**
 * The blocks the start waits for, followed back through every wire, and the
 * start itself: the session's preparation.
 */
function preparationOf(topology: Topology): Set<string> {
  const gate = topology.nodes.find((node) => node.role === "gate");
  if (!gate) return new Set();
  const found = new Set([gate.name]);
  const queue = [gate.name];
  while (queue.length) {
    const name = queue.pop()!;
    for (const edge of topology.edges) {
      if (edge.to === name && !found.has(edge.from)) {
        found.add(edge.from);
        queue.push(edge.from);
      }
    }
  }
  return found;
}

/** The drawing's blocks, without the output symbols hanging off them. */
function blocksOf(cy: cytoscape.Core): cytoscape.NodeCollection {
  return cy.nodes().filter((node) => !node.hasClass("output") && !node.hasClass("setting"));
}

/** One symbol per feeding block, so an output fed twice hangs under both. */
function symbolId(edge: TopologyEdge): string {
  return `${edge.from}>${edge.to}`;
}

/** Two sheets, one over the other: copy. */
const COPY_GLYPH = '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>';
/** A tick: copied. */
const CHECK_GLYPH = '<path d="M5 12.5l4.5 4.5L19 7.5"/>';

/** Line-art glyphs in a 24-unit box, one per kind of input and output. */
const GLYPHS: Record<string, string> = {
  // The player speaks: a microphone.
  "in-audio":
    '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5.5 11a6.5 6.5 0 0 0 13 0M12 17.5V21M8.5 21h7"/>',
  // A picture there from the start, such as the gallery image: a framed picture.
  "in-image":
    '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 16l5-5 4 4 3-3 6 6"/><circle cx="16" cy="9" r="1.5"/>',
  // A game setting, as its author set it in the editor: a gear.
  setting:
    '<path d="M10.3 4.3c.4-1.8 2.9-1.8 3.4 0a1.7 1.7 0 0 0 2.6 1.1c1.5-.9 3.3.8 2.4 2.4a1.7 1.7 0 0 0 1 2.5c1.8.4 1.8 2.9 0 3.4a1.7 1.7 0 0 0-1 2.6c.9 1.5-.8 3.3-2.4 2.4a1.7 1.7 0 0 0-2.6 1c-.4 1.8-2.9 1.8-3.4 0a1.7 1.7 0 0 0-2.5-1c-1.6.9-3.3-.9-2.4-2.4a1.7 1.7 0 0 0-1.1-2.6c-1.8-.4-1.8-2.9 0-3.4a1.7 1.7 0 0 0 1.1-2.5c-.9-1.6.8-3.3 2.4-2.4a1.7 1.7 0 0 0 2.5-1.1z"/><circle cx="12" cy="12" r="3"/>',
  // The player types: a keyboard.
  "in-text":
    '<rect x="2.5" y="6" width="19" height="12" rx="2"/><path d="M6 10h.01M9.5 10h.01M13 10h.01M16.5 10h.01M8 14h8"/>',
  // A speaker with a sound wave.
  audio:
    '<path d="M4 9h4l5-4v14l-5-4H4z"/><path d="M16.5 8.5a5 5 0 0 1 0 7M19 6a8.5 8.5 0 0 1 0 12"/>',
  // A framed picture with a hill and a sun.
  image:
    '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 16l5-5 4 4 3-3 6 6"/><circle cx="16" cy="9" r="1.5"/>',
  // Two sliders: values the game keeps.
  props: '<path d="M4 8h16M4 16h16"/><circle cx="9" cy="8" r="2.2"/><circle cx="15" cy="16" r="2.2"/>',
  // A speech bubble with lines of text.
  text: '<path d="M4 5h16v11H9l-5 4z"/><path d="M8 9h8M8 12h5"/>',
};

/**
 * What a player's input takes in, read off the wires leaving it: audio for a
 * microphone, text for typing.
 */
function inputKind(name: string, edges: readonly TopologyEdge[]): "audio" | "image" | "text" {
  const kinds = edges.filter((edge) => edge.from === name).map((edge) => edge.kind);
  if (kinds.includes("audio")) return "audio";
  // A picture that is there from the start, such as the game's gallery image.
  if (kinds.includes("image")) return "image";
  return "text";
}

/** A symbol for a kind of wire or input, as an image cytoscape can paint. */
function icon(kind: string, colour: string): string {
  const glyph = GLYPHS[kind === "stream" ? "text" : kind] ?? GLYPHS.text!;
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" ` +
    `stroke="${colour}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${glyph}</svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

/**
 * The wiring without where failures go. Any block can fail, so drawing every
 * failure sink and the wire to it buries the flow a turn takes. A failure
 * still reaches the player; it is only not drawn.
 */
function withoutFailures(topology: Topology): Topology {
  const failures = new Set(
    topology.nodes.filter((node) => node.role === "failure").map((node) => node.name),
  );
  return {
    ...topology,
    nodes: topology.nodes.filter((node) => !failures.has(node.name)),
    edges: topology.edges.filter(
      (edge) => edge.kind !== "failure" && !failures.has(edge.from) && !failures.has(edge.to),
    ),
  };
}

/**
 * How far to move a span [low, high] so it stays within [0, size], keeping
 * `lead` clear before it and `trail` after it.
 */
function shift(low: number, high: number, size: number, lead: number, trail: number): number {
  const fits = high - low <= size - lead - trail;
  // A span that fits must stay inside the margins; one that does not must keep
  // them covered.
  if (fits ? low < lead : low > lead) return lead - low;
  if (fits ? high > size - trail : high < size - trail) return size - trail - high;
  return 0;
}

