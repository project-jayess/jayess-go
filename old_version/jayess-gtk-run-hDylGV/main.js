import { init, createWindow, setTitle, show, destroyWindow } from "@jayess/gtk";
function main(args) {
  if (!init()) { console.log("gtk-init:false"); return 0; }
  var window = createWindow();
  if (window == undefined) { console.log("gtk-window:undefined"); return 0; }
  setTitle(window, "Jayess GTK");
  show(window);
  console.log("gtk-window-closed:" + window.closed);
  console.log("gtk-destroy:" + destroyWindow(window));
  return 0;
}
