const svgNamespace = "http://www.w3.org/2000/svg";

export function div(className: string): HTMLDivElement {
  const node = document.createElement("div");
  node.className = className;
  return node;
}

/** The root an animation draws into, hidden from assistive technology. */
export function backdrop(className = "bg-fill"): HTMLDivElement {
  const node = div(className);
  node.setAttribute("aria-hidden", "true");
  return node;
}

export function svg(tag: string, attributes: Record<string, string | number>): SVGElement {
  const node = document.createElementNS(svgNamespace, tag);
  for (const [name, value] of Object.entries(attributes)) node.setAttribute(name, String(value));
  return node;
}
