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

export class Flowchart {
  #cy: cytoscape.Core | null = null;
  #drawn = false;

  constructor(private readonly container: HTMLElement) {}

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
            shape: "round-rectangle",
            width: "label",
            height: 22,
            padding: "6px",
            "background-color": bg,
            "border-width": 1,
            "border-color": line,
            "transition-property": "border-color, background-color",
            "transition-duration": 120,
          },
        },
        // Sources and sinks are shaped differently so the stem and the sinks
        // are findable without reading every label.
        { selector: 'node[role = "source"]', style: { shape: "round-tag" } },
        { selector: 'node[role = "sink"]', style: { shape: "round-diamond", height: 28 } },
        {
          selector: "node.working",
          style: { "border-color": live, "border-width": 2, "background-color": bg },
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
        rankDir: "LR",
        nodeSep: 14,
        rankSep: 40,
        // The observer's correction makes the graph cyclic on purpose, which
        // the layout has to tolerate rather than reject.
        acyclicer: "greedy",
      } as cytoscape.LayoutOptions,
    });

    this.#cy.fit(undefined, 12);
  }

  /** Lights the blocks that are working, and labels each with what it spent. */
  update(phases: Record<string, Phase>, spent: UsageRecord[]): void {
    const cy = this.#cy;
    if (!cy) return;

    cy.batch(() => {
      for (const node of cy.nodes()) {
        const phase = phases[node.id()] ?? "ready";
        node.toggleClass("working", phase === "working");

        const record = spent.find((entry) => entry.node === node.id());
        node.data("label", record ? `${node.id()}  ${cost(record)}` : node.id());
      }
    });
  }
}

/** Cost is an estimate from a hand-maintained table, so it is shown coarsely. */
function cost(record: UsageRecord): string {
  if (!record.cost) return "";
  if (record.cost < 0.01) return "<1¢";
  return `$${record.cost.toFixed(2)}`;
}
