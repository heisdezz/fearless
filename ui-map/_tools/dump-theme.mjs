// Dumps Bruno's light/dark theme tokens with every hsl()/rgba() resolved to
// concrete values. Run from ./bruno with: node ../ui-map/_tools/dump-theme.mjs
import { readFileSync, writeFileSync } from "node:fs";

// --- minimal polished-compatible helpers --------------------------------
function parseColor(c) {
  c = String(c).trim();
  let m = c.match(
    /^hsla?\(\s*([\d.]+)(?:deg)?[\s,]+([\d.]+)%[\s,]+([\d.]+)%\s*(?:[,/]+\s*([\d.]+%?)\s*)?\)$/,
  );
  if (m) {
    const h = parseFloat(m[1]) / 360,
      s = parseFloat(m[2]) / 100,
      l = parseFloat(m[3]) / 100;
    const a = m[4] !== undefined ? (m[4].endsWith('%') ? parseFloat(m[4]) / 100 : parseFloat(m[4])) : 1;
    const hue2rgb = (p, q, t) => {
      if (t < 0) t += 1;
      if (t > 1) t -= 1;
      if (t < 1 / 6) return p + (q - p) * 6 * t;
      if (t < 1 / 2) return q;
      if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
      return p;
    };
    let r, g, b;
    if (s === 0) {
      r = g = b = l;
    } else {
      const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
      const p = 2 * l - q;
      r = hue2rgb(p, q, h + 1 / 3);
      g = hue2rgb(p, q, h);
      b = hue2rgb(p, q, h - 1 / 3);
    }
    return {
      r: Math.round(r * 255),
      g: Math.round(g * 255),
      b: Math.round(b * 255),
      a,
    };
  }
  m = c.match(/^#([0-9a-f]{3,8})$/i);
  if (m) {
    let h = m[1];
    if (h.length === 3)
      h = h
        .split("")
        .map((x) => x + x)
        .join("");
    const n = parseInt(h.slice(0, 6), 16);
    const a = h.length === 8 ? parseInt(h.slice(6, 8), 16) / 255 : 1;
    return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255, a };
  }
  m = c.match(
    /^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)\s*(?:[,\s/]+\s*([\d.]+%?)\s*)?\)$/,
  );
  if (m) {
    const a =
      m[4] !== undefined
        ? m[4].endsWith("%")
          ? parseFloat(m[4]) / 100
          : parseFloat(m[4])
        : 1;
    return { r: +m[1], g: +m[2], b: +m[3], a };
  }
  throw new Error("cannot parse colour: " + c);
}
const hex = (r, g, b) =>
  "#" + [r, g, b].map((v) => v.toString(16).padStart(2, "0")).join("");
function normalize(c) {
  if (typeof c !== "string") return c;
  c = c.trim();
  if (/^#/.test(c) && (c.length === 7 || c.length === 4))
    return c.toLowerCase();
  if (!/^(hsl|rgb|#)/i.test(c)) return c; // plain strings like 'solid 1px ...'
  const { r, g, b, a } = parseColor(c);
  return a >= 0.999 ? hex(r, g, b) : `rgba(${r}, ${g}, ${b}, ${+a.toFixed(3)})`;
}
const rgba = (c, a) => {
  const { r, g, b } = parseColor(c);
  return `rgba(${r}, ${g}, ${b}, ${a})`;
};
const lighten = (amount, c) => {
  // polished lighten adds `amount` (0..1) to the colour's HSL lightness.
  const m = String(c).trim().match(/^hsla?\(\s*([\d.]+)(?:deg)?[\s,]+([\d.]+)%[\s,]+([\d.]+)%/);
  if (!m) throw new Error('lighten needs hsl input, got ' + c);
  const h = parseFloat(m[1]);
  const s = parseFloat(m[2]);
  const l = Math.min(100, parseFloat(m[3]) + amount * 100);
  return `hsl(${h}, ${s}%, ${l}%)`;
};

function loadTheme(file) {
  let src = readFileSync(file, "utf8");
  src = src.replace(/import\s*\{[^}]*\}\s*from\s*['"]polished['"];?/g, "");
  const def = src.match(/export\s+default\s+(\w+)/);
  if (!def) throw new Error("no default export in " + file);
  const body = src
    .replace(/export\s+default\s+\w+;?/, "")
    .replace(/\bexport\s+/g, "");
  const factory = new Function("rgba", "lighten", body + `\nreturn ${def[1]};`);
  return factory(rgba, lighten);
}

function resolveDeep(node) {
  if (typeof node === "string") return normalize(node);
  if (Array.isArray(node)) return node.map(resolveDeep);
  if (node && typeof node === "object") {
    const o = {};
    for (const [k, v] of Object.entries(node)) o[k] = resolveDeep(v);
    return o;
  }
  return node;
}

const root = new URL(
  "../../bruno/packages/bruno-app/src/themes/",
  import.meta.url,
).pathname;
const out = {};
for (const [name, rel] of Object.entries({
  light: "light/light.js",
  dark: "dark/dark.js",
})) {
  out[name] = resolveDeep(loadTheme(root + rel));
}

writeFileSync(
  new URL("./tokens.json", import.meta.url),
  JSON.stringify(out, null, 2),
);
console.log("wrote tokens.json");
console.log(
  "light.primary.solid =",
  out.light.primary.solid,
  " dark =",
  out.dark.primary.solid,
);
console.log(
  "light.get =",
  out.light.request.methods.get,
  " dark.get =",
  out.dark.request.methods.get,
);
console.log(
  "light.bg.base =",
  out.light.background.base,
  " dark.bg.base =",
  out.dark.background.base,
);
