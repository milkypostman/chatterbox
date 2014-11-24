var Chatterbox = function() {
  this.isInitiator_ = false;
  this.streamInitialized_ = false;
  this.haveOffer_ = false;


  this.pc_ = new RTCPeerConnection(null);
  this.pc_.onsignalingstatechange = this.onSignalingStateChange_.bind(this);
  this.pc_.oniceconnectionstatechange = this.onIceConnectionStateChange_.bind(this);
  this.pc_.onicecandidate = this.onIceCandidate_.bind(this);
  this.pc_.onaddstream = this.onAddStream_.bind(this);
  this.onIceConnectionStateChange_();
  this.onSignalingStateChange_();

  this.socket_ = new CSocket('ws://localhost:8080/socket');
  this.socket_.onRtcMessage = this.onRtcMessage_.bind(this);
  this.socket_.onRtcError = this.onRtcError_.bind(this);

  getUserMedia({ "audio": true, "video": true },
      this.onStreamAvailable_.bind(this),
      this.onStreamError_.bind(this));
};


Chatterbox.prototype.maybeCreateAnswer_ = function() {
  trace('Maybe create answer?');
  if(!this.isInitiator_ && this.streamInitialized_ && this.haveOffer_) {
    this.pc_.createAnswer(this.onAnswerCreated_.bind(this));
  };
};

Chatterbox.prototype.onAnswerCreated_ = function(answer) {
  this.pc_.setLocalDescription(answer);
  this.socket_.send(JSON.stringify(answer));
};

Chatterbox.prototype.onOfferCreated_ = function(offer) {
  this.pc_.setLocalDescription(offer);
  this.socket_.send(JSON.stringify(offer));
};

Chatterbox.prototype.onRtcError_ = function(err) {
};

Chatterbox.prototype.handleOffer_ = function(data) {
  if (!this.isInitiator_) {
    this.pc_.setRemoteDescription(new RTCSessionDescription(data),
        function() { trace('Success adding remote by offer'); },
        function() { trace('Error adding remote by offer'); }
        );
    this.haveOffer_ = true;
    this.maybeCreateAnswer_();
  }
};

Chatterbox.prototype.handleAnswer_ = function(data) {
  if (this.isInitiator_) {
    this.pc_.setRemoteDescription(new RTCSessionDescription(data),
        function() { trace('Success adding remote by answer.'); },
        function() { trace('Error adding remote by answer.'); }
        );
  }
};

Chatterbox.prototype.handleIceCandidate_ = function(data) {
  var candidate = data.candidate;
  if (isDefAndNotNull(candidate)) {
    var iceCandidate = new RTCIceCandidate({
        sdpMLineIndex: candidate.sdpMLineIndex,
        candidate: candidate.candidate
    });
    this.pc_.addIceCandidate(iceCandidate,
        function() {trace('Add ice candidate success.');},
        function() {trace('Add ice candidate failure.');}
        );
  } else {
    trace('No more ice candidates.');
  }
};

Chatterbox.prototype.handleInitiator_ = function(data) {
  if (isDefAndNotNull(data.value)) {
    this.isInitiator_ = !!data.value;
    trace('Intiator status: ' + this.isInitiator_);
  }
};

Chatterbox.prototype.onRtcMessage_ = function(msg) {
  var data = JSON.parse(msg);
  switch (data.type) {
    case 'offer':
      this.handleOffer_(data);
      break;
    case 'answer':
      this.handleAnswer_(data);
      break;
    case 'icecandidate':
      this.handleIceCandidate_(data);
      break;
    case 'initiator':
      this.handleInitiator_(data);
      break;
    default:
      trace('Unknown RTC message: ' + data);
      break;
  };
};

Chatterbox.prototype.onStreamAvailable_ = function(stream) {
  trace('Local video stream available.');
  this.pc_.addStream(stream);
  this.streamInitialized_ = true;
  if (this.isInitiator_) {
    this.pc_.createOffer(this.onOfferCreated_.bind(this));
  } else {
    this.maybeCreateAnswer_();
  }

  if (isDefAndNotNull(this.onStreamAvailable)) {
    this.onStreamAvailable(stream);
  }
};

Chatterbox.prototype.onAddStream_ = function(evt) {
  if (isDefAndNotNull(this.onAddStream)) {
    this.onAddStream(evt.stream);
  }
};

Chatterbox.prototype.onIceCandidate_ = function(candidate) {
  var candidateJson = JSON.stringify({type: candidate.type, candidate: candidate.candidate});
  this.socket_.send(candidateJson);
};

Chatterbox.prototype.onSignalingStateChange_ = function () {
  trace('Signaling state changed: ' + this.pc_.signalingState);
};

Chatterbox.prototype.onIceConnectionStateChange_ = function () {
  trace('ICE Connection state: ' + this.pc_.iceConnectionState);
  // if (this.pc_.iceConnectionState == 'connected') {
  //   trace('join time: ' + (Date.now() - joinTime));
  // };
};

Chatterbox.prototype.onStreamError_ = function(error) {
  trace("Errors happen: " + error);
};

Chatterbox.prototype.onSignalingStateChange_ = function() {
  trace('Signaling state changed: ' + this.pc_.signalingState);
};
