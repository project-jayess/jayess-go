import {
  bindNative as webviewBindNative,
  createWindowNative as webviewCreateWindowNative,
  destroyWindowNative as webviewDestroyWindowNative,
  evalJsNative as webviewEvalJsNative,
  hideNative as webviewHideNative,
  initJsNative as webviewInitJsNative,
  navigateNative as webviewNavigateNative,
  nextBindingEventNative as webviewNextBindingEventNative,
  returnBindingNative as webviewReturnBindingNative,
  runNative as webviewRunNative,
  setHtmlNative as webviewSetHtmlNative,
  setSizeNative as webviewSetSizeNative,
  setTitleNative as webviewSetTitleNative,
  showNative as webviewShowNative,
  terminateNative as webviewTerminateNative,
  unbindNative as webviewUnbindNative
} from "./native/webview.bind.js";

export function createWindow(debug) {
  return webviewCreateWindowNative(debug);
}

export function destroyWindow(view) {
  return webviewDestroyWindowNative(view);
}

export function setTitle(view, title) {
  return webviewSetTitleNative(view, title);
}

export function setSize(view, width, height) {
  return webviewSetSizeNative(view, width, height);
}

export function show(view) {
  return webviewShowNative(view);
}

export function hide(view) {
  return webviewHideNative(view);
}

export function setHtml(view, html) {
  return webviewSetHtmlNative(view, html);
}

export function navigate(view, url) {
  return webviewNavigateNative(view, url);
}

export function loadFile(view, path) {
  return webviewNavigateNative(view, "file://" + path);
}

export function initJs(view, source) {
  return webviewInitJsNative(view, source);
}

export function evalJs(view, source) {
  return webviewEvalJsNative(view, source);
}

export function bind(view, name) {
  return webviewBindNative(view, name);
}

export function unbind(view, name) {
  return webviewUnbindNative(view, name);
}

export function nextBindingEvent(view) {
  return webviewNextBindingEventNative(view);
}

export function returnBinding(view, id, status, result) {
  return webviewReturnBindingNative(view, id, status, result);
}

export function run(view) {
  return webviewRunNative(view);
}

export function terminate(view) {
  return webviewTerminateNative(view);
}

