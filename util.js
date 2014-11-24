// Random global utility functions.

navigator.getUserMedia = navigator.getUserMedia || navigator.webkitGetUserMedia || navigator.mozGetUserMedia;

var isDefAndNotNull = function(val) {
  return val != undefined && val != null;
};
