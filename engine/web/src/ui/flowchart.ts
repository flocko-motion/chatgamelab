/**
 * Draws the session's wiring and lights each block as it works.
 *
 * Purely presentational: it is handed topology, phases and costs from the
 * headless core's state and owns none of them. Nothing here decides anything,
 * and the core neither imports this nor knows it exists.
 */
import cytoscape from "cytoscape";
import dagre from "cytoscape-dagre";
import type { Phase, UsageRecord } from "../protocol.js";
import type { Topology } from "../state.js";

cytoscape.use(dagre);

/** Reads a CSS custom property, so the diagram follows the page's theme. */
function token(name: string, fallback: string): string {
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

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

    this.#cy = cytoscape({
      container: this.container,
      // A teaching diagram, not a toy to drag apart.
      userPanningEnabled: true,
      userZoomingEnabled: true,
      boxSelectionEnabled: false,
      autoungrabify: true,
      elements: [
        ...topology.nodes.map((node) => ({
          data: { id: node.name, label: node.name, role: node.role },
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
            "font-family": "ui-sans-serif, system-ui, sans-serif",
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
          // A selected block and the edges touching it, so a reader can follow
          // what a stage is connected to while reading its detail.
          selector: "node.selected",
          style: { "border-color": fg, "border-width": 3 },
        },
        {
          selector: "edge.incident",
          style: { "line-color": fg, "target-arrow-color": fg, width: 2, color: fg },
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
      ],
      layout: {
        name: "dagre",
        // Top down, so the P reads the way the document draws it: the init
        // stem above, the turn loop below.
        rankDir: "TB",
        nodeSep: 18,
        rankSep: 34,
        // The observer's correction makes the graph cyclic on purpose, which
        // the layout has to tolerate rather than reject.
        acyclicer: "greedy",
      } as cytoscape.LayoutOptions,
    });

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
    this.#cy.fit(undefined, 12);
  }

  /** Marks one block and the edges touching it, or clears the marking. */
  select(name: string | null): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().removeClass("selected incident");
      if (!name) return;

      const node = cy.getElementById(name);
      if (node.empty()) return;
      node.addClass("selected");
      node.connectedEdges().addClass("incident");
    });
  }

  /** Marks one wire and the blocks it joins. */
  selectEdge(id: string): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().removeClass("selected incident");
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

/** Cost is an estimate from a hand-maintained table, so it is shown coarsely. */
function cost(record: UsageRecord): string {
  if (!record.cost) return "";
  if (record.cost < 0.01) return "<1¢";
  return `$${record.cost.toFixed(2)}`;
}
