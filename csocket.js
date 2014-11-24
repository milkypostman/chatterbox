// Wrapper around a WebSocket for handling connections to the Chatterbox server.

var CSocket = function(url) {
  this.url_ = url;
  this.msgQueue_ = [];
  this.isOpen_ = false;
  this.connectAttempts_ = 0;
  this.wsOpenTime_ = 0;
  var id = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    var r = Math.random()*16|0, v = c == 'x' ? r : (r&0x3|0x8);
    return v.toString(16);
  });

  this.id_ = id;
  this.room_ = 'abcdef';


  this.openWebsocket_();
};

// Reconnect after 10 seconds.
CSocket.RECONNECT_TIMEOUT = 10 * 1000;

// Try reconnecting 60 times.
CSocket.RECONNECT_ATTEMPTS = 60;

CSocket.prototype.openWebsocket_ = function() {
  if (this.connectAttempts_ < CSocket.RECONNECT_ATTEMPTS) {
    this.wsOpenTime_ = Date.now();
    this.ws_ = new WebSocket(this.url_);
    this.ws_.onopen = this.onOpen_.bind(this);
    this.ws_.onclose = this.onClose_.bind(this);
    this.ws_.onmessage = this.onMessage_.bind(this);
    this.ws_.onerror = this.onError_.bind(this);
    this.connectAttempts_++;
  } else {
    trace('Websocket connection failed after ' + CSocket.RECONNECT_ATTEMPTS + ' attempts.');
  }
};

CSocket.prototype.onError_ = function(err) {
  trace('Websocket error: ' + err);
};

CSocket.prototype.onMessage_ = function(event) {
  var msg = JSON.parse(event.data);
  trace('S->C: ' + event.data);
  if (msg['type'] == 'error') {
    if (isDefAndNotNull(this.onRtcError)) {
      this.onRtcError(contents);
    }
  } else if (msg['type'] == 'msg') {
    var contents = JSON.parse(msg['msg']);
    if (isDefAndNotNull(this.onRtcMessage)) {
      this.onRtcMessage(msg['msg']);
    }
  }
};

CSocket.prototype.onClose_ = function() {
  this.ws_ = undefined;
  this.isOpen = false;
  window.setTimeout(this.openWebsocket_(),
      this.connectAttempts_ == 0 ? 0 : CSocket.RECONNECT_TIMEOUT);
};

CSocket.prototype.onOpen_ = function() {
  trace("Websocket opened in " + (Date.now() - this.wsOpenTime_) + 'ms.');
  this.ws_.send(JSON.stringify({'type': 'join', 'src': this.id_, 'room': this.room_}));
  for (var i in this.msgQueue_) {
    this.ws_.send(this.msgQueue_[i]);
  }
  this.msgQueue_ = [];
  this.isOpen_ = true;
  this.connectAttempts_ = 0;
};


CSocket.prototype.send = function(msg) {
  var jsonMsg = JSON.stringify({'type': 'msg', 'room': this.room_, 'src': this.id_, 'msg': msg});
  if (this.isOpen_) {
    trace('C->S: ' + jsonMsg);
    this.ws_.send(jsonMsg);
  } else {
    this.msgQueue_.push(jsonMsg);
  }
};
