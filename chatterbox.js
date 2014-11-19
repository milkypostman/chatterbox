var MINI = require('minified');
var _=MINI._, $=MINI.$, $$=MINI.$$, EE=MINI.EE, HTML=MINI.HTML;

$(function () {
    navigator.getUserMedia = navigator.getUserMedia || navigator.webkitGetUserMedia || navigator.mozGetUserMedia;

    var $ul = $('#messages');
    var ws;
    var offerJson;
    var peerConnection = new webkitRTCPeerConnection(null);
    var stream;

    // var remoteData(event) {
    //     offer = $.parseJSON(event.data)
    //     if (!offer.sdp) {
    //         window.console.log("offer contained no sdp data");
    //         return
    //     }
    //     peerConnection.setRemoteDescription(new RTCSessionDescription(offer.sdp))
    // }

    var createWebSocket = function() {
        ws = new WebSocket("ws://localhost:8080/socket");
        ws.onmessage = function(event) {
            window.console.log(event);
            msg = $.parseJSON(event.data)
            if (msg['Id'] != undefined && msg['Id'] != 0) {
                $ul.add(EE('li', "My ID is " + msg['Id']))
                return
            }
            if (msg['ClientAdd'] != undefined && msg['ClientAdd'] != 0) {
                $ul.add(EE('li', "New Client is " + msg['ClientAdd']))
            }
            if (msg['ClientDrop'] != undefined && msg['ClientDrop'] != 0) {
                $ul.add(EE('li', "New Client is " + msg['ClientDrop']))
            }
        };
        ws.onclose = createWebSocket;
        ws.onopen = function() {
            // if (offerJson) {
            //     ws.send(offerJson)
            // }
        }
    };
    createWebSocket();

    var streamAvailable = function(s) {
        window.console.log(s);
        stream = s;
        // peerConnection.addStream(stream);
        // peerConnection.createOffer(function(offer) {
        //     window.console.log(offer);
        //     window.console.log($.toJSON(offer));
        //     // peerConnection.setLocalDescription(offer);
        //     // offerJson = $.toJSON(offer);
        //     // ws.send(offerJson);
        // });
    }

    var streamError = function(error) {
        window.console.log("Errors happen: " + error);
    }

    navigator.getUserMedia({ "audio": true, "video": false }, streamAvailable, streamError);

    $('#sendButton').on('click', function(){
        var data = _.trim($('#data').get('value'));
        if (data != '') {
            ws.send(data);
            window.console.log(data);
        }
    });
});
