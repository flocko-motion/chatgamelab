// Shared look for the engine's spec documents.

#let spec(title: none, body) = {
  set document(title: title)
  set page(paper: "a4", margin: (x: 2.2cm, y: 2.4cm), numbering: "1")
  set text(size: 10.5pt, lang: "en", region: "gb")
  set par(justify: true, leading: 0.65em)
  set heading(numbering: "1.1")
  show heading.where(level: 1): set block(above: 1.6em, below: 0.9em)
  show raw.where(block: false): box.with(
    fill: luma(240), inset: (x: 3pt, y: 0pt), outset: (y: 3pt), radius: 2pt,
  )
  show raw.where(block: true): block.with(
    fill: luma(245), inset: 8pt, radius: 3pt, width: 100%,
  )
  set table(stroke: 0.5pt + luma(180), inset: 6pt)
  show table.cell.where(y: 0): strong

  align(center, text(size: 20pt, weight: "bold", title))
  v(1em)
  body
}

// A gap the spec leaves open on purpose: shown inline, in a colour that stands out.
#let todo(body) = box(
  fill: rgb("#fff3cd"), stroke: 0.5pt + rgb("#d9a400"), inset: (x: 4pt, y: 2pt),
  outset: (y: 2pt), radius: 2pt,
)[*TODO:* #body]
