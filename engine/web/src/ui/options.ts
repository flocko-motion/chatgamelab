/**
 * The options dialog, shown in the game's place while the session waits for
 * the player's choice, and the pictures switch in the header during play. Both
 * paint the state and pass the player's decision on; the engine decides.
 */
import type { Choice, OfferOption, Options, Toggles } from "../protocol.js";
import { TOGGLE_LABELS, allows, makesPictures, type PlayerState, type ToggleName } from "../state.js";

export interface OptionsElements {
  form: HTMLFormElement;
  offers: HTMLElement;
  problem: HTMLElement;
  start: HTMLButtonElement;
  /** What the dialog stands in for while it shows. */
  scene: HTMLElement;
  pictures: HTMLButtonElement;
}

export interface OptionsHandlers {
  onChoose(offer: string, choice?: Choice): void;
  onToggle(toggle: ToggleName, on: boolean): void;
}

const TOGGLES = Object.keys(TOGGLE_LABELS) as ToggleName[];

export class OptionsDialog {
  #shown: Options | null = null;
  #offer = "";
  #tier = "";
  #toggles: Toggles = { images: false, "audio-out": false, "audio-in": false, "audio-live": false };

  constructor(
    private readonly elements: OptionsElements,
    private readonly handlers: OptionsHandlers,
  ) {
    elements.form.addEventListener("submit", (event) => {
      event.preventDefault();
      const offer = this.#selected();
      if (!offer) return;
      this.handlers.onChoose(
        offer.offer,
        offer.customizable ? { tier: this.#tier, toggles: { ...this.#toggles } } : undefined,
      );
    });
    elements.pictures.addEventListener("click", () => {
      const on = elements.pictures.getAttribute("aria-pressed") === "true";
      this.handlers.onToggle("images", !on);
    });
  }

  render(state: PlayerState): void {
    const open = state.choosing && state.options !== null;
    this.elements.form.hidden = !open;
    this.elements.scene.style.display = open ? "none" : "";
    if (open && state.options !== this.#shown) {
      this.#shown = state.options;
      this.#preselect(state.options?.offers[0]);
      this.#draw();
    }
    this.elements.problem.hidden = !(open && state.problem);
    this.elements.problem.textContent = open ? (state.problem ?? "") : "";

    this.#renderPictures(state);
  }

  /** On while the session has pictures on; it may go back on only where the offer allows. */
  #renderPictures(state: PlayerState): void {
    const button = this.elements.pictures;
    const settled = state.settled;
    button.hidden = !settled || !makesPictures(state);
    if (!settled) return;
    const on = settled.choice.toggles.images;
    button.setAttribute("aria-pressed", String(on));
    button.textContent = on ? "Pictures on" : "Pictures off";
    button.disabled = !on && !allows(state, "images");
    button.title = button.disabled
      ? "Pictures are not available with this key"
      : on
        ? "Stop making new pictures"
        : "Make pictures again";
  }

  #selected(): OfferOption | undefined {
    return this.#shown?.offers.find((offer) => offer.offer === this.#offer);
  }

  /** The offer's default, preselected. */
  #preselect(offer: OfferOption | undefined): void {
    if (!offer) return;
    this.#offer = offer.offer;
    this.#tier = offer.default.tier;
    this.#toggles = { ...offer.default.toggles };
  }

  #draw(): void {
    const offers = this.#shown?.offers ?? [];
    this.elements.offers.replaceChildren(...offers.map((offer) => this.#card(offer, offers.length > 1)));
    this.elements.start.disabled = !this.#selected();
  }

  #card(offer: OfferOption, pickable: boolean): HTMLElement {
    const selected = offer.offer === this.#offer;
    const card = element("fieldset", selected ? "offer-card selected" : "offer-card");

    const head = element("label", "offer-head");
    if (pickable) {
      const radio = input("radio", "offer", selected);
      radio.addEventListener("change", () => {
        this.#preselect(offer);
        this.#draw();
      });
      head.append(radio);
    }
    head.append(element("strong", "offer-name", offer.name), element("span", "offer-summary", summary(offer)));
    card.append(head);

    if (selected && offer.customizable) card.append(this.#tune(offer));
    return card;
  }

  /** The tier and the toggles allowed at it, for a customizable offer. */
  #tune(offer: OfferOption): HTMLElement {
    const tune = element("div", "offer-tune");

    const tiers = element("div", "offer-tiers");
    for (const tier of offer.tiers) {
      const label = element("label", "offer-tier");
      const radio = input("radio", `tier-${offer.offer}`, tier.tier === this.#tier);
      radio.addEventListener("change", () => {
        this.#tier = tier.tier;
        // A toggle the new tier does not allow goes off.
        for (const toggle of TOGGLES) if (!tier.toggles[toggle]) this.#toggles = { ...this.#toggles, [toggle]: false };
        this.#draw();
      });
      label.append(radio, element("span", "", tier.label), element("small", "offer-models", models(tier.models)));
      tiers.append(label);
    }
    tune.append(tiers);

    const allowed = offer.tiers.find((tier) => tier.tier === this.#tier)?.toggles;
    const toggles = element("div", "offer-toggles");
    for (const toggle of TOGGLES) {
      if (!allowed?.[toggle]) continue;
      const label = element("label", "offer-toggle");
      const box = input("checkbox", toggle, this.#toggles[toggle]);
      box.addEventListener("change", () => {
        this.#toggles = { ...this.#toggles, [toggle]: box.checked };
      });
      label.append(box, element("span", "", TOGGLE_LABELS[toggle]));
      toggles.append(label);
    }
    tune.append(toggles);
    return tune;
  }
}

/** What an offer allows, in a line: its tiers, and what a fixed one runs with. */
function summary(offer: OfferOption): string {
  if (!offer.customizable) {
    const tier = offer.tiers.find((t) => t.tier === offer.default.tier)?.label ?? offer.default.tier;
    return [tier, ...on(offer.default.toggles)].join(" · ");
  }
  const tiers = offer.tiers.map((t) => t.label).join(", ");
  const features = new Set(offer.tiers.flatMap((t) => on(t.toggles)));
  return [tiers, ...features].join(" · ");
}

function on(toggles: Toggles): string[] {
  return TOGGLES.filter((toggle) => toggles[toggle]).map((toggle) => TOGGLE_LABELS[toggle]);
}

/** The models behind a tier, as a detail: the story's and the pictures'. */
function models(models: Record<string, string>): string {
  return [models["story"], models["image"]].filter(Boolean).join(" · ");
}

function input(type: string, name: string, checked: boolean): HTMLInputElement {
  const box = document.createElement("input");
  box.type = type;
  box.name = name;
  box.checked = checked;
  return box;
}

function element(tag: string, className: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}
