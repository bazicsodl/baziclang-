# Bazic UI Interop Adapter Spec (v1)

This spec describes how external UI frameworks can render Bazic UI trees.

## 1) Core Contract
- Bazic UI renders a JSON tree by calling `ui_render(tree_json)`.
- JS runtime exposes `globalThis.BAZIC_UI_RENDER(jsonText)` to consume the tree.
- The tree is a single root node (object) or a text node.

### Node shape
```json
{
  "t": "elem",
  "tag": "div",
  "props": { "id": "root", "class": "card" },
  "children": [
    { "t": "text", "v": "Hello" }
  ]
}
```

Text node:
```json
{ "t": "text", "v": "Hello" }
```

## 2) Props Rules
- `props` is a JSON object (string keys).
- Common fields: `id`, `class`, `key`, `aria-*`, `role`, `placeholder`, `value`, `checked`, `colspan`, `type`.
- Adapters should apply all props as attributes except:
  - `id`, `class`, `key` (handled separately).
  - `value`/`checked` (applied to input/select).
- `key` is used for keyed child reconciliation.

## 3) Children
- `children` is always an array (possibly empty).
- For `input` elements, children are ignored.
- For `select`, children are `option` elements.

## 4) Events (Bridge)
The default `app.js` uses these event keys:
- `event_type` (e.g., `click`, `input`, `route`, `action`)
- `event_target` (the element id)
- `event_value` (string value)
- `event_action` (optional action value for `[data-action]`)

Any adapter must:
- listen for DOM `click`, `input`, `change`
- set `event_*` on `globalThis.BAZIC_WEB`
- call `BAZIC_WEB.set("event", type)` and include `event_type`

## 5) Minimal Adapter API
Adapters must provide:
- `globalThis.BAZIC_UI_RENDER(jsonText)` to accept the tree
- a container element (e.g., `#ui`) to host rendering

## 6) Example: React Adapter (concept)
```js
function toReact(node) {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  const props = node.props || {};
  const children = (node.children || []).map(toReact);
  return React.createElement(node.tag || "div", props, ...children);
}

globalThis.BAZIC_UI_RENDER = (jsonText) => {
  const tree = JSON.parse(jsonText);
  ReactDOM.render(toReact(tree), document.getElementById("ui"));
};
```

## 7) Example: Vue Adapter (concept)
```js
function toVNode(node, h) {
  if (!node) return null;
  if (node.t === "text") return node.v || "";
  const props = node.props || {};
  const children = (node.children || []).map((child) => toVNode(child, h));
  return h(node.tag || "div", props, children);
}
```

## 8) Example: Svelte Adapter (concept)
- Parse the JSON tree and render with a minimal custom component.
- Use `{@html}` only for text nodes if sanitized.

## 9) Adapter Checklist
- Apply props and attributes correctly.
- Preserve `input`/`select` values without clobbering user typing.
- Use `key` for stable child reconciliation.
- Forward `click`, `input`, `change` into Bazic events.
