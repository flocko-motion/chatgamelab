/**
 * Draws the session's wiring and lights each block as it works.
 *
 * Purely presentational: it is handed topology, phases and costs from the
 * headless core's state and owns none of them. Nothing here decides anything,
 * and the core neither imports this nor knows it exists.
 */
import cytoscape from "cytoscape";
import type { Cell, UsageRecord } from "../client.js";
import type { Phase } from "../protocol.js";
import type { Topology } from "../client.js";
import { layoutCode, parked } from "../layout.js";
import { text } from "../texts.js";

/** Reads a CSS custom property, so the diagram follows the page's theme. */
function token(name: string, fallback: string): string {
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

/** Mixes two #rrggbb colours, weight of the first; anything else gives the first. */
function mix(first: string, second: string, weight: number): string {
  const rgb = (hex: string): number[] | null => {
    const m = /^#([0-9a-f]{6})$/i.exec(hex);
    if (!m) return null;
    const n = parseInt(m[1]!, 16);
    return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
  };
  const a = rgb(first);
  const b = rgb(second);
  if (!a || !b) return first;
  const mixed = a.map((v, i) => Math.round(v * weight + b[i]! * (1 - weight)));
  return `#${mixed.map((v) => v.toString(16).padStart(2, "0")).join("")}`;
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
  /** Where each block is drawn; on a dev server, where it has been dragged to. */
  #cells = new Map<string, Cell>();
  /** The cell a dragged block left, so a block it lands on can take it. */
  #grabbed: Cell | null = null;

  /** One grid step, in model units: a column fits the longest block name. */
  static readonly column = 150;
  static readonly row = 64;

  /**
   * How long a finished block keeps its afterglow: long enough to be seen,
   * short enough that a busy graph is not permanently lit.
   */
  static readonly afterglowMs = 700;

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
  draw(topology: Topology): void {
    if (this.#drawn) return;
    this.#drawn = true;

    const fg = token("--fg", "#1b1a19");
    const dim = token("--dim", "#6b6a68");
    const line = token("--line", "#e2e0dc");
    const bg = token("--bg", "#faf9f7");
    const live = token("--live", "#2f7d5a");
    const liveSoft = token("--live-soft", "#d9ece2");
    const recent = token("--recent", "#9ec9b4");
    const into = token("--wire-in", "#16a34a");
    const outOf = token("--wire-out", "#dc2626");
    // A failure edge is the outgoing wire's red faded into the background,
    // pale on a light pane and dark on a dark one.
    const failure = mix(outOf, bg, 0.35);

    // The genre places every block by hand. One the layout misses is parked
    // below the drawing and marked, until a developer gives it a place.
    for (const node of topology.nodes) if (node.cell) this.#cells.set(node.name, node.cell);
    const unplaced = topology.nodes.filter((node) => !node.cell).map((node) => node.name);
    for (const [name, cell] of parked(this.#cells, unplaced)) this.#cells.set(name, cell);

    this.#cy = cytoscape({
      container: this.container,
      // A teaching diagram, not a toy to drag apart; only a developer
      // rearranges it, to copy the layout back into the genre.
      userPanningEnabled: true,
      userZoomingEnabled: true,
      boxSelectionEnabled: false,
      autoungrabify: !topology.dev,
      elements: [
        ...topology.nodes.map((node) => ({
          data: { id: node.name, label: node.name, role: node.role },
          position: Flowchart.#at(this.#cells.get(node.name)!),
          classes: node.cell ? "" : "unplaced",
        })),
        ...topology.edges.map((edge, index) => ({
          data: {
            id: `e${index}`,
            source: edge.from,
            target: edge.to,
            label: edge.kind,
          },
        })),
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
            // The cost goes on its own line under the name rather than beside
            // it, so a long model name never pushes a node wide.
            "text-wrap": "wrap",
            shape: "round-rectangle",
            width: "label",
            height: 22,
            padding: "6px",
            "background-color": bg,
            "border-width": 1,
            "border-color": line,
            // Every style that changes is animated, so a turn reads as motion
            // through the graph rather than as a sequence of stills.
            "transition-property":
              "border-color, border-width, background-color, overlay-opacity",
            "transition-duration": 180,
            "transition-timing-function": "ease-out",
            "overlay-color": live,
            "overlay-opacity": 0,
            "overlay-padding": 6,
          },
        },
        // Sources and sinks are shaped differently so the stem and the sinks
        // are findable without reading every label.
        { selector: 'node[role = "source"]', style: { shape: "round-tag" } },
        { selector: 'node[role = "sink"]', style: { shape: "round-diamond", height: 28 } },
        // The gate is where the stem joins the loop, so it is shaped like a
        // join rather than like the blocks on either side of it.
        { selector: 'node[role = "gate"]', style: { shape: "round-hexagon", height: 26 } },
        {
          // A block the genre's layout misses: findable, so it gets placed.
          selector: "node.unplaced",
          style: { "border-style": "dashed", "border-color": outOf },
        },
        {
          selector: "node.working",
          style: {
            "border-color": live,
            "border-width": 2,
            "background-color": liveSoft,
            // A halo, so a working block is findable at a glance on a graph
            // with a dozen nodes.
            "overlay-opacity": 0.12,
          },
        },
        {
          // The afterglow. Without it a block that finishes in 40 ms never
          // renders at all, and the observer is exactly that fast.
          selector: "node.recent",
          style: { "border-color": recent, "border-width": 2 },
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
          // Where a block's failed call goes: pale and dashed, apart from the
          // flow a turn takes when all goes well.
          selector: 'edge[label = "failure"]',
          style: { "line-color": failure, "target-arrow-color": failure, "line-style": "dashed" },
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

    if (topology.dev) this.#rearrangeable(topology.name);

    this.#cy.on("tap", "node", (event) => {
      this.select(event.target.id());
      this.onNodeClick(event.target.id());
    });
    this.#cy.on("tap", "edge", (event) => {
      const edge = event.target;
      this.selectEdge(edge.id());
      this.onEdgeClick(edge.data("source"), edge.data("target"), edge.data("label"));
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

  /** A cell's place in model units. */
  static #at(cell: Cell): cytoscape.Position {
    return { x: cell.col * Flowchart.column, y: cell.row * Flowchart.row };
  }

  /**
   * Lets a developer drag blocks into place and copy the layout as Go source.
   * A dropped block snaps to the nearest cell; a block already there swaps
   * into the cell the dropped one left, so two never share a place.
   */
  #rearrangeable(genre: string): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.on("grab", "node", (event) => {
      this.#grabbed = this.#cells.get(event.target.id()) ?? null;
    });
    cy.on("free", "node", (event) => {
      const node: cytoscape.NodeSingular = event.target;
      const { x, y } = node.position();
      const cell = { row: Math.round(y / Flowchart.row), col: Math.round(x / Flowchart.column) };
      const left = this.#grabbed;
      this.#grabbed = null;

      cy.batch(() => {
        const other = [...this.#cells].find(
          ([name, c]) => name !== node.id() && c.row === cell.row && c.col === cell.col,
        );
        if (other && left) {
          this.#cells.set(other[0], left);
          cy.getElementById(other[0]).position(Flowchart.#at(left));
        }
        this.#cells.set(node.id(), cell);
        node.position(Flowchart.#at(cell));
        node.removeClass("unplaced");
      });
    });

    const button = document.createElement("button");
    button.type = "button";
    button.className = "copy-layout";
    button.textContent = text("pipeline.copyLayout");
    button.addEventListener("click", async () => {
      const code = layoutCode(genre, this.#cells);
      try {
        await navigator.clipboard.writeText(code);
        button.textContent = text("pipeline.copied");
      } catch {
        // A page without clipboard access still hands the layout over.
        console.log(code);
        button.textContent = text("pipeline.copyFailed");
      }
      setTimeout(() => (button.textContent = text("pipeline.copyLayout")), 1500);
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

      const node = cy.getElementById(name);
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

  /** Lights the blocks that are working, and labels each with what it spent. */
  update(phases: Record<string, Phase>, spent: UsageRecord[]): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      for (const node of cy.nodes()) {
        const id = node.id();
        const phase = phases[id] ?? "ready";
        const previous = this.#phases.get(id);

        if (phase !== previous) {
          this.#phases.set(id, phase);
          node.toggleClass("working", phase === "working");
          if (previous === "working" && phase === "ready") this.#afterglow(node);
        }

        const record = spent.find((entry) => entry.node === id);
        const price = record ? cost(record) : "";
        node.data("label", price ? `${id}\n${price}` : id);
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

/** Cost is an estimate from a hand-maintained table, so it is shown coarsely. */
function cost(record: UsageRecord): string {
  if (!record.cost) return "";
  if (record.cost < 0.01) return "<1¢";
  return `$${record.cost.toFixed(2)}`;
}
