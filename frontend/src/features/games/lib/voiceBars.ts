/**
 * The bar heights of a voice's waveform tile, in percent, drawn from its
 * voice style, so a voice keeps its look.
 */
export function voiceBars(style: string): number[] {
  let h = 7;
  for (const c of style) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return Array.from({ length: 21 }, (_, i) => {
    h = (h * 1103515245 + 12345) >>> 0;
    const envelope = Math.sin((Math.PI * (i + 1)) / 22);
    return 18 + Math.round(envelope * (30 + (h % 50)));
  });
}
