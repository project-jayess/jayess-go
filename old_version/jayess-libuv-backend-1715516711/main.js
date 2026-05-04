
	import { createLoop, scheduleStop, scheduleCallback, readFile, watchSignal, closeSignalWatcher, watchPath, closePathWatcher, spawnProcess, closeProcess, createUDP, bindUDP, recvUDP, sendUDP, closeUDP, createTCPServer, listenTCP, acceptTCP, closeTCPServer, createTCPClient, connectTCP, readTCP, writeTCP, closeTCPClient, run, runOnce, stop, closeLoop, now } from "@jayess/libuv";

function main(args) {
  var loop = createLoop();
  var before = now(loop);
  var count = 0;
  var fileText = "";
  var fileError = "";
  var signalText = "";
  var watchType = "";
  var processExit = "";
  var closeProcessResult = "";
  var udpText = "";
  var closeUDPResult = "";
  var tcpServerText = "";
  var tcpClientText = "";
  var closeTCPServerResult = "";
  var closeAcceptedTCPResult = "";
  var closeTCPClientResult = "";
  var acceptedClient = undefined;
  scheduleCallback(loop, 0, () => {
    count = count + 1;
  });
  readFile(loop, "./hello.txt", (result) => {
    if (result.ok) {
      fileText = result.data;
    } else {
      fileError = result.error.name;
    }
  });
  var watcherToClose = watchSignal(loop, "SIGUSR2", (signal) => {});
  var watcher = watchSignal(loop, "SIGUSR1", (signal) => {
    signalText = signal;
    stop(loop);
  });
  var pathWatcherToClose = watchPath(loop, "./hello.txt", (result) => {});
  var pathWatcher = watchPath(loop, "./watched.txt", (result) => {
    if (result.ok) {
      watchType = result.eventType;
    }
  });
  var process = spawnProcess(loop, "/bin/sh", ["-c", "exit 7"], (result, proc) => {
    processExit = result.exitStatus + ":" + (result.signal === undefined);
    closeProcessResult = "" + closeProcess(proc);
  });
  var udp = createUDP(loop);
  bindUDP(udp, "127.0.0.1", 49488);
  recvUDP(udp, (result) => {
    if (result.ok) {
      udpText = result.data;
    }
  });
  sendUDP(udp, "127.0.0.1", 55280, "pong");
  var tcpServer = createTCPServer(loop);
  listenTCP(tcpServer, "127.0.0.1", 43891, (result) => {
    if (result.ok) {
      acceptedClient = acceptTCP(tcpServer);
      if (acceptedClient != undefined) {
        readTCP(acceptedClient, (packet) => {
          if (packet.ok) {
            tcpServerText = packet.data;
            writeTCP(acceptedClient, "world");
          }
        });
      }
    }
  });
  var tcpClient = createTCPClient(loop);
  connectTCP(tcpClient, "127.0.0.1", 43891, (result) => {
    if (result.ok) {
      readTCP(tcpClient, (packet) => {
        if (packet.ok) {
          tcpClientText = packet.data;
        }
      });
      writeTCP(tcpClient, "hello");
    }
  });
  scheduleStop(loop, 200);
  var ran = run(loop);
  var after = now(loop);
  console.log("libuv-run:" + ran + ":" + (after >= before));
  console.log("libuv-callback:" + count);
  console.log("libuv-read-file:" + fileText + ":" + fileError);
  console.log("libuv-signal:" + signalText);
  console.log("libuv-close-watcher:" + closeSignalWatcher(watcherToClose));
  console.log("libuv-close-active-watcher:" + closeSignalWatcher(watcher));
  console.log("libuv-watch-type:" + watchType);
  console.log("libuv-close-path-watcher:" + closePathWatcher(pathWatcherToClose));
  console.log("libuv-close-active-path-watcher:" + closePathWatcher(pathWatcher));
  console.log("libuv-process-exit:" + processExit);
  console.log("libuv-close-process:" + closeProcessResult);
  console.log("libuv-udp:" + udpText);
  console.log("libuv-close-udp:" + closeUDP(udp));
  if (acceptedClient != undefined) {
    closeAcceptedTCPResult = "" + closeTCPClient(acceptedClient);
  }
  closeTCPClientResult = "" + closeTCPClient(tcpClient);
  closeTCPServerResult = "" + closeTCPServer(tcpServer);
  console.log("libuv-tcp-server:" + tcpServerText);
  console.log("libuv-tcp-client:" + tcpClientText);
  console.log("libuv-close-accepted-tcp:" + closeAcceptedTCPResult);
  console.log("libuv-close-tcp-client:" + closeTCPClientResult);
  console.log("libuv-close-tcp-server:" + closeTCPServerResult);

  scheduleStop(loop, 0);
  var once = runOnce(loop);
  console.log("libuv-once:" + once);
  stop(loop);
  console.log("libuv-close:" + closeLoop(loop));
  try {
    now(loop);
    console.log("libuv-after-close:false");
  } catch (err) {
    console.log("libuv-after-close:" + err.name);
  }
  return 0;
}

