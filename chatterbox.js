var MINI = require('minified');
var _=MINI._, $=MINI.$, $$=MINI.$$, EE=MINI.EE, HTML=MINI.HTML;

$(function () {
  navigator.getUserMedia = navigator.getUserMedia || navigator.webkitGetUserMedia || navigator.mozGetUserMedia;

  var $ul = $('#messages');
  var ws;
  var peerConnection = new webkitRTCPeerConnection(null);
  var stream;
  var msgQueue = [];

  var isInitiator = false;
  var websocketOpen = false;
  var websocketCreateTime;
  var websocketOpenTime;
  var room = 'abcdef';


  var id = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    var r = Math.random()*16|0, v = c == 'x' ? r : (r&0x3|0x8);
    return v.toString(16);
  });

  var client = {
      id: id
  };

  var getUrlParameter = function(sParam) {
    var sPageURL = window.location.search.substring(1);
    var sURLVariables = sPageURL.split('&');
    for (var i = 0; i < sURLVariables.length; i++)
    {
      var sParameterName = sURLVariables[i].split('=');
      if (sParameterName[0] == sParam)
      {
        return sParameterName[1];
      }
    }
    return undefined;
  };

  var websocketRecv = function(event) {
    window.console.log('websocket event');
    msg = $.parseJSON(event.data);
    if (msg['type'] == 'error') {
      $ul.add(EE('li', "Error:  " + msg['msg']));
      return;
    }
    if (msg['type'] == 'msg') {
      $ul.add(EE('li', "Msg:  " + msg['msg']));
      var contents = $.parseJSON(msg['msg']);
      switch (contents.type) {
        case 'offer':
          window.console.log('offer received.');
          if (!isInitiator) {
            peerConnection.setRemoteDescription(new RTCSessionDescription(contents));
          }
          break;
        case 'answer':
          window.console.log('answer received.');
          if (isInitiator) {
            peerConnection.setRemoteDescription(new RTCSessionDescription(contents));
          }
          break;
        case 'candidate':
          var candidate = new RTCIceCandidate({sdpMLineIndex:contents.label, candidate:contents.candidate});
          pc.addIceCandidate(candidate);
        case 'initiator':
          isInitiator = contents.value;
          window.console.log('intiator: ' + isInitiator);
          break;
      }
      return;
    }
  };

  var createWebSocket = function() {
    websocketOpen = false;
    ws = new WebSocket("ws://baracus.kir.corp.google.com:8080/socket");
    websocketCreateTime = Date.now();
    ws.onmessage = websocketRecv;
    ws.onclose = createWebSocket;
    ws.onopen = function() {
      websocketOpen = true;
      websocketOpenTime = Date.now();
      window.console.log("Time to open websocket: " + (websocketOpenTime - websocketCreateTime));
      ws.send($.toJSON({'type': 'join', 'src': client.id, 'room': room}));
      for (var i in msgQueue) {
        ws.send(msgQueue[i]);
        window.console.log('sending queued message');
      }
      msgQueue = [];
    };
    ws.onerror = function(evt) {
      window.console.log("Websocket error.");
      window.console.log(evt);
    };
  };
  createWebSocket();

  var sendMsg = function(msg) {
    var jsonMsg = $.toJSON({'type': 'msg', 'room': room, 'src': client.id, 'msg': msg});
    if (websocketOpen) {
      ws.send(jsonMsg);
      window.console.log('sending message');
    } else {
      msgQueue.push(jsonMsg);
    }
  };


  var streamAvailable = function(stream) {
    window.console.log('stream available');
    var element = document.querySelector("#local");
    window.console.log(element);
    element.src = URL.createObjectURL(stream);
    peerConnection.addStream(stream);
    peerConnection.createOffer(function(offer) {
      window.console.log('sending offer');
      var offerJson = $.toJSON(offer);
      peerConnection.setLocalDescription(offer);
      sendMsg(offerJson);
    });
    peerConnection.onicecandidate = function(candidate) {
      window.console.log('new ice candidate');
      var candidateJson = $.toJSON(candidate);
      sendMsg(candidateJson);
    };
    peerConnection.onaddstream = function(evt) {
      var element = document.querySelector("#remote");
      element.src = URL.createObjectURL(evt.stream);
    };
  };

  var streamError = function(error) {
    window.console.log("Errors happen: " + error);
  };

  navigator.getUserMedia({ "audio": true, "video": true }, streamAvailable, streamError);

  $('#sendButton').on('click', function(){
    var data = _.trim($('#data').get('value'));
    if (data != '') {
      sendMsg(data);
    }
  });
});
