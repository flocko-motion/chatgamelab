/**
 * The launch panel, shown in the game's place before a session is had: it
 * offers to continue a remembered game or start anew, and says why a game
 * cannot start. It asks; main decides what follows.
 */
export interface LaunchElements {
  panel: HTMLElement;
  text: HTMLElement;
  actions: HTMLElement;
  /** What the panel stands in for while it shows. */
  scene: HTMLElement;
}

export class LaunchPanel {
  constructor(private readonly elements: LaunchElements) {}

  /** Shows a message with buttons, and resolves with the label pressed; without buttons, never. */
  ask(message: string, actions: readonly string[]): Promise<string> {
    const { panel, text, scene } = this.elements;
    text.textContent = message;
    panel.hidden = false;
    scene.style.display = "none";
    return new Promise((resolve) => {
      this.elements.actions.replaceChildren(
        ...actions.map((label, i) => {
          const button = document.createElement("button");
          button.type = "button";
          button.className = i === 0 ? "launch-action primary" : "launch-action";
          button.textContent = label;
          button.addEventListener("click", () => resolve(label));
          return button;
        }),
      );
    });
  }

  /** Shows that something is under way, such as a game being created. */
  wait(message: string): void {
    void this.ask(message, []);
  }

  hide(): void {
    this.elements.panel.hidden = true;
    this.elements.actions.replaceChildren();
    this.elements.scene.style.display = "";
  }
}
