package hx

// replayScript records the events of the user until the WASM has hydrated the
// page. It listens on the document, in the capture phase, so it sees an event
// before anything else does, and keeps the event object itself: hydrating
// reads from it the type, the target, the key... and fires the handlers with it.
//
// Rules, kept in step with wasm/replay.go:
//   - only click, input, change and keydown, the events that carry an intent;
//   - an input or change that follows one of the same kind on the same element
//     replaces it: only the last value matters, and the element has it;
//   - at most 200 events are kept, the oldest are dropped;
//   - an event inside an element that Hydrate has taken (it has __hxLive) is
//     not kept: its listeners are already working.
//
// __hxqStop removes the listeners (wasm.StopReplay calls it).
const replayScript = `(function(){var q=window.__hxq=[],t=["click","input","change","keydown"];` +
	`function h(e){var n=e.target,l=q[q.length-1];if(!n||n.nodeType!==1)return;` +
	`for(;n;n=n.parentNode)if(n.__hxLive)return;` +
	`if(l&&l.target===e.target&&l.type===e.type&&(e.type==="input"||e.type==="change"))q[q.length-1]=e;` +
	`else{if(q.length>=200)q.shift();q.push(e)}}` +
	`t.forEach(function(n){document.addEventListener(n,h,true)});` +
	`window.__hxqStop=function(){t.forEach(function(n){document.removeEventListener(n,h,true)});window.__hxq=undefined}})()`

// ReplayScript returns a <script> element that remembers what the user does
// while the WASM is still loading, so wasm.Hydrate can do it once the page is
// alive: a click on a button, text typed in a field, a change of a select, a
// key pressed. Without it, those events are lost.
//
// Put it as early as you can, in the head, before the page content and the
// script that loads the WASM. It needs no state and does nothing on its own:
// if hydration never happens, it only keeps a short list in memory. It is an
// inline script, so a Content-Security-Policy has to allow it (a nonce or a
// hash of its content).
//
// What it replays is the intent, not the browser event: the handlers run with
// the event that happened, but what they do to it (PreventDefault,
// StopPropagation) is too late. A form that was sent before hydration was sent
// by the browser, as a form without WASM is.
func ReplayScript() string {
	return "<script>" + replayScript + "</script>"
}
