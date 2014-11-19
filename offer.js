var peerConnection = new RTCPeerConnection({});
navigator.getUserMedia({ "audio": true }, gotStream, logError);

function gotStream(stream) {
    peerConn.addStream
}

function logError(error) {
    window.console.log("Errors happen: " + error);
}